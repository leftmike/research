package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// codexCmd returns the codex command and arguments prefix.
// It tries "codex" first, then falls back to "npx @openai/codex".
func codexCmd(t *testing.T) (string, []string) {
	t.Helper()
	if path, err := exec.LookPath("codex"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("npx"); err == nil {
		return path, []string{"@openai/codex"}
	}
	t.Fatal("codex CLI not found: install via npm install -g @openai/codex")
	return "", nil
}

// runCodex runs the codex CLI with the given arguments.
func runCodex(t *testing.T, args ...string) (string, error) {
	t.Helper()
	bin, prefix := codexCmd(t)
	fullArgs := append(prefix, args...)
	cmd := exec.Command(bin, fullArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// buildBinary builds the sysmcp binary and returns its absolute path.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	binary := filepath.Join(dir, "sysmcp")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binary
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set; skipping e2e tests")
	}

	binary := buildBinary(t)
	mcpName := "sysmcp-e2e-test"

	// Defensively remove any prior registration.
	runCodex(t, "mcp", "remove", mcpName)

	// Register the MCP server.
	t.Logf("Registering MCP server %s -> %s", mcpName, binary)
	if out, err := runCodex(t, "mcp", "add", mcpName, "--", binary); err != nil {
		t.Fatalf("mcp add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		runCodex(t, "mcp", "remove", mcpName)
	})

	tests := []struct {
		name    string
		tool    string
		pattern string
	}{
		{
			name:    "date",
			tool:    "date",
			pattern: `\d{4}-\d{2}-\d{2}`,
		},
		{
			name:    "time",
			tool:    "time",
			pattern: `\d{2}:\d{2}:\d{2}`,
		},
		{
			name:    "os",
			tool:    "os",
			pattern: `(?i)OS:`,
		},
		{
			name:    "hardware",
			tool:    "hardware",
			pattern: `(?i)(CPU:|Cores:|RAM:)`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "output.txt")
			prompt := "Use the " + tc.tool + " tool from the " + mcpName +
				" MCP server. Reply with ONLY the tool output, nothing else."

			out, err := runCodex(t,
				"exec",
				"--skip-git-repo-check",
				"--dangerously-bypass-approvals-and-sandbox",
				"-o", outFile,
				prompt,
			)
			if err != nil {
				t.Fatalf("codex exec failed: %v\n%s", err, out)
			}

			data, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("reading output file: %v", err)
			}

			output := strings.TrimSpace(string(data))
			if output == "" {
				t.Fatal("codex exec produced empty output")
			}

			t.Logf("Output: %s", output)

			re := regexp.MustCompile(tc.pattern)
			if !re.MatchString(output) {
				t.Errorf("output %q did not match pattern %q", output, tc.pattern)
			}
		})
	}
}
