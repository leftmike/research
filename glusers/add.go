package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

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

func parseExpires(input string, now time.Time) (string, error) {
	if input == "" {
		return "", nil
	}
	last := input[len(input)-1]
	if last == 'd' || last == 'w' {
		n, err := strconv.Atoi(input[:len(input)-1])
		if err != nil || n <= 0 {
			return "", fmt.Errorf("invalid expires value %q: expected positive number with d or w suffix", input)
		}
		days := n
		if last == 'w' {
			days = n * 7
		}
		t := now.AddDate(0, 0, days)
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

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	urlFlag := fs.String("url", "", "GitLab base URL (or $GITLAB_URL)")
	tokenFlag := fs.String("token", "", "personal access token (or $GITLAB_TOKEN)")
	insecureFlag := fs.Bool("insecure", false, "skip TLS certificate verification")
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
		parsedExpires, err := parseExpires(expires, time.Now())
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

	fmt.Printf("Added %s (id=%d) as %s\n", member.Username, member.ID, accessLevelName(member.AccessLevel))
}
