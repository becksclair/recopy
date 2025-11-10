//go:build linux

package cli

import (
	"os"

	"golang.org/x/sys/unix"
)

// probeCopyFileRange tests if copy_file_range syscall is available
func probeCopyFileRange() (bool, error) {
	// Create two temporary files to test the syscall
	src, err := os.CreateTemp("", "recopy-probe-src-*")
	if err != nil {
		return false, nil // not an error, just unavailable
	}
	defer os.Remove(src.Name())
	defer src.Close()

	// Write test data
	if _, err := src.Write([]byte("test")); err != nil {
		return false, nil
	}
	if _, err := src.Seek(0, 0); err != nil {
		return false, nil
	}

	dst, err := os.CreateTemp("", "recopy-probe-dst-*")
	if err != nil {
		return false, nil
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	// Try copy_file_range
	_, err = unix.CopyFileRange(int(src.Fd()), nil, int(dst.Fd()), nil, 4, 0)
	if err == nil {
		return true, nil
	}

	// ENOSYS means syscall not available (old kernel)
	// EXDEV, EOPNOTSUPP might occur but syscall exists
	if err == unix.ENOSYS {
		return false, nil
	}

	// Syscall exists but failed for other reasons (e.g., fs limitation)
	// Consider it supported at kernel level
	return true, nil
}

// probeIOUring tests if io_uring is available
func probeIOUring() (bool, error) {
	// Check for io_uring availability via syscall (syscall 425 on x86_64)
	// We just check if the syscall exists, not if it succeeds
	// This is a simple heuristic: kernel >= 5.1 has io_uring

	// For now, return false as a conservative estimate
	// Full io_uring support requires liburing bindings
	// which are not in the MVP scope
	return false, nil
}
