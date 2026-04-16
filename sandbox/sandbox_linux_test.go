//go:build linux

package sandbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildTestHelper compiles ./internal/testhelper into a temp binary and returns
// its path. The binary is removed when the test run ends.
func buildTestHelper(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "helper")
	cmd := exec.Command("go", "build", "-o", bin, "./internal/testhelper")
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build helper: %v\n%s", err, out)
	}
	return bin
}

// kernelHasUserNotify returns true when the running kernel supports seccomp
// user-notify (kernel >= 5.0 for GET_NOTIF_SIZES, >= 5.5 for FLAG_CONTINUE).
func kernelHasUserNotify() bool {
	return checkNotifSizes() == nil
}

func parseAudit(t *testing.T, raw []byte) []auditEntry {
	t.Helper()
	var out []auditEntry
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		var e auditEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("bad audit JSON %q: %v", sc.Text(), err)
		}
		out = append(out, e)
	}
	return out
}

func runHelper(t *testing.T, bin string, auth Authorizer, paths ...string) (stdout string, entries []auditEntry, code int) {
	t.Helper()
	var outBuf, auditBuf bytes.Buffer
	c, err := Run(Config{
		Program:    bin,
		Args:       append([]string{bin}, paths...),
		Authorizer: auth,
		AuditLog:   &auditBuf,
		Stdout:     &outBuf,
		Stderr:     os.Stderr,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return outBuf.String(), parseAudit(t, auditBuf.Bytes()), c
}

func containsAudit(entries []auditEntry, path string, allowed bool) bool {
	for _, e := range entries {
		if e.Path == path && e.Allowed == allowed {
			return true
		}
	}
	return false
}

func TestAllowAll(t *testing.T) {
	if !kernelHasUserNotify() {
		t.Skip("kernel lacks seccomp user-notify")
	}
	bin := buildTestHelper(t)
	stdout, entries, code := runHelper(t, bin,
		func(OpenRequest) bool { return true },
		"/etc/hostname", "/etc/hosts",
	)
	if code != 0 {
		t.Fatalf("helper exited %d", code)
	}
	for _, p := range []string{"/etc/hostname", "/etc/hosts"} {
		if !strings.Contains(stdout, "OK:"+p) {
			t.Errorf("expected OK for %s; stdout:\n%s", p, stdout)
		}
		if !containsAudit(entries, p, true) {
			t.Errorf("audit log missing allowed entry for %s", p)
		}
	}
}

func TestDenyAll(t *testing.T) {
	if !kernelHasUserNotify() {
		t.Skip("kernel lacks seccomp user-notify")
	}
	bin := buildTestHelper(t)
	stdout, entries, _ := runHelper(t, bin,
		func(OpenRequest) bool { return false },
		"/etc/hostname",
	)
	if !strings.Contains(stdout, "ERR:/etc/hostname:") {
		t.Fatalf("expected ERR line; stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "permission denied") {
		t.Fatalf("expected 'permission denied' in output; stdout:\n%s", stdout)
	}
	if !containsAudit(entries, "/etc/hostname", false) {
		t.Errorf("audit log missing deny entry for /etc/hostname: %+v", entries)
	}
}

func TestSelectiveAuth(t *testing.T) {
	if !kernelHasUserNotify() {
		t.Skip("kernel lacks seccomp user-notify")
	}
	bin := buildTestHelper(t)
	stdout, entries, _ := runHelper(t, bin,
		func(r OpenRequest) bool { return r.Path == "/etc/hostname" },
		"/etc/hostname", "/etc/hosts",
	)
	if !strings.Contains(stdout, "OK:/etc/hostname") {
		t.Fatalf("expected /etc/hostname to be allowed; stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "ERR:/etc/hosts:") {
		t.Fatalf("expected /etc/hosts to be denied; stdout:\n%s", stdout)
	}
	if !containsAudit(entries, "/etc/hostname", true) {
		t.Errorf("audit missing allow for /etc/hostname")
	}
	if !containsAudit(entries, "/etc/hosts", false) {
		t.Errorf("audit missing deny for /etc/hosts")
	}
}

func TestNonexistentProgram(t *testing.T) {
	_, err := Run(Config{
		Program: "/nonexistent/definitely-not-here",
		Args:    []string{"x"},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent program, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) && !strings.Contains(err.Error(), "start") {
		t.Logf("(acceptable error shape) %v", err)
	}
}
