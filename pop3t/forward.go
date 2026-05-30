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

	host := cfg.smtpHost()
	user := cfg.smtpUser()
	password := cfg.smtpPassword()

	port := cfg.SMTP.Port
	if port == 0 {
		if cfg.SMTP.NoTLS {
			port = 25
		} else {
			port = 587
		}
	}

	conn := cfg.newConn()
	defer conn.Quit()

	auth := smtp.PlainAuth("", user, password, host)
	addr := fmt.Sprintf("%s:%d", host, port)

	for _, id := range ids {
		buf, err := conn.Cmd("RETR", true, id)
		if err != nil {
			fatal(err)
		}

		if err := smtp.SendMail(addr, auth, user, []string{to}, buf.Bytes()); err != nil {
			fatal(err)
		}

		fmt.Printf("forwarded %d to %s\n", id, to)
	}
}
