package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/smtp"
	"os"
	"strings"
)

var (
	subject string
)

func sendFlags(fs *flag.FlagSet) {
	fs.StringVar(&subject, "subject", "", "email subject")
}

func send(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s [-subject <subject>] <to-address>...\n",
			os.Args[0], os.Args[1])
		os.Exit(1)
	}

	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fatal(err)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", cfg.smtpUser())
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(args, ", "))
	if subject != "" {
		fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	}
	fmt.Fprintf(&buf, "\r\n")
	buf.Write(body)

	addr, auth, err := cfg.newSend()
	if err != nil {
		fatal(err)
	}

	if err := smtp.SendMail(addr, auth, cfg.smtpUser(), args, buf.Bytes()); err != nil {
		fatal(err)
	}

	fmt.Printf("sent to %s\n", strings.Join(args, ", "))
}
