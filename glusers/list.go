package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	urlFlag := fs.String("url", "", "GitLab base URL (or $GITLAB_URL)")
	tokenFlag := fs.String("token", "", "personal access token (or $GITLAB_TOKEN)")
	insecureFlag := fs.Bool("insecure", false, "skip TLS certificate verification")
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
