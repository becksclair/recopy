//go:build linux

package copy

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// tryReflink attempts a reflink clone (CoW copy) using FICLONE
func tryReflink(src, dest string, info FileInfo) error {
	// Remove dest if it exists
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Attempt FICLONE ioctl
	err = unix.IoctlFileClone(int(dstFile.Fd()), int(srcFile.Fd()))
	if err != nil {
		// Remove partial dest on failure
		os.Remove(dest)
		if isReflinkUnsupported(err) {
			return ErrNotSupported
		}
		return err
	}

	return dstFile.Sync()
}

// tryCopyFileRange uses the copy_file_range syscall for zero-copy
func tryCopyFileRange(src, dest string, info FileInfo, opts Options) error {
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

	size := info.Size()
	var offset int64

	for offset < size {
		// copy_file_range can return partial copies, so loop until done
		// Use 1 GiB chunks to avoid hitting kernel limits
		remain := size - offset
		chunkSize := int(remain)
		if chunkSize > 1<<30 {
			chunkSize = 1 << 30
		}

		n, err := unix.CopyFileRange(int(srcFile.Fd()), &offset, int(dstFile.Fd()), nil, chunkSize, 0)
		if err != nil {
			// Remove partial dest on failure
			os.Remove(dest)
			if isCopyFileRangeUnsupported(err) {
				return ErrNotSupported
			}
			return err
		}

		if n == 0 {
			// Should not happen, but guard against infinite loop
			break
		}
	}

	return dstFile.Sync()
}

// preserveMetadata sets ownership, permissions, and timestamps on dest to match info
func preserveMetadata(dest string, info FileInfo) error {
	// Set ownership (may fail if not root; that's OK)
	if stat, ok := info.Sys().(*unix.Stat_t); ok {
		_ = os.Chown(dest, int(stat.Uid), int(stat.Gid))
	}

	// Set permissions
	if err := os.Chmod(dest, info.Mode()); err != nil {
		return err
	}

	// Set timestamps
	if stat, ok := info.Sys().(*unix.Stat_t); ok {
		atime := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
		mtime := time.Unix(int64(stat.Mtim.Sec), int64(stat.Mtim.Nsec))
		if err := os.Chtimes(dest, atime, mtime); err != nil {
			return err
		}
	}

	return nil
}

func isReflinkUnsupported(err error) bool {
	return errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EXDEV) || errors.Is(err, unix.ENOTTY)
}

func isCopyFileRangeUnsupported(err error) bool {
	// ENOSYS: syscall not implemented (old kernel)
	// EXDEV: cross-device copy not supported
	// EOPNOTSUPP: filesystem doesn't support it
	// EINVAL: invalid arguments (sometimes means unsupported)
	return errors.Is(err, unix.ENOSYS) ||
		errors.Is(err, unix.EXDEV) ||
		errors.Is(err, unix.EOPNOTSUPP) ||
		errors.Is(err, unix.EINVAL)
}

// trySendfile uses sendfile syscall as another zero-copy option
// This is an alternative to copy_file_range for older kernels
func trySendfile(src, dest string, info FileInfo) error {
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

	size := info.Size()
	var offset int64

	for offset < size {
		// sendfile can return partial transfers
		remain := size - offset
		chunkSize := int(remain)
		if chunkSize > 1<<30 {
			chunkSize = 1 << 30
		}

		n, err := unix.Sendfile(int(dstFile.Fd()), int(srcFile.Fd()), &offset, chunkSize)
		if err != nil {
			os.Remove(dest)
			if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EINVAL) {
				return ErrNotSupported
			}
			return err
		}

		if n == 0 {
			break
		}
	}

	return dstFile.Sync()
}

// sparseReader wraps io.Reader to detect and skip sparse regions (reserved for future use)
type sparseReader struct {
	file *os.File
}

// Read implements io.Reader, using SEEK_DATA/SEEK_HOLE to skip sparse regions
func (sr *sparseReader) Read(p []byte) (n int, err error) {
	// For now, just delegate to regular Read
	// Future optimization: use SEEK_DATA/SEEK_HOLE via unix.Seek with SEEK_DATA constant
	return sr.file.Read(p)
}
