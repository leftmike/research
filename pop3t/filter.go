package main

import (
	"fmt"

	"github.com/pemistahl/lingua-go"
)

func filter(cfg *config, args []string) {
	delete := len(args) > 0 && args[0] == "delete"

	conn := cfg.newConn()
	defer conn.Quit()

	mids, err := conn.List(0)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Found %d messages\n", len(mids))

	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	deleted := 0
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

		lang, conf, exists := msg.detectLanguage(ld)
		if !exists || lang == lingua.English || conf < 0.5 {
			continue
		}
		fmt.Printf("%3d [%s %.0f%%] %s\n", mid.ID, lang, conf*100, msg.subject)

		if delete {
			if err := conn.Dele(mid.ID); err != nil {
				fmt.Printf("failed to delete message %d: %v", mid.ID, err)
				continue
			}
		}
		deleted += 1
	}

	action := "would delete"
	if delete {
		action = "deleted"
	}
	fmt.Printf("done: %s %d of %d messages\n", action, deleted, len(mids))
}
