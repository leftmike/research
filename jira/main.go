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
	fmt.Fprintf(os.Stderr, "  list PROJECT --created DURATION | --updated DURATION\n")
	fmt.Fprintf(os.Stderr, "      List tickets in a project by recency. DURATION examples: 24h, 7d, 2w.\n")
	fmt.Fprintf(os.Stderr, "      --created and --updated may be combined (tickets matching either are shown).\n\n")
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

func cmdList(client *jira.Client, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	created := fs.String("created", "", "show tickets created within this duration (e.g. 24h, 7d, 2w)")
	updated := fs.String("updated", "", "show tickets updated within this duration (e.g. 24h, 7d, 2w)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: jira list PROJECT [--created DURATION] [--updated DURATION]\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}
	if *created == "" && *updated == "" {
		fmt.Fprintf(os.Stderr, "error: at least one of --created or --updated is required\n")
		fs.Usage()
		os.Exit(1)
	}

	project := fs.Arg(0)
	jql, err := buildListJQL(project, *created, *updated)
	if err != nil {
		log.Fatalf("invalid duration: %v", err)
	}

	issues, _, err := client.Issue.Search(context.Background(), jql, &jira.SearchOptions{
		MaxResults: 50,
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	if len(issues) == 0 {
		fmt.Println("No tickets found.")
		return
	}
	printIssueList(issues)
}

// buildListJQL constructs a JQL query for the list command.
// If both created and updated are given, tickets matching either are returned.
func buildListJQL(project, created, updated string) (string, error) {
	jql := fmt.Sprintf("project = %q", project)

	var conditions []string
	if created != "" {
		d, err := parseDuration(created)
		if err != nil {
			return "", fmt.Errorf("--created: %w", err)
		}
		since := time.Now().Add(-d).UTC().Format("2006-01-02 15:04")
		conditions = append(conditions, fmt.Sprintf(`created >= "%s"`, since))
	}
	if updated != "" {
		d, err := parseDuration(updated)
		if err != nil {
			return "", fmt.Errorf("--updated: %w", err)
		}
		since := time.Now().Add(-d).UTC().Format("2006-01-02 15:04")
		conditions = append(conditions, fmt.Sprintf(`updated >= "%s"`, since))
	}

	if len(conditions) == 1 {
		jql += " AND " + conditions[0]
	} else {
		jql += " AND (" + strings.Join(conditions, " OR ") + ")"
	}
	jql += " ORDER BY updated DESC"
	return jql, nil
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
		fmt.Printf("Created:   %s\n", created.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	if updated := time.Time(f.Updated); !updated.IsZero() {
		fmt.Printf("Updated:   %s\n", updated.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	if f.Description != "" {
		fmt.Printf("\nDescription:\n")
		for _, line := range strings.Split(f.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
}

func printIssueList(issues []jira.Issue) {
	fmt.Printf("%-14s  %-20s  %-10s  %s\n", "KEY", "STATUS", "UPDATED", "TITLE")
	fmt.Printf("%-14s  %-20s  %-10s  %s\n", strings.Repeat("-", 14), strings.Repeat("-", 20), strings.Repeat("-", 10), strings.Repeat("-", 40))
	for _, issue := range issues {
		f := issue.Fields
		status := ""
		if f.Status != nil {
			status = f.Status.Name
		}
		updated := ""
		if u := time.Time(f.Updated); !u.IsZero() {
			updated = u.UTC().Format("2006-01-02")
		}
		fmt.Printf("%-14s  %-20s  %-10s  %s\n", issue.Key, status, updated, f.Summary)
	}
}
