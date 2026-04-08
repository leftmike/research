package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/tabwriter"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func newGitLabClient(baseURL, token string, insecure bool) (*gitlab.Client, error) {
	opts := []gitlab.ClientOptionFunc{
		gitlab.WithBaseURL(baseURL),
	}
	if insecure {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
		opts = append(opts, gitlab.WithHTTPClient(httpClient))
	}
	return gitlab.NewClient(token, opts...)
}

func resolveUserID(gl *gitlab.Client, username string) (int64, error) {
	users, _, err := gl.Users.ListUsers(&gitlab.ListUsersOptions{
		Username: gitlab.Ptr(username),
	})
	if err != nil {
		return 0, fmt.Errorf("look up user %q: %w", username, err)
	}
	if len(users) == 0 {
		return 0, fmt.Errorf("user %q not found", username)
	}
	if len(users) > 1 {
		return 0, fmt.Errorf("user %q matched %d users", username, len(users))
	}
	return users[0].ID, nil
}

func parseAccessLevel(s string) (gitlab.AccessLevelValue, error) {
	switch s {
	case "guest":
		return gitlab.GuestPermissions, nil
	case "reporter":
		return gitlab.ReporterPermissions, nil
	case "developer":
		return gitlab.DeveloperPermissions, nil
	case "maintainer":
		return gitlab.MaintainerPermissions, nil
	case "owner":
		return gitlab.OwnerPermissions, nil
	default:
		return 0, fmt.Errorf("unknown access level %q: must be guest|reporter|developer|maintainer|owner", s)
	}
}

func accessLevelName(v gitlab.AccessLevelValue) string {
	switch v {
	case gitlab.GuestPermissions:
		return "Guest"
	case gitlab.ReporterPermissions:
		return "Reporter"
	case gitlab.DeveloperPermissions:
		return "Developer"
	case gitlab.MaintainerPermissions:
		return "Maintainer"
	case gitlab.OwnerPermissions:
		return "Owner"
	default:
		return fmt.Sprintf("Level(%d)", v)
	}
}

func cmdList(gl *gitlab.Client, project string, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gltool list")
	}
	fs.Parse(args) //nolint:errcheck

	members, _, err := gl.ProjectMembers.ListAllProjectMembers(project, nil)
	if err != nil {
		log.Fatalf("list members: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tID\tACCESS LEVEL\tEXPIRES")
	for _, m := range members {
		expires := "-"
		if m.ExpiresAt != nil {
			expires = m.ExpiresAt.String()
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", m.Username, m.ID, accessLevelName(m.AccessLevel), expires)
	}
	w.Flush()
}

func cmdAdd(gl *gitlab.Client, project string, args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	level := fs.String("level", "", "access level: guest|reporter|developer|maintainer|owner (required)")
	expires := fs.String("expires", "", "expiration date in YYYY-MM-DD format (optional)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gltool add <username> -level <level> [-expires YYYY-MM-DD]")
		fs.PrintDefaults()
	}
	fs.Parse(args) //nolint:errcheck

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}
	username := fs.Arg(0)

	if *level == "" {
		fmt.Fprintln(os.Stderr, "error: -level is required")
		fs.Usage()
		os.Exit(1)
	}

	accessLevel, err := parseAccessLevel(*level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	userID, err := resolveUserID(gl, username)
	if err != nil {
		log.Fatalf("resolve user: %v", err)
	}

	opts := &gitlab.AddProjectMemberOptions{
		UserID:      gitlab.Ptr(userID),
		AccessLevel: gitlab.Ptr(accessLevel),
	}
	if *expires != "" {
		opts.ExpiresAt = expires
	}

	member, _, err := gl.ProjectMembers.AddProjectMember(project, opts)
	if err != nil {
		log.Fatalf("add member: %v", err)
	}
	fmt.Printf("Added %s (id=%d) as %s\n", member.Username, member.ID, accessLevelName(member.AccessLevel))
}

func cmdRemove(gl *gitlab.Client, project string, args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gltool remove <username>")
	}
	fs.Parse(args) //nolint:errcheck

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}
	username := fs.Arg(0)

	userID, err := resolveUserID(gl, username)
	if err != nil {
		log.Fatalf("resolve user: %v", err)
	}

	_, err = gl.ProjectMembers.DeleteProjectMember(project, userID, nil)
	if err != nil {
		log.Fatalf("remove member: %v", err)
	}
	fmt.Printf("Removed %s from project\n", username)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: gltool [flags] <subcommand> [subcommand flags] [args]

Global flags:
  -url string      GitLab base URL (or $GITLAB_URL)
  -token string    Personal access token (or $GITLAB_TOKEN)
  -project string  Project path or ID (or $GITLAB_PROJECT)
  -insecure        Skip TLS certificate verification

Subcommands:
  list                              List project members
  add <username> -level <level>     Add a member (levels: guest|reporter|developer|maintainer|owner)
                 [-expires DATE]    Optional expiration date (YYYY-MM-DD)
  remove <username>                 Remove a member`)
}

func main() {
	urlFlag := flag.String("url", "", "GitLab base URL")
	tokenFlag := flag.String("token", "", "personal access token")
	projectFlag := flag.String("project", "", "project path or numeric ID")
	insecureFlag := flag.Bool("insecure", false, "skip TLS certificate verification")

	flag.Usage = usage
	flag.Parse()

	if *urlFlag == "" {
		*urlFlag = os.Getenv("GITLAB_URL")
	}
	if *tokenFlag == "" {
		*tokenFlag = os.Getenv("GITLAB_TOKEN")
	}
	if *projectFlag == "" {
		*projectFlag = os.Getenv("GITLAB_PROJECT")
	}

	if *urlFlag == "" || *tokenFlag == "" || *projectFlag == "" {
		fmt.Fprintln(os.Stderr, "error: -url, -token, and -project are required (or set GITLAB_URL, GITLAB_TOKEN, GITLAB_PROJECT)")
		usage()
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		usage()
		os.Exit(1)
	}

	gl, err := newGitLabClient(*urlFlag, *tokenFlag, *insecureFlag)
	if err != nil {
		log.Fatalf("create gitlab client: %v", err)
	}

	subcommand := flag.Arg(0)
	rest := flag.Args()[1:]

	switch subcommand {
	case "list":
		cmdList(gl, *projectFlag, rest)
	case "add":
		cmdAdd(gl, *projectFlag, rest)
	case "remove":
		cmdRemove(gl, *projectFlag, rest)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q\n", subcommand)
		usage()
		os.Exit(1)
	}
}
