//go:build !linux

package copy

import (
	"os"
)

// tryReflink is not available on non-Linux platforms
func tryReflink(src, dest string, info FileInfo) error {
	return ErrNotSupported
}

// tryCopyFileRange is not available on non-Linux platforms
func tryCopyFileRange(src, dest string, info FileInfo, opts Options) error {
	return ErrNotSupported
}

// preserveMetadata sets permissions and timestamps on non-Linux platforms
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
