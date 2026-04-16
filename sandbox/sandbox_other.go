//go:build !linux

package sandbox

import "errors"

// Init is a no-op on non-Linux systems.
func Init() {}

// Run is unsupported on non-Linux systems.
func Run(cfg Config) (int, error) {
	return 0, errors.New("sandbox: only supported on Linux")
}
