package main

import (
	"fmt"

	"github.com/pemistahl/lingua-go"
)

func list(cfg *config, args []string) {
	conn := cfg.newConn()
	defer conn.Quit()

	mids, err := conn.List(0)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("found %d messages\n", len(mids))

	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	for _, mid := range mids {
		entity, err := conn.Retr(mid.ID)
		if err != nil {
			fmt.Printf("skipping: retr(%d): %s", mid.ID, err)
			continue
		}

		msg, err := messageFromEntity(entity)
		if err != nil {
			fmt.Printf("skipping: entity(%d): %s", mid.ID, err)
			continue
		}

		lang, conf, _ := msg.detectLanguage(ld)
		fmt.Printf("%3d  [%s %.0f%%] %s\n", mid.ID, lang, conf*100, msg.subject)
	}
}
