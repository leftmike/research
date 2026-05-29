/*
To Do:
- verbose (including -v) flag
- clean up filter and list to make code more common
- filter: specific the language(s) as english !russian
- filter: actions: delete, forward, etc
- option to save all messages that are deleted in raw format
*/
package main

import (
	"flag"
	"fmt"
	"os"
)

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], os.Args[1], err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <cmd> [flags] [args]\n", os.Args[0])
	for name, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "    %s: %s\n", name, cmd.help)
	}
}

type cmd struct {
	flags func(fs *flag.FlagSet)
	run   func(cfg *config, args []string)
	help  string
}

var (
	cmds = map[string]cmd{
		"delete":  {run: delete, help: "delete one or more messages by id"},
		"filter":  {run: filter, help: "list non-English messages; pass 'delete' to remove them"},
		"forward": {run: forward, help: "forward one or more messages by id to an email address"},
		"get":     {flags: getFlags, run: get, help: "fetch and display a single message by id"},
		"help":    {help: "show this help"},
		"list":    {run: list, help: "list all messages with id, language, and subject"},
	}
)

func main() {
	if len(os.Args) >= 2 {
		cmd, ok := cmds[os.Args[1]]
		if ok {
			if os.Args[1] == "help" {
				usage()
				return
			}

			fs := flag.NewFlagSet(fmt.Sprintf("%s %s", os.Args[0], os.Args[1]), flag.ExitOnError)
			if cmd.flags != nil {
				cmd.flags(fs)
			}
			cfg, args := loadConfig(fs, os.Args[2:])

			cmd.run(cfg, args)
			return
		}
	}

	usage()
	os.Exit(1)
}
