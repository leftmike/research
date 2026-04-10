package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

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
