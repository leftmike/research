package main

import (
	"bytes"
	"flag"
	"fmt"
	"mime"
	"mime/multipart"
	"os"
	"slices"
	"strconv"
	"strings"
)

func isBase64(cte string) bool {
	return strings.EqualFold(strings.TrimSpace(cte), "base64")
}

func displayBody(contentType, transferEnc string, body []byte) string {
	if isBase64(transferEnc) {
		return fmt.Sprintf("[%s]", contentType)
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return extractPlainText(contentType, transferEnc, body)
	}
	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var parts []string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		pt, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if pt != "text/plain" {
			continue
		}
		cte := part.Header.Get("Content-Transfer-Encoding")
		if isBase64(cte) {
			parts = append(parts, fmt.Sprintf("[%s]", part.Header.Get("Content-Type")))
		} else {
			parts = append(parts, decodeBody(cte, part))
		}
	}
	return strings.Join(parts, "\n")
}

var (
	format = "normal"
)

func getFlags(fs *flag.FlagSet) {
	fs.StringVar(&format, "format", "normal", "brief|normal|full|raw")
}

func get(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <id>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}
	if !slices.Contains([]string{"brief", "normal", "full", "raw"}, format) {
		fmt.Fprintf(os.Stderr, "usage: %s %s: expected brief, normal, full, or raw for format\n",
			os.Args[0], os.Args[1])
		os.Exit(1)
	}

	conn := cfg.newConn()
	defer conn.Quit()

	for i, arg := range args {
		id, err := strconv.Atoi(arg)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "get: invalid message id: %s\n", arg)
			os.Exit(1)
		}

		entity, err := conn.Retr(id)
		if err != nil {
			fatal(err)
		}

		if i > 0 {
			fmt.Println(
				"--------------------------------------------------------------------------------")
		}

		msg, err := messageFromEntity(entity)
		if err != nil {
			fatal(err)
		}

		switch format {
		case "brief":
		// XXX

		case "normal":
			for _, field := range []string{"Date", "From", "To"} {
				raw := msg.header.Get(field)
				if decoded, err := (&mime.WordDecoder{}).DecodeHeader(raw); err == nil {
					raw = decoded
				}
				if raw != "" {
					fmt.Printf("%s: %s\n", field, raw)
				}
			}
			fmt.Printf("Subject: %s\n", msg.subject)
			fmt.Printf("Content-Type: %s\n", msg.header.Get("Content-Type"))
			if cte := msg.header.Get("Content-Transfer-Encoding"); cte != "" {
				fmt.Printf("Content-Transfer-Encoding: %s\n", cte)
			}
			fmt.Println()
			fmt.Println(displayBody(msg.header.Get("Content-Type"),
				msg.header.Get("Content-Transfer-Encoding"), msg.body))

		case "full":
			// XXX

		case "raw":
			// XXX
		}
	}
}
