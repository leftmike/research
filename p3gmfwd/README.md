# p3gmfwd

Fetches emails from a POP3 server and inserts them into a Gmail account using the Google Gmail API.

## Setup

1. Create a [Google Cloud project](https://console.cloud.google.com/) with the Gmail API enabled.
2. Create an OAuth2 **Desktop app** credential and download it as `credentials.json`.
3. Build: `go build`

## Usage

```
p3gmfwd --pop3-server mail.example.com --pop3-user user@example.com --pop3-pass secret
```

Flags:
- `--pop3-server` — POP3 server hostname (required)
- `--pop3-user` — POP3 username (required)
- `--pop3-pass` — POP3 password (required)
- `--pop3-port` — POP3 port (default: 995, TLS)
- `--delete` — delete messages from POP3 after successful insert
- `--label` — Gmail label to apply (created if it doesn't exist)
- `--credentials` — path to OAuth2 credentials file (default: credentials.json)
- `--token` — path to cached OAuth2 token (default: token.json)

On first run, you'll be prompted to visit a URL and paste an authorization code to complete the OAuth2 flow. The token is cached in `token.json` for subsequent runs.
