//go:build linux && arm64

package sandbox

import "golang.org/x/sys/unix"

const (
	nativeArchValue = uint32(unix.AUDIT_ARCH_AARCH64)
	sysOpenAt       = int(unix.SYS_OPENAT)
	// arm64 has no legacy open(2) syscall; use -1 as a sentinel that never
	// matches a real syscall number in the BPF filter.
	sysOpen = -1
)
