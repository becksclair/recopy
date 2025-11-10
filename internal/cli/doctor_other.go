//go:build !linux

package cli

// probeCopyFileRange is not available on non-Linux platforms
func probeCopyFileRange() (bool, error) {
	return false, nil
}

// probeIOUring is not available on non-Linux platforms
func probeIOUring() (bool, error) {
	return false, nil
}
