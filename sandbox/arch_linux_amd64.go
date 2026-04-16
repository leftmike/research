//go:build linux && amd64

package sandbox

import "golang.org/x/sys/unix"

const (
	nativeArchValue = uint32(unix.AUDIT_ARCH_X86_64)
	sysOpenAt       = int(unix.SYS_OPENAT)
	sysOpen         = int(unix.SYS_OPEN)
)
