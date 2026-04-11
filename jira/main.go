package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/onpremise"
)

var (
	jiraURL    = flag.String("url", "", "Jira base URL (e.g. https://jira.mycompany.com)")
	jiraToken  = flag.String("token", "", "Jira personal access token (sent as Bearer)")
)

// patTransport adds a Bearer token header for PAT-based authentication.
type patTransport struct {
	token string
}

func (t *patTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req2)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: jira [global flags] <command> [args]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  get TICKET-ID\n")
	fmt.Fprintf(os.Stderr, "      Fetch and display a single ticket.\n\n")
	fmt.Fprintf(os.Stderr, "  created PROJECT SINCE\n")
	fmt.Fprintf(os.Stderr, "      List tickets in a project created since a duration ago. Examples: 24h, 7d, 2w.\n\n")
	fmt.Fprintf(os.Stderr, "  updated PROJECT SINCE\n")
	fmt.Fprintf(os.Stderr, "      List tickets in a project updated since a duration ago. Examples: 24h, 7d, 2w.\n\n")
	fmt.Fprintf(os.Stderr, "  list -updated PROJECT SINCE\n")
	fmt.Fprintf(os.Stderr, "      Alias for 'updated'.\n\n")
	fmt.Fprintf(os.Stderr, "  help\n")
	fmt.Fprintf(os.Stderr, "      Show this help.\n\n")
	fmt.Fprintf(os.Stderr, "Global flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nCredentials may also be set via environment variables: JIRA_URL, JIRA_TOKEN\n")
}

func main() {
	flag.Usage = printUsage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// help requires no credentials.
	if args[0] == "help" {
		printUsage()
		return
	}

	url := *jiraURL
	if url == "" {
		url = os.Getenv("JIRA_URL")
	}
	token := *jiraToken
	if token == "" {
		token = os.Getenv("JIRA_TOKEN")
	}

	if url == "" || token == "" {
		fmt.Fprintf(os.Stderr, "error: --url and --token are required (or set JIRA_URL, JIRA_TOKEN)\n")
		os.Exit(1)
	}

	httpClient := &http.Client{Transport: &patTransport{token: token}}

	client, err := jira.NewClient(url, httpClient)
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	switch args[0] {
	case "get":
		cmdGet(client, args[1:])
	case "created":
		cmdCreated(client, args[1:])
	case "updated":
		cmdUpdated(client, args[1:])
	case "list":
		cmdList(client, args[1:])
	default:
		// Bare ticket ID: treat as "get" for backward compatibility.
		cmdGet(client, args)
	}
}

func cmdGet(client *jira.Client, args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jira get TICKET-ID\n")
	}
	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}

	issueKey := fs.Arg(0)
	issue, _, err := client.Issue.Get(context.Background(), issueKey, nil)
	if err != nil {
		log.Fatalf("get issue %s: %v", issueKey, err)
	}

	printIssue(issue)
}

func cmdCreated(client *jira.Client, args []string) {
	fs := flag.NewFlagSet("created", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jira created PROJECT SINCE\n")
	}
	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(1)
	}

	project := fs.Arg(0)
	since, err := parseDuration(fs.Arg(1))
	if err != nil {
		log.Fatalf("invalid since duration: %v", err)
	}

	jql := buildSinceJQL(project, "created", since, true)
	issues, err := searchAllIssues(context.Background(), client, jql)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	if len(issues) == 0 {
		fmt.Println("No tickets found.")
		return
	}
	printIssueList(issues, "created")
}

func cmdUpdated(client *jira.Client, args []string) {
	fs := flag.NewFlagSet("updated", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jira updated PROJECT SINCE\n")
	}
	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(1)
	}

	project := fs.Arg(0)
	since, err := parseDuration(fs.Arg(1))
	if err != nil {
		log.Fatalf("invalid since duration: %v", err)
	}

	jql := buildSinceJQL(project, "updated", since, true)
	issues, err := searchAllIssues(context.Background(), client, jql)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	if len(issues) == 0 {
		fmt.Println("No tickets found.")
		return
	}
	printIssueList(issues, "updated")
}

func cmdList(client *jira.Client, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	updated := fs.Bool("updated", false, "list tickets updated since a duration ago (alias for 'updated')")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jira list -updated PROJECT SINCE\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if !*updated {
		fs.Usage()
		os.Exit(1)
	}

	cmdUpdated(client, fs.Args())
}

