//go:build !linux

package ops

import "os"

func reflinkCopy(src, dest string, info os.FileInfo) error {
	return errReflinkUnsupported
}
