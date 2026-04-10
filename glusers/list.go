package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type sortKey struct {
	field string
	desc  bool
}

func truncateUsername(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

func parseRoleFilters(values []string) (map[gitlab.AccessLevelValue]bool, error) {
	if len(values) == 0 {
		return nil, nil
	}
	filter := make(map[gitlab.AccessLevelValue]bool)
	for _, value := range values {
		part := strings.TrimSpace(value)
		if part == "" {
			return nil, fmt.Errorf("invalid role filter: empty role")
		}
		accessLevel, err := parseAccessLevel(strings.ToLower(part))
		if err != nil {
			return nil, err
		}
		filter[accessLevel] = true
	}
	return filter, nil
}

func parseListArgs(args []string) (map[gitlab.AccessLevelValue]bool, []sortKey, error) {
	var roles []string
	var keys []sortKey
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			return nil, nil, fmt.Errorf("invalid list argument: empty value")
		}
		switch arg[0] {
		case '+', '-':
			if len(arg) == 1 {
				return nil, nil, fmt.Errorf("invalid sort key: %q", arg)
			}
			desc := arg[0] == '-'
			key := arg[1:]
			switch key {
			case "user", "role", "expires":
				keys = append(keys, sortKey{field: key, desc: desc})
			default:
				return nil, nil, fmt.Errorf("invalid sort key %q: must be user, role, or expires", key)
			}
		default:
			roles = append(roles, arg)
		}
	}
	filterSet, err := parseRoleFilters(roles)
	if err != nil {
		return nil, nil, err
	}
	if len(keys) == 0 {
		keys = []sortKey{
			{field: "role", desc: true},
			{field: "expires", desc: false},
			{field: "user", desc: false},
		}
	}
	return filterSet, keys, nil
}

func compareExpires(a, b *gitlab.ISOTime) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	at := time.Time(*a)
	bt := time.Time(*b)
	if at.Before(bt) {
		return -1
	}
	if at.After(bt) {
		return 1
	}
	return 0
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	urlFlag := fs.String("url", "", "GitLab base URL (or $GITLAB_URL)")
	tokenFlag := fs.String("token", "", "personal access token (or $GITLAB_TOKEN)")
	insecureFlag := fs.Bool("insecure", false, "skip TLS certificate verification")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: glusers list [flags] <project> [roles...] [+key|-key]...")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "sort keys: +user -user +role -role +expires -expires")
		fmt.Fprintln(os.Stderr, "default sort: -role +expires +user")
		fmt.Fprintln(os.Stderr, "roles: guest reporter developer maintainer owner")
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
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	project := fs.Arg(0)
	filterSet, sortKeys, err := parseListArgs(fs.Args()[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fs.Usage()
		os.Exit(1)
	}

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

	if len(filterSet) > 0 {
		filtered := members[:0]
		for _, m := range members {
			if filterSet[m.AccessLevel] {
				filtered = append(filtered, m)
			}
		}
		members = filtered
	}

	if len(sortKeys) > 0 && len(members) > 1 {
		sort.SliceStable(members, func(i, j int) bool {
			a := members[i]
			b := members[j]
			for _, key := range sortKeys {
				var cmp int
				switch key.field {
				case "user":
					au := strings.ToLower(a.Username)
					bu := strings.ToLower(b.Username)
					if au < bu {
						cmp = -1
					} else if au > bu {
						cmp = 1
					}
				case "role":
					if a.AccessLevel < b.AccessLevel {
						cmp = -1
					} else if a.AccessLevel > b.AccessLevel {
						cmp = 1
					}
				case "expires":
					cmp = compareExpires(a.ExpiresAt, b.ExpiresAt)
				}
				if cmp == 0 {
					continue
				}
				if key.desc {
					return cmp > 0
				}
				return cmp < 0
			}
			return false
		})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USER\tACCESS LEVEL\tEXPIRES")
	for _, m := range members {
		expires := ""
		if m.ExpiresAt != nil {
			expires = m.ExpiresAt.String()
		}
		username := strings.TrimSpace(m.Username)
		username = truncateUsername(username, 20)
		fmt.Fprintf(w, "%s\t%s\t%s\n", username, accessLevelName(m.AccessLevel), expires)
	}
	w.Flush()
}
