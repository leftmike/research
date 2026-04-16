//go:build linux

package sandbox

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// seccomp_data layout (man 2 seccomp):
//   int   nr;                   // offset 0 (u32)
//   __u32 arch;                 // offset 4
//   __u64 instruction_pointer;  // offset 8
//   __u64 args[6];              // offset 16
const (
	offNR   = 0
	offArch = 4
)

// retErrnoENOSYS encodes a seccomp return that delivers ENOSYS to the caller.
// SECCOMP_RET_ERRNO's low 16 bits carry the errno value.
const retErrnoENOSYS = unix.SECCOMP_RET_ERRNO | uint32(unix.ENOSYS)

// nativeArch is the AUDIT_ARCH_* value for this build's target architecture.
// Defined per-arch in arch_linux_*.go.
const nativeArch = nativeArchValue

// buildFilter returns the BPF program that routes openat/open to the
// user-notify mechanism and allows all other syscalls.
//
// Program layout:
//   0: LD  [arch]
//   1: JEQ #nativeArch, jt=1, jf=0   ; match → skip to 3, else fall to 2
//   2: RET #ERRNO|ENOSYS
//   3: LD  [nr]
//   4: JEQ #sysOpenAt, jt=2, jf=0    ; match → skip to 7
//   5: JEQ #sysOpen,   jt=1, jf=0    ; match → skip to 7
//   6: RET #RET_ALLOW
//   7: RET #RET_USER_NOTIF
func buildFilter() []unix.SockFilter {
	const (
		ld  = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
		jeq = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
		ret = unix.BPF_RET | unix.BPF_K
	)
	// Bounce through typed variables to convert potentially-negative constants
	// to uint32 without triggering Go's compile-time overflow check.
	// On arm64, sysOpen == -1 → 0xFFFFFFFF, which never matches a real nr.
	var openAt32, open32 int32 = int32(sysOpenAt), int32(sysOpen)
	openAtK := uint32(openAt32)
	openK := uint32(open32)

	return []unix.SockFilter{
		{Code: ld, K: offArch},
		{Code: jeq, Jt: 1, Jf: 0, K: nativeArch},
		{Code: ret, K: retErrnoENOSYS},
		{Code: ld, K: offNR},
		{Code: jeq, Jt: 2, Jf: 0, K: openAtK},
		{Code: jeq, Jt: 1, Jf: 0, K: openK},
		{Code: ret, K: unix.SECCOMP_RET_ALLOW},
		{Code: ret, K: unix.SECCOMP_RET_USER_NOTIF},
	}
}

// installListener installs the seccomp filter on the calling thread using
// SECCOMP_FILTER_FLAG_NEW_LISTENER and returns the resulting notifier fd.
// PR_SET_NO_NEW_PRIVS must have been set on the same thread beforehand.
func installListener() (int, error) {
	filter := buildFilter()
	prog := unix.SockFprog{
		Len:    uint16(len(filter)),
		Filter: &filter[0],
	}
	r1, _, errno := unix.Syscall(unix.SYS_SECCOMP,
		uintptr(unix.SECCOMP_SET_MODE_FILTER),
		uintptr(unix.SECCOMP_FILTER_FLAG_NEW_LISTENER),
		uintptr(unsafe.Pointer(&prog)),
	)
	if errno != 0 {
		return -1, fmt.Errorf("seccomp(SET_MODE_FILTER, NEW_LISTENER): %w", errno)
	}
	return int(r1), nil
}
