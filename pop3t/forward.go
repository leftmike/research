package main

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
)

func forward(cfg *config, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <addr> <id>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}

	to := args[0]
	ids := args[1:]

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

	for _, arg := range ids {
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "forward: invalid message id: %s\n", arg)
			os.Exit(1)
		}

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
