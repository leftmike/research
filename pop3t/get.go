package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

func parseIDs(args []string) ([]int, error) {
	var ids []int
	for _, arg := range args {
		parts := strings.Split(arg, "-")
		switch len(parts) {
		case 1:
			id, err := strconv.Atoi(parts[0])
			if err != nil || id < 1 {
				return nil, fmt.Errorf("invalid message id: %s", arg)
			}
			ids = append(ids, id)
		case 2:
			lo, err := strconv.Atoi(parts[0])
			if err != nil || lo < 1 {
				return nil, fmt.Errorf("invalid message id range: %s", arg)
			}
			hi, err := strconv.Atoi(parts[1])
			if err != nil || hi < lo {
				return nil, fmt.Errorf("invalid message id range: %s", arg)
			}
			for id := lo; id <= hi; id++ {
				ids = append(ids, id)
			}
		default:
			return nil, fmt.Errorf("invalid message id: %s", arg)
		}
	}

	return ids, nil
}

var (
	format = "normal"
)

func getFlags(fs *flag.FlagSet) {
	fs.StringVar(&format, "format", "normal", "brief|normal|full|raw")
}

func get(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <id|range>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}
	if !slices.Contains([]string{"brief", "normal", "full", "raw"}, format) {
		fmt.Fprintf(os.Stderr, "usage: %s %s: expected brief, normal, full, or raw for format\n",
			os.Args[0], os.Args[1])
		os.Exit(1)
	}

	ids, err := parseIDs(args)
	if err != nil {
		fatal(err)
	}

	conn, err := cfg.newConn()
	if err != nil {
		fatal(err)
	}
	defer conn.Quit()

	for i, id := range ids {
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
