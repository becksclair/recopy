package fsprobe

import "fmt"

const btrfsSuperMagic = 0x9123683E

// Info bundles Btrfs traits for a path.
type Info struct {
	IsBtrfs     bool
	IsSubvolume bool
}

// SameDevice reports whether two paths are on the same device ID.
func SameDevice(a, b string) (bool, error) {
	da, err := deviceID(a)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", a, err)
	}
	db, err := deviceID(b)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", b, err)
	}
	return da == db, nil
}

// SupportsReflink best-effort detects reflink support for a path.
func SupportsReflink(path string) bool {
	ok, _ := tryReflink(path)
	return ok
}

// BtrfsInfo probes filesystem type and simple subvolume heuristics.
func BtrfsInfo(path string) (Info, error) {
	return probeBtrfs(path)
}
