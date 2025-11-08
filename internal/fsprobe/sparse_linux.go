//go:build linux

package fsprobe

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	fiemapExtentCount              = 128
	fiemapFlagSync                 = 0x00000001
	fiemapExtentFlagUnknown        = 0x00000002
	fiemapExtentFlagDelalloc       = 0x00000004
	fiemapExtentFlagEncoded        = 0x00000008
	fiemapExtentFlagUnwritten      = 0x00000800
	fsIOCFiemap               uint = 0xC020660B
)

type fiemapExtent struct {
	Logical    uint64
	Physical   uint64
	Length     uint64
	Reserved64 [2]uint64
	Flags      uint32
	Reserved   [3]uint32
}

type fiemapRequest struct {
	Start         uint64
	Length        uint64
	Flags         uint32
	MappedExtents uint32
	ExtentCount   uint32
	Reserved      uint32
	Extents       [fiemapExtentCount]fiemapExtent
}

// HasSparseData reports whether the file at path contains holes via FIEMAP.
func HasSparseData(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	var req fiemapRequest
	req.Length = ^uint64(0)
	req.Flags = fiemapFlagSync
	req.ExtentCount = fiemapExtentCount

	if err := ioctlFiemap(f.Fd(), &req); err != nil {
		if isFiemapUnsupported(err) {
			return false, ErrSparseUnsupported
		}
		return false, fmt.Errorf("fiemap: %w", err)
	}

	sparse := analyzeSparse(&req, uint64(info.Size()))
	return sparse, nil
}

func ioctlFiemap(fd uintptr, req *fiemapRequest) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(fsIOCFiemap), uintptr(unsafe.Pointer(req)))
	if errno != 0 {
		return errno
	}
	return nil
}

func isFiemapUnsupported(err error) bool {
	return errors.Is(err, unix.ENOTTY) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EINVAL)
}

func analyzeSparse(req *fiemapRequest, fileSize uint64) bool {
	if fileSize == 0 {
		return false
	}
	if req.MappedExtents == 0 {
		return true
	}
	var logicalCursor uint64
	for i := uint32(0); i < req.MappedExtents && i < req.ExtentCount; i++ {
		ext := req.Extents[i]
		if ext.Flags&(fiemapExtentFlagUnknown|fiemapExtentFlagDelalloc|fiemapExtentFlagEncoded|fiemapExtentFlagUnwritten) != 0 {
			return true
		}
		if ext.Logical > logicalCursor {
			return true
		}
		logicalCursor = ext.Logical + ext.Length
	}
	return logicalCursor < fileSize
}
