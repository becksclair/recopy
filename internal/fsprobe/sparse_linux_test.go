//go:build linux

package fsprobe

import (
	"os"
	"testing"
)

func TestHasSparseDataDetectsHoles(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sparse"
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()

	if err := f.Truncate(1 << 20); err != nil {
		t.Skipf("truncate unsupported: %v", err)
	}
	if _, err := f.WriteAt([]byte("recopy"), 0); err != nil {
		t.Fatalf("write start: %v", err)
	}
	if _, err := f.WriteAt([]byte("tail"), 1<<20-4); err != nil {
		t.Fatalf("write tail: %v", err)
	}

	sparse, err := HasSparseData(path)
	if err != nil {
		if err == ErrSparseUnsupported {
			t.Skip("fiemap unsupported")
		}
		t.Fatalf("has sparse: %v", err)
	}
	if !sparse {
		t.Fatalf("expected sparse file")
	}
}

func TestHasSparseDataDenseFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/dense"
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write dense: %v", err)
	}

	sparse, err := HasSparseData(path)
	if err != nil {
		if err == ErrSparseUnsupported {
			t.Skip("fiemap unsupported")
		}
		t.Fatalf("has sparse: %v", err)
	}
	if sparse {
		t.Fatalf("expected dense file")
	}
}
