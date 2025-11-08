//go:build !linux && !darwin

package fsprobe

import "errors"

func deviceID(path string) (uint64, error) {
	return 0, errors.New("fsprobe: unsupported OS")
}

func tryReflink(path string) (bool, error) {
	return false, errors.New("fsprobe: unsupported OS")
}

func probeBtrfs(path string) (Info, error) {
	return Info{}, errors.New("fsprobe: unsupported OS")
}
