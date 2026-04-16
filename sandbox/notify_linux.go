//go:build linux

package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ── Kernel UAPI struct definitions ────────────────────────────────────────────
//
// These match the Linux kernel UAPI exactly. Sizes are verified against
// SECCOMP_GET_NOTIF_SIZES at runtime in checkNotifSizes.

// seccompData mirrors struct seccomp_data (64 bytes).
type seccompData struct {
	Nr                 int32
	Arch               uint32
	InstructionPointer uint64
	Args               [6]uint64
}

// seccompNotif mirrors struct seccomp_notif (80 bytes).
type seccompNotif struct {
	ID    uint64
	PID   uint32
	Flags uint32
	Data  seccompData
}

// seccompNotifResp mirrors struct seccomp_notif_resp (24 bytes).
type seccompNotifResp struct {
	ID    uint64
	Val   int64
	Error int32
	Flags uint32
}

// seccompNotifSizes mirrors struct seccomp_notif_sizes.
type seccompNotifSizes struct {
	Notif     uint16
	NotifResp uint16
	Data      uint16
}

// ── Runtime size check ─────────────────────────────────────────────────────

// checkNotifSizes asks the kernel for its UAPI struct sizes and verifies they
// match our Go definitions. Fails fast before any fd is used.
func checkNotifSizes() error {
	var s seccompNotifSizes
	_, _, errno := unix.Syscall(unix.SYS_SECCOMP,
		uintptr(unix.SECCOMP_GET_NOTIF_SIZES), 0,
		uintptr(unsafe.Pointer(&s)),
	)
	if errno != 0 {
		return fmt.Errorf("seccomp(GET_NOTIF_SIZES): %w", errno)
	}
	wantNotif := uint16(unsafe.Sizeof(seccompNotif{}))
	wantResp := uint16(unsafe.Sizeof(seccompNotifResp{}))
	wantData := uint16(unsafe.Sizeof(seccompData{}))
	if s.Notif != wantNotif || s.NotifResp != wantResp || s.Data != wantData {
		return fmt.Errorf("seccomp struct size mismatch: kernel{notif=%d resp=%d data=%d} Go{%d %d %d}",
			s.Notif, s.NotifResp, s.Data, wantNotif, wantResp, wantData)
	}
	return nil
}

// ── Supervisor loop ────────────────────────────────────────────────────────

const maxPath = 4096

// auditEntry is emitted to cfg.AuditLog (JSON, one line per event).
type auditEntry struct {
	Time    string `json:"time"`
	PID     int    `json:"pid"`
	Syscall string `json:"syscall"`
	Dirfd   int    `json:"dirfd,omitempty"`
	Path    string `json:"path"`
	Flags   int    `json:"flags"`
	Mode    uint32 `json:"mode,omitempty"`
	Allowed bool   `json:"allowed"`
	Note    string `json:"note,omitempty"`
}

// supervise is the main notification loop. It returns nil when the child exits
// (kernel closes the listener, RECV returns ENOENT/EBADF) or a fatal error.
func supervise(listenerFd int, cfg Config) error {
	auditW := cfg.AuditLog
	if auditW == nil {
		auditW = io.Discard
	}
	enc := json.NewEncoder(auditW)

	for {
		var notif seccompNotif
		if err := ioctlPtr(listenerFd, unix.SECCOMP_IOCTL_NOTIF_RECV,
			unsafe.Pointer(&notif)); err != nil {
			if errors.Is(err, unix.EINTR) {
				continue // retry on signal
			}
			// ENOENT: child exited; EBADF: we closed the fd after child exit.
			if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.EBADF) {
				return nil
			}
			return fmt.Errorf("NOTIF_RECV: %w", err)
		}

		req, note := decodeRequest(listenerFd, &notif)

		allowed := cfg.Authorizer(req)

		_ = enc.Encode(auditEntry{
			Time:    time.Now().UTC().Format(time.RFC3339Nano),
			PID:     req.PID,
			Syscall: syscallName(req.Nr),
			Dirfd:   req.Dirfd,
			Path:    req.Path,
			Flags:   req.Flags,
			Mode:    req.Mode,
			Allowed: allowed,
			Note:    note,
		})

		resp := seccompNotifResp{ID: notif.ID}
		if allowed {
			resp.Flags = unix.SECCOMP_USER_NOTIF_FLAG_CONTINUE
		} else {
			resp.Error = -int32(unix.EACCES)
		}

		if err := ioctlPtr(listenerFd, unix.SECCOMP_IOCTL_NOTIF_SEND,
			unsafe.Pointer(&resp)); err != nil {
			// ENOENT: child died between RECV and SEND — benign.
			if errors.Is(err, unix.ENOENT) {
				continue
			}
			return fmt.Errorf("NOTIF_SEND: %w", err)
		}
	}
}

