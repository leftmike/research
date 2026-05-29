package main

import (
	"fmt"

	msgformat "github.com/emersion/go-message"
	"github.com/knadh/go-pop3"
	"github.com/pemistahl/lingua-go"
)

func list(cfg *config, args []string) {
	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	cfg.list(func(_ *pop3.Conn, id int, entity *msgformat.Entity) {
		msg, err := messageFromEntity(entity)
		if err != nil {
			fmt.Printf("skipping: entity(%d): %s", id, err)
			return
		}
		lang, conf, _ := msg.detectLanguage(ld)
		fmt.Printf("%3d  [%s %.0f%%] %s\n", id, lang, conf*100, msg.subject)
	})
}
