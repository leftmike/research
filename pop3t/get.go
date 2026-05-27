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
	"unicode"
)

func isBase64(cte string) bool {
	return strings.EqualFold(strings.TrimSpace(cte), "base64")
}

func truncate(s string, n int) string {
	var buf strings.Builder
	pr := rune('\n')
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == '\u0085' || r == '\u2028' ||
			r == '\u2029' {

			if pr != '\n' {
				buf.WriteRune('\n')
			}
			pr = '\n'
		} else if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			if pr != '\n' {
				pr = ' '
			}
		} else {
			if pr == ' ' {
				buf.WriteRune(' ')
			}
			buf.WriteRune(r)
			pr = r
		}

		if buf.Len() >= n {
			buf.WriteString("...")
			break
		}
	}

	return strings.TrimSpace(buf.String())
}

func formatBody(contentType, transferEnc string, body []byte, summarize bool) string {
	if isBase64(transferEnc) {
		return fmt.Sprintf("[%s, %s, %d bytes]", contentType, transferEnc, len(body))
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		text := decodeBody(transferEnc, bytes.NewReader(body))
		if summarize {
			return truncate(text, 90)
		}
		return text
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
			if summarize {
				var header string
				if cte != "" {
					header = fmt.Sprintf("[%s, %s, %d bytes]\n", ct, cte, len(text))
				} else {
					header = fmt.Sprintf("[%s, %d bytes]\n", ct, len(text))
				}
				parts = append(parts, header+truncate(text, 90))
			} else {
				parts = append(parts, text)
			}
		} else {
			b, _ := io.ReadAll(part)
			if cte != "" {
				parts = append(parts, fmt.Sprintf("[%s, %s, %d bytes]", ct, cte, len(b)))
			} else {
				parts = append(parts, fmt.Sprintf("[%s, %d bytes]", ct, len(b)))
			}
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
				fmt.Println(formatBody(msg.header.Get("Content-Type"),
					msg.header.Get("Content-Transfer-Encoding"), msg.body, true))

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
				fmt.Println(formatBody(msg.header.Get("Content-Type"),
					msg.header.Get("Content-Transfer-Encoding"), msg.body, false))

			case "full":
				// XXX
			}
		}
	}
}
