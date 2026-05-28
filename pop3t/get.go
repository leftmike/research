package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
)

var (
	format = "normal"
)

func getFlags(fs *flag.FlagSet) {
	fs.StringVar(&format, "format", "normal", "brief|normal|full|raw")
}

func get(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <id>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}
	if !slices.Contains([]string{"brief", "normal", "full", "raw"}, format) {
		fmt.Fprintf(os.Stderr, "usage: %s %s: expected brief, normal, full, or raw for format\n",
			os.Args[0], os.Args[1])
		os.Exit(1)
	}

	conn := cfg.newConn()
	defer conn.Quit()

	for i, arg := range args {
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "get: invalid message id: %s\n", arg)
			os.Exit(1)
		}

		if i > 0 {
			fmt.Println(
				"--------------------------------------------------------------------------------")
		}

		if format == "raw" {
			buf, err := conn.Cmd("RETR", true, id)
			if err != nil {
				fatal(err)
			}
			os.Stdout.Write(buf.Bytes())
		} else {
			entity, err := conn.Retr(id)
			if err != nil {
				fatal(err)
			}

			msg, err := messageFromEntity(entity)
			if err != nil {
				fatal(err)
			}

			switch format {
			case "brief":
				msg.printHeaders([]string{"Date", "From"})
				fmt.Println()
				fmt.Println(msg.formatBody(true))

			case "normal":
				msg.printHeaders([]string{"Date", "From", "To", "CC", "BCC"})
				fmt.Printf("Content-Type: %s\n", msg.header.Get("Content-Type"))
				if cte := msg.header.Get("Content-Transfer-Encoding"); cte != "" {
					fmt.Printf("Content-Transfer-Encoding: %s\n", cte)
				}
				fmt.Println()
				fmt.Println(msg.formatBody(false))

			case "full":
				// XXX
			}
		}
	}
}
