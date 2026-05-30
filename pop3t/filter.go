package main

import (
	"fmt"

	msgformat "github.com/emersion/go-message"
	"github.com/knadh/go-pop3"
	"github.com/pemistahl/lingua-go"
)

func filter(cfg *config, args []string) {
	doDelete := len(args) > 0 && args[0] == "delete"
	cnt := 0

	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	tot, err := cfg.list(func(conn *pop3.Conn, id int, entity *msgformat.Entity) error {
		msg, err := messageFromEntity(entity)
		if err != nil {
			fmt.Printf("skipping: entity(%d): %s\n", id, err)
			return nil
		}
		lang, conf, exists := msg.detectLanguage(ld)
		if !exists || lang == lingua.English || conf < 0.5 {
			return nil
		}
		fmt.Printf("%3d [%s %.0f%%] %s\n", id, lang, conf*100, msg.subject)
		if doDelete {
			if err := conn.Dele(id); err != nil {
				fmt.Printf("failed to delete message %d: %s\n", id, err)
				return nil
			}
		}
		cnt += 1
		return nil
	})
	if err != nil {
		fatal(err)
	}

	action := "would delete"
	if doDelete {
		action = "deleted"
	}
	fmt.Printf("done: %s %d of %d messages\n", action, cnt, tot)
}
