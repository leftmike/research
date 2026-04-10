package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

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

func parseExpires(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	lower := strings.ToLower(input)
	last := lower[len(lower)-1]
	if last == 'd' || last == 'w' {
		n, err := strconv.Atoi(lower[:len(lower)-1])
		if err != nil || n <= 0 {
			return "", fmt.Errorf("invalid expires value %q: expected positive number with d or w suffix", input)
		}
		days := n
		if last == 'w' {
			days = n * 7
		}
		t := time.Now().AddDate(0, 0, days)
		return t.Format("2006-01-02"), nil
	}

	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"01-02-2006",
		"Jan 2, 2006",
		"2 Jan 2006",
		"2 Jan 06",
		"02 Jan 06",
		"02 Jan 2006",
		"02-Jan-06",
		"02-Jan-2006",
	}
	candidates := []string{
		input,
		strings.Title(strings.ToLower(input)),
	}
	for _, candidate := range candidates {
		for _, layout := range layouts {
			if t, err := time.Parse(layout, candidate); err == nil {
				return t.Format("2006-01-02"), nil
			}
		}
	}
	return "", fmt.Errorf("invalid expires value %q: expected YYYY-MM-DD, YYYY/MM/DD, MM/DD/YYYY, MM-DD-YYYY, Jan 2, 2006, 2 Jan 2006, 2 Jan 06, 02 Jan 06, 02 Jan 2006, 02-Jan-06, 02-Jan-2006, or Nd/Nw", input)
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

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	urlFlag := fs.String("url", "", "GitLab base URL (or $GITLAB_URL)")
	tokenFlag := fs.String("token", "", "personal access token (or $GITLAB_TOKEN)")
	insecureFlag := fs.Bool("insecure", false, "skip TLS certificate verification")
	jsonOut := fs.Bool("json", false, "output as JSON array")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: glusers list [flags] <project>")
		fs.PrintDefaults()
	}
	fs.Parse(args) //nolint:errcheck

	if *urlFlag == "" {
		*urlFlag = os.Getenv("GITLAB_URL")
	}
	if *tokenFlag == "" {
		*tokenFlag = os.Getenv("GITLAB_TOKEN")
	}
	if *urlFlag == "" || *tokenFlag == "" {
		fmt.Fprintln(os.Stderr, "error: -url and -token are required (or set GITLAB_URL, GITLAB_TOKEN)")
		fs.Usage()
		os.Exit(1)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}
	project := fs.Arg(0)

	gl, err := newGitLabClient(*urlFlag, *tokenFlag, *insecureFlag)
	if err != nil {
		log.Fatalf("create gitlab client: %v", err)
	}

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

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	urlFlag := fs.String("url", "", "GitLab base URL (or $GITLAB_URL)")
	tokenFlag := fs.String("token", "", "personal access token (or $GITLAB_TOKEN)")
	insecureFlag := fs.Bool("insecure", false, "skip TLS certificate verification")
	jsonOut := fs.Bool("json", false, "output result as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: glusers add [flags] <project> <username> <level> [expires]")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "expires: YYYY-MM-DD, YYYY/MM/DD, MM/DD/YYYY, MM-DD-YYYY, Jan 2, 2006, 2 Jan 2006, 2 Jan 06, 02 Jan 06, 02 Jan 2006, 02-Jan-06, 02-Jan-2006, or relative like 3d or 6w")
	}
	fs.Parse(args) //nolint:errcheck

	if *urlFlag == "" {
		*urlFlag = os.Getenv("GITLAB_URL")
	}
	if *tokenFlag == "" {
		*tokenFlag = os.Getenv("GITLAB_TOKEN")
	}
	if *urlFlag == "" || *tokenFlag == "" {
		fmt.Fprintln(os.Stderr, "error: -url and -token are required (or set GITLAB_URL, GITLAB_TOKEN)")
		fs.Usage()
		os.Exit(1)
	}

	gl, err := newGitLabClient(*urlFlag, *tokenFlag, *insecureFlag)
	if err != nil {
		log.Fatalf("create gitlab client: %v", err)
	}

	if fs.NArg() < 3 || fs.NArg() > 4 {
		fs.Usage()
		os.Exit(1)
	}
	project := fs.Arg(0)
	username := fs.Arg(1)
	level := fs.Arg(2)
	var expires string
	if fs.NArg() == 4 {
		expires = fs.Arg(3)
	}

	accessLevel, err := parseAccessLevel(level)
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
	if expires != "" {
		parsedExpires, err := parseExpires(expires)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		opts.ExpiresAt = &parsedExpires
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

func usage() {
	fmt.Fprintln(os.Stderr, `usage: glusers <command> [flags] [args]

Commands:
  list    List project members (requires <project>)
  add     Add a member (requires <project> <username> <level> [expires])

Run "glusers <command> -h" for command-specific flags.`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	subcommand := os.Args[1]
	rest := os.Args[2:]

	switch subcommand {
	case "list":
		cmdList(rest)
	case "add":
		cmdAdd(rest)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", subcommand)
		usage()
		os.Exit(1)
	}
}
