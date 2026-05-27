package main

import (
	"fmt"
	"os"
)

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <cmd> [flags] [args]\n", os.Args[0])
	for name, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "    %s: %s\n", name, cmd.help)
	}
}

type cmd struct {
	fn   func(cfg *config, args []string)
	help string
}

var (
	cmds = map[string]cmd{
		"filter": {fn: filter, help: ""},
		"get":    {fn: get, help: "fetch and display a single message by id"},
		"help":   {help: ""},
		"list":   {fn: list, help: ""},
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

			cfg, args := loadConfig()
			cmd.fn(cfg, args)
			return
		}
	}

	usage()
	os.Exit(1)
}