// ── Request decoding ───────────────────────────────────────────────────────

func decodeRequest(listenerFd int, notif *seccompNotif) (OpenRequest, string) {
	nr := int(notif.Data.Nr)
	pid := int(notif.PID)
	req := OpenRequest{PID: pid, Nr: nr, Dirfd: -1}

	var pathAddr uintptr
	switch nr {
	case sysOpenAt:
		req.Dirfd = int(int32(notif.Data.Args[0]))
		pathAddr = uintptr(notif.Data.Args[1])
		req.Flags = int(int32(notif.Data.Args[2]))
		req.Mode = uint32(notif.Data.Args[3])
	case sysOpen:
		pathAddr = uintptr(notif.Data.Args[0])
		req.Flags = int(int32(notif.Data.Args[1]))
		req.Mode = uint32(notif.Data.Args[2])
	default:
		return req, fmt.Sprintf("unexpected syscall nr=%d", nr)
	}

	path, note := readPath(listenerFd, pid, notif.ID, pathAddr)
	req.Path = path
	return req, note
}

// readPath reads a NUL-terminated path from the target's address space.
// It validates the notification cookie before and after the read, per the
// recommendation in man 2 seccomp_unotify, to guard against PID reuse.
func readPath(listenerFd, pid int, cookie uint64, addr uintptr) (string, string) {
	memPath := fmt.Sprintf("/proc/%d/mem", pid)
	f, err := os.OpenFile(memPath, os.O_RDONLY, 0)
	if err != nil {
		return "", fmt.Sprintf("open %s: %v", memPath, err)
	}
	defer f.Close()

	// First validity check.
	if err := notifIDValid(listenerFd, cookie); err != nil {
		return "", fmt.Sprintf("id_valid(pre): %v", err)
	}

	buf := make([]byte, maxPath)
	n, err := f.ReadAt(buf, int64(addr))
	if err != nil && !errors.Is(err, io.EOF) && n == 0 {
		return "", fmt.Sprintf("read /proc/%d/mem: %v", pid, err)
	}

	// Second validity check: confirm the notification is still live so we know
	// the bytes we read belong to this process and not a recycled PID.
	if err := notifIDValid(listenerFd, cookie); err != nil {
		return "", fmt.Sprintf("id_valid(post): %v", err)
	}

	// Truncate at first NUL.
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return string(buf[:i]), ""
		}
	}
	return "", "path not NUL-terminated within PATH_MAX"
}

// ── ioctl helpers ──────────────────────────────────────────────────────────

func notifIDValid(listenerFd int, cookie uint64) error {
	return ioctlPtr(listenerFd, unix.SECCOMP_IOCTL_NOTIF_ID_VALID,
		unsafe.Pointer(&cookie))
}

func ioctlPtr(fd int, req uint, ptr unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		uintptr(fd), uintptr(req), uintptr(ptr))
	if errno != 0 {
		return errno
	}
	return nil
}

// ── Utilities ──────────────────────────────────────────────────────────────

func syscallName(nr int) string {
	switch nr {
	case sysOpenAt:
		return "openat"
	case sysOpen:
		return "open"
	default:
		return fmt.Sprintf("syscall_%d", nr)
	}
}
