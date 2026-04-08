package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
)

var (
	jiraURL   = flag.String("url", "", "Jira base URL (e.g. https://mycompany.atlassian.net)")
	jiraUser  = flag.String("user", "", "Jira username (email)")
	jiraToken = flag.String("token", "", "Jira API token")
)

func main() {
	flag.Parse()

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

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: jira [flags] TICKET-ID\n\n")
		fmt.Fprintf(os.Stderr, "Credentials may be set via flags or environment variables (JIRA_URL, JIRA_USER, JIRA_TOKEN).\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if url == "" || user == "" || token == "" {
		fmt.Fprintf(os.Stderr, "error: --url, --user, and --token are required (or set JIRA_URL, JIRA_USER, JIRA_TOKEN)\n")
		os.Exit(1)
	}

	issueKey := flag.Arg(0)

	tp := jira.BasicAuthTransport{
		Username: user,
		APIToken: token,
	}
	client, err := jira.NewClient(url, tp.Client())
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	issue, _, err := client.Issue.Get(context.Background(), issueKey, nil)
	if err != nil {
		log.Fatalf("get issue %s: %v", issueKey, err)
	}

	printIssue(issue)
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
