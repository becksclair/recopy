//go:build !linux

package fsprobe

// HasSparseData is unsupported on non-Linux platforms.
func HasSparseData(path string) (bool, error) {
	return false, ErrSparseUnsupported
}
