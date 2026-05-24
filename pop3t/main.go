package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"os"
	"strings"

	"github.com/pemistahl/lingua-go"
)

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func decodeBody(enc string, r io.Reader) string {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "quoted-printable":
		b, _ := io.ReadAll(quotedprintable.NewReader(r))
		return string(b)
	default:
		b, _ := io.ReadAll(r)
		return string(b)
	}
}

func extractPlainText(contentType, transferEnc string, body []byte) string {
	if contentType == "" {
		return decodeBody(transferEnc, bytes.NewReader(body))
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return decodeBody(transferEnc, bytes.NewReader(body))
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		return decodeBody(transferEnc, bytes.NewReader(body))
	}

	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var parts []string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		pt, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if pt == "text/plain" {
			parts = append(parts, decodeBody(part.Header.Get("Content-Transfer-Encoding"), part))
		}
	}
	return strings.Join(parts, "\n")
}

func detectLang(detector lingua.LanguageDetector, subject, contentType, transferEnc string, body []byte) (lingua.Language, float64, bool) {
	detectText := subject + "\n"
	if isASCII(subject) {
		detectText += extractPlainText(contentType, transferEnc, body)
	}
	vals := detector.ComputeLanguageConfidenceValues(detectText)
	if len(vals) == 0 {
		return lingua.Unknown, 0, false
	}
	return vals[0].Language(), vals[0].Value(), true
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <cmd> [flags] [args]\n", os.Args[0])
	for name, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "    %s: %s\n", name, cmd.help)
	}
}

type cmd struct {
	fn   func(cfg *config, args []string)
	help string
}

var (
	cmds = map[string]cmd{
		"filter": {fn: filter, help: ""},
		"help":   {help: ""},
		"list":   {fn: list, help: ""},
	}
)

func main() {
	if len(os.Args) >= 2 {
		cmd, ok := cmds[os.Args[1]]
		if ok {
			if os.Args[1] == "help" {
				usage()
				return
			}

			cfg, args := loadConfig()
			cmd.fn(cfg, args)
			return
		}
	}

	usage()
	os.Exit(1)
}
