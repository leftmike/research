package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var (
	pop3Server = flag.String("pop3-server", "", "POP3 server hostname")
	pop3Port   = flag.String("pop3-port", "995", "POP3 server port (TLS)")
	pop3User   = flag.String("pop3-user", "", "POP3 username")
	pop3Pass   = flag.String("pop3-pass", "", "POP3 password")
	credFile   = flag.String("credentials", "credentials.json", "path to OAuth2 credentials file")
	tokenFile  = flag.String("token", "token.json", "path to OAuth2 token file")
	labelName  = flag.String("label", "", "Gmail label to apply (optional)")
	deleteMsgs = flag.Bool("delete", false, "delete messages from POP3 after insert")
)

// POP3 session

type pop3Session struct {
	conn   net.Conn
	reader *bufio.Reader
}

func pop3Dial(server, port, user, pass string) (*pop3Session, error) {
	conn, err := tls.Dial("tcp", server+":"+port, nil)
	if err != nil {
		return nil, fmt.Errorf("pop3 dial: %w", err)
	}
	s := &pop3Session{conn: conn, reader: bufio.NewReader(conn)}
	if _, err := s.readLine(); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := s.cmd("USER " + user); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := s.cmd("PASS " + pass); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}

func (s *pop3Session) readLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("pop3 read: %w", err)
	}
	if strings.HasPrefix(line, "-ERR") {
		return "", fmt.Errorf("pop3: %s", strings.TrimSpace(line))
	}
	return line, nil
}

func (s *pop3Session) cmd(command string) (string, error) {
	fmt.Fprintf(s.conn, "%s\r\n", command)
	return s.readLine()
}

func (s *pop3Session) stat() (int, error) {
	line, err := s.cmd("STAT")
	if err != nil {
		return 0, err
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0, fmt.Errorf("pop3 stat: unexpected response: %s", line)
	}
	return strconv.Atoi(parts[1])
}

func (s *pop3Session) retr(msgNum int) ([]byte, error) {
	if _, err := s.cmd(fmt.Sprintf("RETR %d", msgNum)); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("pop3 retr: %w", err)
		}
		if line == ".\r\n" || line == ".\n" {
			break
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		buf.WriteString(line)
	}
	return buf.Bytes(), nil
}

func (s *pop3Session) dele(msgNum int) error {
	_, err := s.cmd(fmt.Sprintf("DELE %d", msgNum))
	return err
}

func (s *pop3Session) quit() error {
	_, err := s.cmd("QUIT")
	s.conn.Close()
	return err
}

// Gmail OAuth2

func getGmailService(credPath, tokPath string) (*gmail.Service, error) {
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	config, err := google.ConfigFromJSON(b, gmail.GmailInsertScope, gmail.GmailLabelsScope)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	tok, err := loadToken(tokPath)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		saveToken(tokPath, tok)
	}
	ctx := context.Background()
	return gmail.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, tok)))
}

func loadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var tok oauth2.Token
	return &tok, json.NewDecoder(f).Decode(&tok)
}

func saveToken(path string, tok *oauth2.Token) {
	f, err := os.Create(path)
	if err != nil {
		log.Printf("Warning: could not save token: %v", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(tok)
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit this URL to authorize:\n%s\nPaste the authorization code: ", authURL)
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("read auth code: %w", err)
	}
	return config.Exchange(context.Background(), code)
}

// Label resolution

func resolveLabel(svc *gmail.Service, name string) (string, error) {
	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		return "", fmt.Errorf("list labels: %w", err)
	}
	for _, l := range resp.Labels {
		if l.Name == name {
			return l.Id, nil
		}
	}
	label, err := svc.Users.Labels.Create("me", &gmail.Label{
		Name:                name,
		LabelListVisibility: "labelShow",
		MessageListVisibility: "show",
	}).Do()
	if err != nil {
		return "", fmt.Errorf("create label: %w", err)
	}
	return label.Id, nil
}

func main() {
	flag.Parse()
	if *pop3Server == "" || *pop3User == "" || *pop3Pass == "" {
		fmt.Fprintf(os.Stderr, "Usage: p3gmfwd --pop3-server HOST --pop3-user USER --pop3-pass PASS\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	svc, err := getGmailService(*credFile, *tokenFile)
	if err != nil {
		log.Fatalf("Gmail setup: %v", err)
	}

	labelIDs := []string{"INBOX"}
	if *labelName != "" {
		id, err := resolveLabel(svc, *labelName)
		if err != nil {
			log.Fatalf("Label setup: %v", err)
		}
		labelIDs = append(labelIDs, id)
	}

	sess, err := pop3Dial(*pop3Server, *pop3Port, *pop3User, *pop3Pass)
	if err != nil {
		log.Fatalf("POP3 connect: %v", err)
	}

	count, err := sess.stat()
	if err != nil {
		log.Fatalf("POP3 stat: %v", err)
	}
	log.Printf("POP3: %d messages", count)

	var inserted []int
	for i := 1; i <= count; i++ {
		raw, err := sess.retr(i)
		if err != nil {
			log.Printf("POP3 retr %d: %v", i, err)
			continue
		}
		msg := &gmail.Message{
			Raw:      base64.URLEncoding.EncodeToString(raw),
			LabelIds: labelIDs,
		}
		_, err = svc.Users.Messages.Insert("me", msg).
			InternalDateSource("dateHeader").Do()
		if err != nil {
			log.Printf("Gmail insert %d: %v", i, err)
			continue
		}
		log.Printf("Inserted %d/%d", i, count)
		inserted = append(inserted, i)
	}

	if *deleteMsgs {
		for _, i := range inserted {
			if err := sess.dele(i); err != nil {
				log.Printf("POP3 dele %d: %v", i, err)
			}
		}
	}

	sess.quit()
	log.Printf("Done: %d/%d messages inserted", len(inserted), count)
}
