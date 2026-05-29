package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func delete(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <id>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}

	var ids []int
	for _, arg := range args {
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "delete: invalid message id: %s\n", arg)
			os.Exit(1)
		}
		ids = append(ids, id)
	}

	conn := cfg.newConn()
	defer conn.Quit()

	if cfg.Archive != "" {
		if _, err := os.Stat(cfg.Archive); err != nil {
			fatal(fmt.Errorf("archive directory does not exist: %s", cfg.Archive))
		}
		for _, id := range ids {
			uidls, err := conn.Uidl(id)
			if err != nil {
				fatal(err)
			}
			uid := uidls[0].UID
			buf, err := conn.Cmd("RETR", true, id)
			if err != nil {
				fatal(err)
			}
			path := filepath.Join(cfg.Archive, uid+".eml")
			if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
				fatal(err)
			}
			if err := conn.Dele(id); err != nil {
				fatal(err)
			}
			fmt.Printf("deleted %d (saved to %s)\n", id, path)
		}
		return
	}

	if err := conn.Dele(ids...); err != nil {
		fatal(err)
	}
	fmt.Println("deleted", strings.Join(args, ", "))
}
