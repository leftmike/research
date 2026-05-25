package main

import (
	"fmt"
	"io"
	"mime"

	"github.com/pemistahl/lingua-go"
)

func filter(cfg *config, args []string) {
	delete := len(args) > 0 && args[0] == "delete"

	detector := lingua.NewLanguageDetectorBuilder().
		FromAllLanguages().
		Build()

	conn := cfg.newConn()
	defer conn.Quit()

	msgs, err := conn.List(0)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Found %d messages\n", len(msgs))

	deleted := 0
	for _, m := range msgs {
		msg, err := conn.Retr(m.ID)
		if err != nil {
			fmt.Printf("skipping message %d: %v", m.ID, err)
			continue
		}

		body, _ := io.ReadAll(msg.Body)
		dec := mime.WordDecoder{}
		subject, _ := dec.DecodeHeader(msg.Header.Get("Subject"))
		lang, conf, exists := detectLang(detector, subject, msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), body)
		if !exists || lang == lingua.English {
			continue
		}
		fmt.Printf("[%s %.0f%%] msg %d: %q\n", lang, conf*100, m.ID, subject)

		if delete {
			if err := conn.Dele(m.ID); err != nil {
				fmt.Printf("failed to delete message %d: %v", m.ID, err)
				continue
			}
		}
		deleted++
	}

	action := "would delete"
	if delete {
		action = "deleted"
	}
	fmt.Printf("Done: %s %d of %d messages\n", action, deleted, len(msgs))
}
