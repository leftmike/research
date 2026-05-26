package main

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"strings"
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

	subject, err := (&mime.WordDecoder{}).DecodeHeader(entity.Header.Get("Subject"))
	if err != nil {
		return nil, err
	}

	return &message{
		header:  &entity.Header,
		subject: subject,
		body:    body,
	}, nil
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
