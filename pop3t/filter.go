package main

import (
	"fmt"

	msgformat "github.com/emersion/go-message"
	"github.com/knadh/go-pop3"
	"github.com/pemistahl/lingua-go"
)

func filter(cfg *config, args []string) {
	doDelete := len(args) > 0 && args[0] == "delete"
	deleted := 0

	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	total := cfg.list(func(conn *pop3.Conn, id int, entity *msgformat.Entity) {
		msg, err := messageFromEntity(entity)
		if err != nil {
			fmt.Printf("skipping: entity(%d): %s", id, err)
			return
		}
		lang, conf, exists := msg.detectLanguage(ld)
		if !exists || lang == lingua.English || conf < 0.5 {
			return
		}
		fmt.Printf("%3d [%s %.0f%%] %s\n", id, lang, conf*100, msg.subject)
		if doDelete {
			if err := conn.Dele(id); err != nil {
				fmt.Printf("failed to delete message %d: %v", id, err)
				return
			}
		}
		deleted++
	})

	action := "would delete"
	if doDelete {
		action = "deleted"
	}
	fmt.Printf("done: %s %d of %d messages\n", action, deleted, total)
}
