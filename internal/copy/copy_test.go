package copy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileCopy(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	// Write test data
	testData := []byte("hello recopy performance test")
	if err := os.WriteFile(srcPath, testData, 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	// Get source info
	info, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("failed to stat source: %v", err)
	}

	// Copy file
	opts := Options{PreserveAll: true, Sparse: false}
	if err := File(srcPath, dstPath, info, opts); err != nil {
		t.Fatalf("File() failed: %v", err)
	}

	// Verify destination exists and has same content
	gotData, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}

	if string(gotData) != string(testData) {
		t.Errorf("content mismatch: got %q, want %q", gotData, testData)
	}

	// Verify metadata preserved (at least mode)
	destInfo, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("failed to stat dest: %v", err)
	}

	if destInfo.Mode().Perm() != info.Mode().Perm() {
		t.Errorf("permissions mismatch: got %o, want %o", destInfo.Mode().Perm(), info.Mode().Perm())
	}
}
