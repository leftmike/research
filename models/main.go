package main

import (
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintln(os.Stderr, `usage: models [global flags] <command> [args]

Reads model, provider, and lab information from models.dev and litellm.

Commands:
  summary                Overview: source, provider, lab, and model counts.
  providers [SUBSTR]     List providers (optionally filtered by substring).
  provider  <id>         Show one provider and the models it serves.
  labs      [SUBSTR]     List labs (model creators) and their model counts.
  lab       <name>       Show one lab and its models.
  models    [SUBSTR]     List models (optionally filtered by substring).
  model     <id>         Show full detail for one model, merged across sources.

If the first argument is not one of the commands above, it is matched against
providers; when exactly one provider matches it is shown, and an optional
second argument selects a model within it:

  models <provider>            Show the matching provider (like "provider").
  models <provider> <model>    Show a model within that provider.

Global flags:
  -refresh   Ignore cached data and re-download from the sources.
  -no-cache  Do not read or write the on-disk cache.

Cached data lives under the user cache dir (refreshed every 24h).
Run "models <command> -h" for command-specific flags.`)
}

func main() {
	refresh := flag.Bool("refresh", false, "ignore cached data and re-download")
	noCache := flag.Bool("no-cache", false, "do not read or write the cache")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	opts := defaultFetchOptions()
	opts.refresh = *refresh
	opts.noCache = *noCache

	cmd := args[0]
	rest := args[1:]

	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		usage()
		return
	}

	reg, err := buildRegistry(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "summary":
		cmdSummary(reg, rest)
	case "providers":
		cmdProviders(reg, rest)
	case "provider":
		cmdProvider(reg, rest)
	case "labs":
		cmdLabs(reg, rest)
	case "lab":
		cmdLab(reg, rest)
	case "models":
		cmdModels(reg, rest)
	case "model":
		cmdModel(reg, rest)
	default:
		// No recognized command: treat the args as "<provider> [<model>]".
		cmdDefault(reg, args)
	}
}
