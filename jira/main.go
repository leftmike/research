package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
)

var (
	jiraURL    = flag.String("url", "", "Jira base URL (e.g. https://mycompany.atlassian.net)")
	jiraUser   = flag.String("user", "", "Jira username (email); omit to use PAT/Bearer auth")
	jiraToken  = flag.String("token", "", "Jira API token (Basic Auth) or PAT (Bearer)")
	jsonOutput = flag.Bool("json", false, "Output as JSON")
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

// IssueOutput is the JSON-serializable representation of a Jira issue.
type IssueOutput struct {
	Key         string `json:"key"`
	Summary     string `json:"summary"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Assignee    string `json:"assignee,omitempty"`
	Reporter    string `json:"reporter,omitempty"`
	Created     string `json:"created,omitempty"`
	Updated     string `json:"updated,omitempty"`
	Description string `json:"description,omitempty"`
}

func issueToOutput(issue *jira.Issue) IssueOutput {
	f := issue.Fields
	out := IssueOutput{
		Key:     issue.Key,
		Summary: f.Summary,
		Type:    f.Type.Name,
	}
	if f.Status != nil {
		out.Status = f.Status.Name
	}
	if f.Priority != nil {
		out.Priority = f.Priority.Name
	}
	if f.Assignee != nil {
		out.Assignee = f.Assignee.DisplayName
	}
	if f.Reporter != nil {
		out.Reporter = f.Reporter.DisplayName
	}
	if created := time.Time(f.Created); !created.IsZero() {
		out.Created = created.UTC().Format("2006-01-02 15:04:05 UTC")
	}
	if updated := time.Time(f.Updated); !updated.IsZero() {
		out.Updated = updated.UTC().Format("2006-01-02 15:04:05 UTC")
	}
	out.Description = f.Description
	return out
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
	fmt.Fprintf(os.Stderr, "      Print machine-readable JSON documentation (useful for agents).\n\n")
	fmt.Fprintf(os.Stderr, "Global flags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nCredentials may also be set via environment variables: JIRA_URL, JIRA_USER, JIRA_TOKEN\n")
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
		cmdHelp()
		return
	}

	url := *jiraURL
	if url == "" {
		url = os.Getenv("JIRA_URL")
	}
	user := *jiraUser
	if user == "" {
		user = os.Getenv("JIRA_USER")
	}
	token := *jiraToken
	if token == "" {
		token = os.Getenv("JIRA_TOKEN")
	}

	if url == "" || token == "" {
		fmt.Fprintf(os.Stderr, "error: --url and --token are required (or set JIRA_URL, JIRA_TOKEN)\n")
		os.Exit(1)
	}

	var httpClient *http.Client
	if user != "" {
		tp := jira.BasicAuthTransport{Username: user, APIToken: token}
		httpClient = tp.Client()
	} else {
		httpClient = &http.Client{Transport: &patTransport{token: token}}
	}

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

	if *jsonOutput {
		printJSON(issueToOutput(issue))
	} else {
		printIssue(issue)
	}
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

	if *jsonOutput {
		out := make([]IssueOutput, len(issues))
		for i, issue := range issues {
			out[i] = issueToOutput(&issue)
		}
		printJSON(out)
	} else {
		if len(issues) == 0 {
			fmt.Println("No tickets found.")
			return
		}
		printIssueList(issues)
	}
}

// cmdHelp prints machine-readable JSON documentation for use by agents.
func cmdHelp() {
	type flagDoc struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Required    bool   `json:"required,omitempty"`
	}
	type argDoc struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	}
	type commandDoc struct {
		Name        string    `json:"name"`
		Syntax      string    `json:"syntax"`
		Description string    `json:"description"`
		Args        []argDoc  `json:"args,omitempty"`
		Flags       []flagDoc `json:"flags,omitempty"`
	}
	type helpDoc struct {
		Tool        string       `json:"tool"`
		Description string       `json:"description"`
		GlobalFlags []flagDoc    `json:"global_flags"`
		Commands    []commandDoc `json:"commands"`
		Notes       []string     `json:"notes"`
	}

	doc := helpDoc{
		Tool:        "jira",
		Description: "CLI for fetching and listing Jira tickets. Supports Basic Auth (email + API token) and PAT (Bearer) authentication.",
		GlobalFlags: []flagDoc{
			{Name: "--url", Type: "string", Description: "Jira base URL, e.g. https://mycompany.atlassian.net. Overrides JIRA_URL env var.", Required: true},
			{Name: "--user", Type: "string", Description: "Jira username/email for Basic Auth. Omit to use PAT/Bearer auth. Overrides JIRA_USER env var."},
			{Name: "--token", Type: "string", Description: "API token (with --user) or PAT (without --user). Overrides JIRA_TOKEN env var.", Required: true},
			{Name: "--json", Type: "bool", Description: "Emit JSON instead of human-readable text. Applies to get and list."},
		},
		Commands: []commandDoc{
			{
				Name:        "get",
				Syntax:      "jira [global-flags] get TICKET-ID",
				Description: "Fetch a single Jira ticket and display its fields.",
				Args: []argDoc{
					{Name: "TICKET-ID", Description: "Jira issue key, e.g. PROJ-123.", Required: true},
				},
			},
			{
				Name:        "list",
				Syntax:      "jira [global-flags] list PROJECT --created DURATION | --updated DURATION",
				Description: "List up to 50 tickets in a project filtered by recency. At least one of --created or --updated is required. When both are given, tickets matching either condition are returned, ordered by updated descending.",
				Args: []argDoc{
					{Name: "PROJECT", Description: "Jira project key, e.g. PROJ.", Required: true},
				},
				Flags: []flagDoc{
					{Name: "--created", Type: "duration", Description: "Include tickets created within this duration. Examples: 24h, 7d, 2w.", Required: false},
					{Name: "--updated", Type: "duration", Description: "Include tickets updated within this duration. Examples: 24h, 7d, 2w.", Required: false},
				},
			},
			{
				Name:        "help",
				Syntax:      "jira help",
				Description: "Print this machine-readable JSON documentation. No credentials required.",
			},
		},
		Notes: []string{
			"Duration format: h=hours, m=minutes, d=days, w=weeks. Examples: 1h, 24h, 7d, 2w.",
			"Credentials precedence: flags override environment variables (JIRA_URL, JIRA_USER, JIRA_TOKEN).",
			"With --user omitted, --token is sent as a Bearer header (PAT auth).",
			"JSON output fields for get and list: key, summary, type, status, priority, assignee, reporter, created, updated, description.",
		},
	}
	printJSON(doc)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Fatalf("json encode: %v", err)
	}
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
