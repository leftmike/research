package main

import (
	"fmt"
	"net/smtp"
	"os"
)

func forward(cfg *config, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <addr> <id|range>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}

	to := args[0]

	ids, err := parseIDs(args[1:])
	if err != nil {
		fatal(err)
	}
	addr, auth, err := cfg.newSend()
	if err != nil {
		fatal(err)
	}

	conn, err := cfg.newConn()
	if err != nil {
		fatal(err)
	}
	defer conn.Quit()

	for _, id := range ids {
		buf, err := conn.Cmd("RETR", true, id)
		if err != nil {
			fatal(err)
		}

		err = smtp.SendMail(addr, auth, cfg.smtpUser(), []string{to}, buf.Bytes())
		if err != nil {
			fatal(err)
		}

		fmt.Printf("forwarded %d to %s\n", id, to)
	}
}
