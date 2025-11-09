package mv

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Rename moves src into dest, creating the destination parent tree when needed.
func Rename(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dest)
}

// PruneEmptyDirs removes empty directories rooted at each path, walking bottom-up.
func PruneEmptyDirs(paths []string) error {
	for _, root := range paths {
		if err := pruneRoot(root); err != nil {
			return err
		}
	}
	return nil
}

func pruneRoot(root string) error {
	info, err := os.Lstat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	var dirs []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		if err := os.Remove(dir); err != nil {
			if shouldIgnoreRemoveError(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func shouldIgnoreRemoveError(err error) bool {
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	var pe *os.PathError
	if errors.As(err, &pe) {
		if pe.Err == syscall.ENOTEMPTY || pe.Err == syscall.EEXIST {
			return true
		}
	}
	return false
}
