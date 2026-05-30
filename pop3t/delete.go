package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/knadh/go-pop3"
)

func deleteId(cfg *config, conn *pop3.Conn, id int) (string, error) {
	if cfg.Archive != "" {
		if _, err := os.Stat(cfg.Archive); err != nil {
			return "", fmt.Errorf("archive directory does not exist: %s", cfg.Archive)
		}

		uidls, err := conn.Uidl(id)
		if err != nil {
			return "", err
		}

		buf, err := conn.Cmd("RETR", true, id)
		if err != nil {
			return "", err
		}

		path := filepath.Join(cfg.Archive, uidls[0].UID+".eml")
		err = os.WriteFile(path, buf.Bytes(), 0644)
		if err != nil {
			return "", err
		}
		err = conn.Dele(id)
		if err != nil {
			return "", err
		}

		return path, nil
	}

	return "", conn.Dele(id)
}

func delete(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <id>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}

	conn := cfg.newConn()
	defer conn.Quit()

	for _, arg := range args {
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "delete: invalid message id: %s\n", arg)
			os.Exit(1)
		}

		path, err := deleteId(cfg, conn, id)
		if err != nil {
			fatal(err)
		}

		fmt.Printf("deleted %d", id)
		if path != "" {
			fmt.Printf(" (saved to %s)", path)
		}
		fmt.Println()
	}
}
