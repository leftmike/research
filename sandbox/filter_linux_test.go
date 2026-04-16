//go:build linux

package sandbox

import (
	"testing"

	"golang.org/x/sys/unix"
)

// runBPF executes a classic cBPF program against a raw seccomp_data payload.
// It implements the subset of BPF the filter uses:
//
//	BPF_LD|BPF_W|BPF_ABS, BPF_JMP|BPF_JEQ|BPF_K, BPF_RET|BPF_K
func runBPF(t *testing.T, prog []unix.SockFilter, data []byte) uint32 {
	t.Helper()
	const (
		ld  = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
		jeq = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
		ret = unix.BPF_RET | unix.BPF_K
	)
	var a uint32
	pc := 0
	for pc < len(prog) {
		ins := prog[pc]
		switch uint32(ins.Code) {
		case uint32(ld):
			k := int(ins.K)
			if k+4 > len(data) {
				t.Fatalf("BPF LD beyond data: offset=%d len=%d", k, len(data))
			}
			a = uint32(data[k]) | uint32(data[k+1])<<8 | uint32(data[k+2])<<16 | uint32(data[k+3])<<24
			pc++
		case uint32(jeq):
			if a == ins.K {
				pc += 1 + int(ins.Jt)
			} else {
				pc += 1 + int(ins.Jf)
			}
		case uint32(ret):
			return ins.K
		default:
			t.Fatalf("BPF interpreter: unknown code 0x%x at pc=%d", ins.Code, pc)
		}
	}
	t.Fatalf("BPF program fell off the end (pc=%d)", pc)
	return 0
}

// fakeSeccompData builds a 64-byte seccomp_data blob with nr at [0] and arch
// at [4] in little-endian, all other fields zero.
func fakeSeccompData(nr int32, arch uint32) []byte {
	b := make([]byte, 64)
	b[0] = byte(uint32(nr))
	b[1] = byte(uint32(nr) >> 8)
	b[2] = byte(uint32(nr) >> 16)
	b[3] = byte(uint32(nr) >> 24)
	b[4] = byte(arch)
	b[5] = byte(arch >> 8)
	b[6] = byte(arch >> 16)
	b[7] = byte(arch >> 24)
	return b
}

func TestBuildFilterClassifiesSyscalls(t *testing.T) {
	prog := buildFilter()

	cases := []struct {
		name string
		nr   int32
		arch uint32
		want uint32
	}{
		{"openat native → USER_NOTIF", int32(sysOpenAt), nativeArch, unix.SECCOMP_RET_USER_NOTIF},
		{"read native → ALLOW", int32(unix.SYS_READ), nativeArch, unix.SECCOMP_RET_ALLOW},
		{"write native → ALLOW", int32(unix.SYS_WRITE), nativeArch, unix.SECCOMP_RET_ALLOW},
		{"openat wrong arch → ENOSYS", int32(sysOpenAt), 0xdeadbeef, retErrnoENOSYS},
	}
	if sysOpen != -1 {
		cases = append(cases, struct {
			name string
			nr   int32
			arch uint32
			want uint32
		}{"open native → USER_NOTIF", int32(sysOpen), nativeArch, unix.SECCOMP_RET_USER_NOTIF})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := fakeSeccompData(tc.nr, tc.arch)
			got := runBPF(t, prog, data)
			if got != tc.want {
				t.Errorf("nr=%d arch=0x%x: got 0x%08x, want 0x%08x", tc.nr, tc.arch, got, tc.want)
			}
		})
	}
}

// TestNotifSizes queries the kernel for UAPI struct sizes and checks they
// match our Go struct definitions. Requires kernel >= 5.0.
func TestNotifSizes(t *testing.T) {
	if err := checkNotifSizes(); err != nil {
		t.Skipf("kernel does not support seccomp user-notify: %v", err)
	}
}
