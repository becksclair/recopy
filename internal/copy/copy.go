package copy

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrNotSupported indicates the operation is not supported on this platform/filesystem
	ErrNotSupported = errors.New("copy operation not supported")
)

// FileInfo wraps os.FileInfo with additional metadata needed for copying
type FileInfo interface {
	os.FileInfo
}

// Options controls copy behavior
type Options struct {
	// PreserveAll preserves permissions, ownership, and timestamps
	PreserveAll bool
	// Sparse enables sparse file handling
	Sparse bool
}

// File copies a single file from src to dest using the most efficient method available.
// It attempts reflink first, then copy_file_range, then sendfile, falling back to
// buffered I/O if necessary. Metadata is preserved according to opts.
func File(src, dest string, info FileInfo, opts Options) error {
	if info.IsDir() {
		return fmt.Errorf("cannot copy directory as file: %s", src)
	}

	// Try reflink (instant CoW clone on supported filesystems)
	if err := tryReflink(src, dest, info); err == nil {
		if opts.PreserveAll {
			return preserveMetadata(dest, info)
		}
		return nil
	} else if !errors.Is(err, ErrNotSupported) {
		return fmt.Errorf("reflink %s: %w", src, err)
	}

	// Try copy_file_range (kernel zero-copy)
	if err := tryCopyFileRange(src, dest, info, opts); err == nil {
		if opts.PreserveAll {
			return preserveMetadata(dest, info)
		}
		return nil
	} else if !errors.Is(err, ErrNotSupported) {
		return fmt.Errorf("copy_file_range %s: %w", src, err)
	}

	// Try sendfile as another zero-copy option before falling back to buffered copy
	if err := trySendfile(src, dest, info); err == nil {
		if opts.PreserveAll {
			return preserveMetadata(dest, info)
		}
		return nil
	} else if !errors.Is(err, ErrNotSupported) {
		return fmt.Errorf("sendfile %s: %w", src, err)
	}

	// Fall back to buffered copy
	if err := bufferedCopy(src, dest, info, opts); err != nil {
		return fmt.Errorf("buffered copy %s: %w", src, err)
	}

	if opts.PreserveAll {
		return preserveMetadata(dest, info)
	}
	return nil
}

// bufferedCopy performs a traditional read/write loop with a large buffer
func bufferedCopy(src, dest string, info FileInfo, opts Options) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Use 1 MiB buffer for efficiency
	buf := make([]byte, 1<<20)
	if _, err := io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return err
	}

	return dstFile.Sync()
}
