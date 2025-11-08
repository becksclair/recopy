//go:build linux || darwin

package fsprobe

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func deviceID(path string) (uint64, error) {
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}

func tryReflink(path string) (bool, error) {
	dir := path
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			dir = filepath.Dir(path)
		}
	} else {
		dir = filepath.Dir(path)
	}

	if dir == "" {
		dir = "."
	}

	src, err := os.CreateTemp(dir, ".recopy-reflink-src")
	if err != nil {
		return false, err
	}
	defer os.Remove(src.Name())
	defer src.Close()

	if _, err = src.Write([]byte("recopy")); err != nil {
		return false, err
	}

	dst, err := os.CreateTemp(dir, ".recopy-reflink-dst")
	if err != nil {
		return false, err
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err := unix.IoctlFileClone(int(dst.Fd()), int(src.Fd())); err != nil {
		return false, err
	}
	return true, nil
}

func probeBtrfs(path string) (Info, error) {
	var info Info
	var fs unix.Statfs_t
	if err := unix.Statfs(path, &fs); err != nil {
		return info, err
	}
	if uint64(fs.Type) != btrfsSuperMagic {
		return info, nil
	}
	info.IsBtrfs = true

	// Subvolume heuristic: inode number 256 typically indicates subvolume roots.
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return info, err
	}
	if st.Ino == 256 {
		info.IsSubvolume = true
	}
	return info, nil
}
