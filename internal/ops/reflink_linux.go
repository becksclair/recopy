//go:build linux

package ops

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

func reflinkCopy(src, dest string, info os.FileInfo) error {
	if err := mkdirAll(filepath.Dir(dest)); err != nil {
		return err
	}
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
	if err := unix.IoctlFileClone(int(dstFile.Fd()), int(srcFile.Fd())); err != nil {
		if isReflinkUnsupported(err) {
			return errReflinkUnsupported
		}
		return err
	}
	return preserveMetadata(dest, info)
}

func preserveMetadata(dest string, info os.FileInfo) error {
	stat, ok := info.Sys().(*unix.Stat_t)
	if ok {
		if err := os.Chown(dest, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}
	}
	if err := os.Chmod(dest, info.Mode()); err != nil {
		return err
	}
	if ok {
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
