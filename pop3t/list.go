package main

import (
	"fmt"
	"io"
	"log"
	"mime"

	"github.com/pemistahl/lingua-go"
)

func list(cfg *config, args []string) {
	detector := lingua.NewLanguageDetectorBuilder().
		FromAllLanguages().
		Build()

	conn := cfg.newConn()
	defer conn.Quit()

	msgs, err := conn.List(0)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d messages\n", len(msgs))

	dec := mime.WordDecoder{}
	for _, m := range msgs {
		msg, err := conn.Retr(m.ID)
		if err != nil {
			log.Printf("skipping message %d: %v", m.ID, err)
			continue
		}
		body, _ := io.ReadAll(msg.Body)
		subject, _ := dec.DecodeHeader(msg.Header.Get("Subject"))
		lang, conf, _ := detectLang(detector, subject, msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), body)
		fmt.Printf("%3d  [%s %.0f%%] %s\n", m.ID, lang, conf*100, subject)
	}
}
