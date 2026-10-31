// Command usage prints your Claude plan usage (the same numbers as the
// "Your usage" settings page and Claude Code's /usage) to the terminal.
//
// It reuses the OAuth login that Claude Code stores, so run `claude` and log
// in once first. The token is read from, in order:
//
//   - $CLAUDE_CODE_OAUTH_TOKEN
//   - ~/.claude/.credentials.json (Linux, Windows)
//   - the macOS keychain item "Claude Code-credentials"
//
// The usage endpoint is the undocumented one Claude Code itself calls, so it
// may change without notice.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultURL = "https://api.anthropic.com/api/oauth/usage"

type window struct {
	Utilization float64    `json:"utilization"`
	ResetsAt    *time.Time `json:"resets_at"`
}

type usage struct {
	FiveHour     *window `json:"five_hour"`
	SevenDay     *window `json:"seven_day"`
	SevenDayOpus *window `json:"seven_day_opus"`
}

type credentials struct {
	ClaudeAiOauth struct {
		AccessToken      string `json:"accessToken"`
		ExpiresAt        int64  `json:"expiresAt"` // Unix ms
		SubscriptionType string `json:"subscriptionType"`
	} `json:"claudeAiOauth"`
}

// loadCredentials finds Claude Code's stored OAuth token.
func loadCredentials() (credentials, error) {
	var c credentials
	if t := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); t != "" {
		c.ClaudeAiOauth.AccessToken = t
		return c, nil
	}

	var data []byte
	home, _ := os.UserHomeDir()
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		dir = filepath.Join(home, ".claude")
	}
	data, err := os.ReadFile(filepath.Join(dir, ".credentials.json"))
	if err != nil && runtime.GOOS == "darwin" {
		data, err = exec.Command("security", "find-generic-password",
			"-s", "Claude Code-credentials", "-w").Output()
	}
	if err != nil {
		return c, errors.New("no Claude Code login found; run `claude` and log in first")
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("parsing credentials: %w", err)
	}
	if c.ClaudeAiOauth.AccessToken == "" {
		return c, errors.New("credentials contain no claude.ai OAuth token")
	}
	if exp := c.ClaudeAiOauth.ExpiresAt; exp > 0 && time.Now().After(time.UnixMilli(exp)) {
		return c, errors.New("OAuth token has expired; run `claude` once to refresh it")
	}
	return c, nil
}

func fetchUsage(token string) (*usage, error) {
	url := os.Getenv("USAGE_URL") // for testing
	if url == "" {
		url = defaultURL
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usage request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var u usage
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("parsing usage: %w", err)
	}
	return &u, nil
}

func bar(pct, width int) string {
	n := min(pct*width/100, width)
	if pct > 0 && n == 0 {
		n = 1
	}
	return "\x1b[34m" + strings.Repeat("█", n) + "\x1b[90m" + strings.Repeat("░", width-n) + "\x1b[0m"
}

// resetText formats a reset time like the web page: a time today,
// otherwise a weekday within the next week, otherwise a date.
func resetText(t *time.Time) string {
	if t == nil {
		return ""
	}
	l, now := t.Local().Round(time.Minute), time.Now()
	switch {
	case l.YearDay() == now.YearDay() && l.Year() == now.Year():
		return "Resets at " + l.Format("3:04 PM")
	case l.Sub(now) < 7*24*time.Hour:
		return "Resets " + l.Format("Monday 3:04 PM")
	default:
		return "Resets " + l.Format("Jan 2 3:04 PM")
	}
}

func printMeter(name string, w *window) {
	if w == nil {
		return
	}
	pct := int(w.Utilization + 0.5)
	fmt.Printf("%-24s %s  %3d%% used\n", name, bar(pct, 36), pct)
	fmt.Printf("\x1b[90m%s\x1b[0m\n\n", resetText(w.ResetsAt))
}

func main() {
	creds, err := loadCredentials()
	if err == nil {
		var u *usage
		if u, err = fetchUsage(creds.ClaudeAiOauth.AccessToken); err == nil {
			render(u, creds.ClaudeAiOauth.SubscriptionType)
			return
		}
	}
	fmt.Fprintln(os.Stderr, "usage:", err)
	os.Exit(1)
}

func render(u *usage, plan string) {
	if plan != "" {
		plan = strings.ToUpper(plan[:1]) + plan[1:]
	}
	fmt.Printf("\x1b[1mYour usage\x1b[0m  \x1b[90m%s\x1b[0m\n\n", plan)
	if u.SevenDay != nil {
		fmt.Printf("\x1b[1m%d%% of your weekly limit used.\x1b[0m\n", int(u.SevenDay.Utilization+0.5))
	}
	fmt.Println(strings.Repeat("─", 72))
	printMeter("Current session", u.FiveHour)
	printMeter("This week", u.SevenDay)
	printMeter("This week (Opus)", u.SevenDayOpus)
}
