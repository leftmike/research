package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"strings"
	"unicode"
	"unicode/utf8"

	msgformat "github.com/emersion/go-message"
	"github.com/pemistahl/lingua-go"
)

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

type message struct {
	header  *msgformat.Header
	subject string
	body    []byte
}

func messageFromEntity(entity *msgformat.Entity) (*message, error) {
	body, err := io.ReadAll(entity.Body)
	if err != nil {
		return nil, err
	}

	wd := &mime.WordDecoder{CharsetReader: msgformat.CharsetReader}
	subject, err := wd.DecodeHeader(entity.Header.Get("Subject"))
	if err != nil {
		return nil, err
	}

	return &message{
		header:  &entity.Header,
		subject: subject,
		body:    body,
	}, nil
}

func truncate(s string, n int) string {
	var buf strings.Builder
	pr := rune('\n')
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == '' || r == ' ' ||
			r == ' ' {

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

func (msg *message) printHeaders(fields []string) {
	for _, field := range fields {
		raw := msg.header.Get(field)
		decoded, err := (&mime.WordDecoder{
			CharsetReader: msgformat.CharsetReader,
		}).DecodeHeader(raw)
		if err == nil {
			raw = decoded
		}
		if raw != "" {
			fmt.Printf("%s: %s\n", field, raw)
		}
	}
	fmt.Printf("Subject: %s\n", msg.subject)
}

func (msg *message) formatBody(summarize bool) string {
	contentType := msg.header.Get("Content-Type")
	transferEnc := msg.header.Get("Content-Transfer-Encoding")
	if strings.EqualFold(strings.TrimSpace(transferEnc), "base64") {
		return fmt.Sprintf("[%s, %s, %d bytes]", contentType, transferEnc, len(msg.body))
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		text := decodeBody(transferEnc, bytes.NewReader(msg.body))
		if summarize {
			return truncate(text, 90)
		}
		return text
	}

	mr := multipart.NewReader(bytes.NewReader(msg.body), params["boundary"])
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
		if strings.HasPrefix(pt, "text/plain") && !strings.EqualFold(strings.TrimSpace(cte), "base64") {
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

func (msg *message) detectLanguage(ld lingua.LanguageDetector) (lingua.Language, float64, bool) {
	detectText := msg.subject + "\n"
	if len(msg.subject) == utf8.RuneCountInString(msg.subject) {
		detectText += extractPlainText(msg.header.Get("Content-Type"),
			msg.header.Get("Content-Transfer-Encoding"), msg.body)
	}
	vals := ld.ComputeLanguageConfidenceValues(detectText)
	if len(vals) == 0 {
		return lingua.Unknown, 0, false
	}
	/*
		if vals[1].Value() > 0 {
			fmt.Printf("%s %.0f%% ", vals[1].Language(), vals[1].Value()*100)
		}
	*/
	return vals[0].Language(), vals[0].Value(), true
}
