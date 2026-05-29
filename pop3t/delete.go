package main

import (
	"fmt"
	"os"
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

	err := conn.Dele(ids...)
	if err != nil {
		fatal(err)
	}

	fmt.Println("deleted", strings.Join(args, ", "))
}
