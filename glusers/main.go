package main

import (
	"crypto/tls"
	"encoding/json"
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

// memberJSON is the JSON representation of a project member.
type memberJSON struct {
	Username    string  `json:"username"`
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	AccessLevel string  `json:"access_level"`
	Expires     *string `json:"expires"`
}

func memberToJSON(m *gitlab.ProjectMember) memberJSON {
	j := memberJSON{
		Username:    m.Username,
		ID:          m.ID,
		Name:        m.Name,
		AccessLevel: accessLevelName(m.AccessLevel),
	}
	if m.ExpiresAt != nil {
		s := m.ExpiresAt.String()
		j.Expires = &s
	}
	return j
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Fatalf("json encode: %v", err)
	}
}

func cmdList(gl *gitlab.Client, project string, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output as JSON array")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: glusers list [-json]")
		fs.PrintDefaults()
	}
	fs.Parse(args) //nolint:errcheck

	var members []*gitlab.ProjectMember
	opts := &gitlab.ListProjectMembersOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	for {
		page, resp, err := gl.ProjectMembers.ListAllProjectMembers(project, opts)
		if err != nil {
			log.Fatalf("list members: %v", err)
		}
		members = append(members, page...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if *jsonOut {
		out := make([]memberJSON, len(members))
		for i, m := range members {
			out[i] = memberToJSON(m)
		}
		printJSON(out)
		return
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
	jsonOut := fs.Bool("json", false, "output result as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: glusers add <username> -level <level> [-expires YYYY-MM-DD] [-json]")
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

	if *jsonOut {
		printJSON(memberToJSON(member))
		return
	}
	fmt.Printf("Added %s (id=%d) as %s\n", member.Username, member.ID, accessLevelName(member.AccessLevel))
}

func cmdRemove(gl *gitlab.Client, project string, args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output result as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: glusers remove [-json] <username>")
		fs.PrintDefaults()
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

	if *jsonOut {
		printJSON(map[string]any{"removed": true, "username": username})
		return
	}
	fmt.Printf("Removed %s from project\n", username)
}

// helpFlag describes a single flag for the help command output.
type helpFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// helpArg describes a positional argument.
type helpArg struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// helpCommand describes a subcommand.
type helpCommand struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Flags       []helpFlag `json:"flags"`
	Args        []helpArg  `json:"args"`
	Example     string     `json:"example"`
}

// helpDoc is the full machine-readable help document.
type helpDoc struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	GlobalFlags []helpFlag    `json:"global_flags"`
	Commands    []helpCommand `json:"commands"`
}

func cmdHelp() {
	doc := helpDoc{
		Name:        "glusers",
		Description: "Manage GitLab project membership: list, add, and remove members.",
		GlobalFlags: []helpFlag{
			{Name: "url", Type: "string", Required: true, Description: "GitLab base URL (e.g. https://gitlab.example.com). Also read from $GITLAB_URL."},
			{Name: "token", Type: "string", Required: true, Description: "GitLab personal access token with api scope. Also read from $GITLAB_TOKEN."},
			{Name: "project", Type: "string", Required: true, Description: "Project path (e.g. mygroup/myrepo) or numeric project ID. Also read from $GITLAB_PROJECT."},
			{Name: "insecure", Type: "bool", Required: false, Default: "false", Description: "Skip TLS certificate verification. Use for self-hosted instances with self-signed certificates."},
		},
		Commands: []helpCommand{
			{
				Name:        "list",
				Description: "List all direct members of the project with their access levels and expiry dates.",
				Flags: []helpFlag{
					{Name: "json", Type: "bool", Required: false, Default: "false", Description: "Output members as a JSON array instead of a human-readable table."},
				},
				Args:    []helpArg{},
				Example: "glusers -project mygroup/myrepo list -json",
			},
			{
				Name:        "add",
				Description: "Add a GitLab user to the project with the specified access level.",
				Flags: []helpFlag{
					{Name: "level", Type: "string", Required: true, Description: "Access level to grant. One of: guest, reporter, developer, maintainer, owner."},
					{Name: "expires", Type: "string", Required: false, Description: "Optional membership expiration date in YYYY-MM-DD format. Omit for no expiry."},
					{Name: "json", Type: "bool", Required: false, Default: "false", Description: "Output the newly added member as a JSON object instead of a human-readable message."},
				},
				Args: []helpArg{
					{Name: "username", Required: true, Description: "GitLab username of the user to add."},
				},
				Example: "glusers -project mygroup/myrepo add alice -level developer -expires 2026-12-31 -json",
			},
			{
				Name:        "remove",
				Description: "Remove a GitLab user from the project.",
				Flags: []helpFlag{
					{Name: "json", Type: "bool", Required: false, Default: "false", Description: "Output the result as a JSON object instead of a human-readable message."},
				},
				Args: []helpArg{
					{Name: "username", Required: true, Description: "GitLab username of the user to remove."},
				},
				Example: "glusers -project mygroup/myrepo remove alice -json",
			},
			{
				Name:        "help",
				Description: "Print this machine-readable help document as JSON. Does not require -url, -token, or -project.",
				Flags:       []helpFlag{},
				Args:        []helpArg{},
				Example:     "glusers help",
			},
		},
	}
	printJSON(doc)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: glusers [flags] <subcommand> [subcommand flags] [args]

Global flags:
  -url string      GitLab base URL (or $GITLAB_URL)
  -token string    Personal access token (or $GITLAB_TOKEN)
  -project string  Project path or ID (or $GITLAB_PROJECT)
  -insecure        Skip TLS certificate verification

Subcommands:
  list [-json]                              List project members
  add <username> -level <level> [-json]     Add a member (levels: guest|reporter|developer|maintainer|owner)
                 [-expires DATE]            Optional expiration date (YYYY-MM-DD)
  remove [-json] <username>                 Remove a member
  help                                      Print machine-readable help as JSON`)
}

func main() {
	urlFlag := flag.String("url", "", "GitLab base URL")
	tokenFlag := flag.String("token", "", "personal access token")
	projectFlag := flag.String("project", "", "project path or numeric ID")
	insecureFlag := flag.Bool("insecure", false, "skip TLS certificate verification")

	flag.Usage = usage
	flag.Parse()

	// help doesn't need credentials — handle it before validation.
	if flag.NArg() > 0 && flag.Arg(0) == "help" {
		cmdHelp()
		return
	}

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
