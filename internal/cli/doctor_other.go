//go:build !linux

package cli

// probeCopyFileRange reports whether the copy_file_range syscall is available on the current system.
// On non-Linux builds this always returns false and a nil error.
// The boolean is true if the syscall is supported; the error is non-nil only if probing fails.
func probeCopyFileRange() (bool, error) {
	return false, nil
}

// probeIOUring reports whether io_uring is available on the current platform.
// On non-Linux builds this always returns false and a nil error.
func probeIOUring() (bool, error) {
	return false, nil
}