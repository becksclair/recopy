//go:build !linux

package copy

import (
	"os"
)

// tryReflink attempts to create a reflink copy but is not supported on non-Linux platforms and always returns ErrNotSupported.
func tryReflink(src, dest string, info FileInfo) error {
	return ErrNotSupported
}

// tryCopyFileRange reports that file-range copy is not supported on non-Linux platforms.
// It always returns ErrNotSupported.
func tryCopyFileRange(src, dest string, info FileInfo, opts Options) error {
	return ErrNotSupported
}

// preserveMetadata sets the file mode and access/modification times of dest from info.
// It applies permissions via os.Chmod and updates both access and modification times to
// info.ModTime using os.Chtimes. Returns the first error encountered from these operations.
func preserveMetadata(dest string, info FileInfo) error {
	// Set permissions
	if err := os.Chmod(dest, info.Mode()); err != nil {
		return err
	}

	// Set timestamps (best effort)
	mtime := info.ModTime()
	if err := os.Chtimes(dest, mtime, mtime); err != nil {
		return err
	}

	return nil
}