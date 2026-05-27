package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
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

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func briefBody(contentType, transferEnc string, body []byte) string {
	if isBase64(transferEnc) {
		return fmt.Sprintf("[%s]", contentType)
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return truncate(decodeBody(transferEnc, bytes.NewReader(body)), 200)
	}

	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var parts []string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		ct := part.Header.Get("Content-Type")
		cte := part.Header.Get("Content-Transfer-Encoding")
		pt, _, _ := mime.ParseMediaType(ct)
		if pt == "" {
			pt = "text/plain"
		}
		if strings.HasPrefix(pt, "text/plain") && !isBase64(cte) {
			text := decodeBody(cte, part)
			parts = append(parts, "[text/plain] "+truncate(text, 120))
		} else {
			b, _ := io.ReadAll(part)
			parts = append(parts, fmt.Sprintf("[%s %d bytes]", pt, len(b)))
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

		if i > 0 {
			fmt.Println(
				"--------------------------------------------------------------------------------")
		}

		if format == "raw" {
			buf, err := conn.Cmd("RETR", true, id)
			if err != nil {
				fatal(err)
			}
			os.Stdout.Write(buf.Bytes())
		} else {
			entity, err := conn.Retr(id)
			if err != nil {
				fatal(err)
			}

			msg, err := messageFromEntity(entity)
			if err != nil {
				fatal(err)
			}

			switch format {
			case "brief":
				for _, field := range []string{"Date", "From"} {
					raw := msg.header.Get(field)
					if decoded, err := (&mime.WordDecoder{}).DecodeHeader(raw); err == nil {
						raw = decoded
					}
					if raw != "" {
						fmt.Printf("%s: %s\n", field, raw)
					}
				}
				fmt.Printf("Subject: %s\n", msg.subject)
				fmt.Println()
				fmt.Println(briefBody(msg.header.Get("Content-Type"),
					msg.header.Get("Content-Transfer-Encoding"), msg.body))

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
			}
		}
	}
}
