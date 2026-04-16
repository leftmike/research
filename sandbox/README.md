# sandbox

A minimal Linux sandbox for Go programs. Runs a target program under a seccomp
user-notify filter so its file-open syscalls can be **audited** and
**dynamically authorized** by a supervisor in the calling process.

v1 intercepts `openat` and `open` only. Allowed syscalls are completed
normally by the kernel; denied syscalls return `EACCES`. An optional
JSON-lines audit log records every decision.

## Requirements

| | |
|---|---|
| Linux kernel | **5.5+** (`SECCOMP_RET_USER_NOTIF` + `SECCOMP_USER_NOTIF_FLAG_CONTINUE`) |
| Architecture | `amd64` or `arm64` |
| Privileges | None — uses `PR_SET_NO_NEW_PRIVS`, no `CAP_SYS_ADMIN` |
| CGO | Not required |

## Quick start (demo CLI)

```sh
go build ./cmd/sbx

# Allow only /etc/ paths:
./sbx -allow '^/etc/' cat /etc/hostname

# Deny everything:
./sbx -deny '.*' cat /etc/hostname        # → Permission denied

# Interactive y/n prompt per open:
./sbx -interactive cat /etc/hostname

# Allow-all with audit log to file:
./sbx -audit audit.log cat /etc/hostname
```

`sbx` flags:

| Flag | Description |
|------|-------------|
| `-allow REGEX` | Allow opens whose path matches; deny all others |
| `-deny REGEX` | Deny opens whose path matches; allow all others |
| `-interactive` | Prompt y/n on `/dev/tty` for each open |
| `-audit FILE` | Write JSON-line audit log to this file (default: stderr) |

## Using the library

```go
package main

import (
    "os"
    "strings"
    "github.com/leftmike/research/sandbox"
)

func main() {
    // Must be the very first call in main().
    sandbox.Init()

    code, err := sandbox.Run(sandbox.Config{
        Program: "/bin/cat",
        Args:    []string{"cat", "/etc/hostname"},
        Authorizer: func(r sandbox.OpenRequest) bool {
            return strings.HasPrefix(r.Path, "/etc/")
        },
        AuditLog: os.Stderr,
    })
    if err != nil {
        panic(err)
    }
    os.Exit(code)
}
```

`sandbox.Init()` detects the re-exec'd child-init mode via the
`__SANDBOX_INIT=1` environment variable set by the parent. In the parent it
returns immediately; in the child it installs the filter and execs the target.

### Audit log format

One JSON object per line:

```json
{"time":"2026-01-01T00:00:00Z","pid":1234,"syscall":"openat","dirfd":-100,"path":"/etc/hostname","flags":0,"allowed":true}
```

## How it works

```
parent                              child (re-exec)                 target
──────                              ───────────────                 ──────
Run(cfg):
  socketpair()
  exec.Cmd("/proc/self/exe") ──►  Init():
  recv listener fd ◄───────────     LockOSThread
  checkNotifSizes                    prctl(NO_NEW_PRIVS)
  supervise loop:                    seccomp(SET_MODE_FILTER,
    NOTIF_RECV                               FLAG_NEW_LISTENER) → fd
    open /proc/pid/mem               sendmsg(SCM_RIGHTS, fd) ──►
    NOTIF_ID_VALID × 2               close sock, fd
    Authorizer(req)                  exec(target) ────────────────► running
    write audit entry
    NOTIF_SEND (CONTINUE|EACCES)
  cmd.Wait()
```

The supervisor exits when the child dies and the kernel closes the listener
(RECV returns `ENOENT`).

## Limitations

- **Not a security sandbox against hostile code.** `SECCOMP_USER_NOTIF_FLAG_CONTINUE`
  is documented as unsafe for security enforcement (see `man 2 seccomp_unotify`):
  a multi-threaded target can rewrite the path argument between the time the
  supervisor reads it from `/proc/<pid>/mem` and when the kernel dispatches the
  syscall. Use this package for **auditing** and cooperative authorization, not
  to contain a malicious workload.

- **Relative paths are not resolved.** The audit log records the raw
  `(dirfd, path)` pair. For openat with a non-`AT_FDCWD` dirfd, an authorizer
  that needs the absolute path must resolve `/proc/<pid>/fd/<dirfd>` itself.

- **Only `openat` and `open` are intercepted.** Other file-creating calls
  (`openat2`, `creat`, `memfd_create`, etc.) pass through unfiltered.

- **One `Run` at a time per process.** The parent re-execs `/proc/self/exe`
  and uses fd 3 for the socketpair; concurrent `Run` calls from the same
  process would collide.

## File layout

```
sandbox.go            Public API: Config, OpenRequest, Authorizer
run_linux.go          Run() — socketpair, re-exec, recv fd, supervise, wait
child_linux.go        childInit() — NO_NEW_PRIVS, install filter, sendmsg, exec
filter_linux.go       buildFilter() BPF + installListener() via SYS_SECCOMP
notify_linux.go       supervise loop, ioctl wrappers, /proc/pid/mem reader
arch_linux_amd64.go   nativeArch = AUDIT_ARCH_X86_64, sysOpen/sysOpenAt
arch_linux_arm64.go   nativeArch = AUDIT_ARCH_AARCH64, sysOpenAt (no SYS_OPEN)
sandbox_other.go      !linux stub (ErrUnsupported)
cmd/sbx/main.go       Demo CLI
internal/testhelper/  Binary compiled by integration tests
```