func searchAllIssues(ctx context.Context, client *jira.Client, jql string) ([]jira.Issue, error) {
	const pageSize = 200
	opts := &jira.SearchOptions{StartAt: 0, MaxResults: pageSize}
	issues := make([]jira.Issue, 0, pageSize)
	err := client.Issue.SearchPages(ctx, jql, opts, func(issue jira.Issue) error {
		issues = append(issues, issue)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func buildSinceJQL(project, field string, since time.Duration, ascending bool) string {
	jql := fmt.Sprintf("project = %q", project)
	sinceTS := time.Now().Add(-since).UTC().Format("2006-01-02 15:04")
	jql += fmt.Sprintf(` AND %s >= "%s"`, field, sinceTS)
	order := "DESC"
	if ascending {
		order = "ASC"
	}
	jql += fmt.Sprintf(" ORDER BY %s %s", field, order)
	return jql
}

// parseDuration extends time.ParseDuration with support for d (days) and w (weeks).
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasSuffix(s, "d"):
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "d"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	case strings.HasSuffix(s, "w"):
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "w"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(n * float64(7 * 24 * time.Hour)), nil
	default:
		return time.ParseDuration(s)
	}
}

func printIssue(issue *jira.Issue) {
	f := issue.Fields
	fmt.Printf("Key:       %s\n", issue.Key)
	fmt.Printf("Summary:   %s\n", f.Summary)
	if f.Type.Name != "" {
		fmt.Printf("Type:      %s\n", f.Type.Name)
	}
	if f.Status != nil {
		fmt.Printf("Status:    %s\n", f.Status.Name)
	}
	if f.Priority != nil {
		fmt.Printf("Priority:  %s\n", f.Priority.Name)
	}
	if f.Assignee != nil {
		fmt.Printf("Assignee:  %s\n", f.Assignee.DisplayName)
	}
	if f.Reporter != nil {
		fmt.Printf("Reporter:  %s\n", f.Reporter.DisplayName)
	}
	if created := time.Time(f.Created); !created.IsZero() {
		fmt.Printf("Created:   %s\n", created.UTC().Format("01-02-2006"))
	}
	if updated := time.Time(f.Updated); !updated.IsZero() {
		fmt.Printf("Updated:   %s\n", updated.UTC().Format("01-02-2006"))
	}
	if f.Description != "" {
		fmt.Printf("\nDescription:\n")
		for _, line := range strings.Split(f.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
}

func printIssueList(issues []jira.Issue, dateField string) {
	const (
		lineWidth   = 100
		keyWidth    = 10
		statusWidth = 12
		dateWidth   = 10
	)

	dateHeader := strings.ToUpper(dateField)
	fmt.Printf("%-*s  %-*s  %-*s  %s\n", keyWidth, "KEY", statusWidth, "STATUS", dateWidth, dateHeader, "TITLE")
	fmt.Printf("%-*s  %-*s  %-*s  %s\n", keyWidth, strings.Repeat("-", keyWidth), statusWidth, strings.Repeat("-", statusWidth), dateWidth, strings.Repeat("-", dateWidth), strings.Repeat("-", 40))

	titleWidth := lineWidth - (keyWidth + 2 + statusWidth + 2 + dateWidth + 2)
	if titleWidth < 4 {
		titleWidth = 4
	}

	for _, issue := range issues {
		f := issue.Fields
		status := ""
		if f.Status != nil {
			status = f.Status.Name
		}
		if rs := []rune(status); len(rs) > 12 {
			status = string(rs[:12])
		}

		dateValue := ""
		switch dateField {
		case "created":
			if t := time.Time(f.Created); !t.IsZero() {
				dateValue = t.UTC().Format("01-02-2006")
			}
		case "updated":
			if t := time.Time(f.Updated); !t.IsZero() {
				dateValue = t.UTC().Format("01-02-2006")
			}
		default:
			log.Fatalf("unknown date field %q", dateField)
		}

		title := strings.Join(strings.Fields(f.Summary), " ")
		title = truncateWithEllipsis(title, titleWidth)
		fmt.Printf("%-*s  %-*s  %-*s  %s\n", keyWidth, issue.Key, statusWidth, status, dateWidth, dateValue, title)
	}
}

func truncateWithEllipsis(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(rs[:maxRunes])
	}
	return string(rs[:maxRunes-3]) + "..."
}
