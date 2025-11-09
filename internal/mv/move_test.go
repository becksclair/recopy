package mv

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameAndPrune(t *testing.T) {
	tmp := t.TempDir()

	srcDir := filepath.Join(tmp, "src", "tree")
	if err := os.MkdirAll(filepath.Join(srcDir, "child"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	filePath := filepath.Join(srcDir, "child", "data.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	destDir := filepath.Join(tmp, "dest", "tree")
	if err := Rename(srcDir, destDir); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if _, err := os.Stat(destDir); err != nil {
		t.Fatalf("dest missing after rename: %v", err)
	}
	if _, err := os.Stat(srcDir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("source still present: %v", err)
	}

	// Recreate source tree remnants to exercise prune.
	emptyRoot := filepath.Join(tmp, "src", "leftover")
	if err := os.MkdirAll(filepath.Join(emptyRoot, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir empty root: %v", err)
	}
	keepRoot := filepath.Join(tmp, "src", "keep")
	if err := os.MkdirAll(filepath.Join(keepRoot, "child"), 0o755); err != nil {
		t.Fatalf("mkdir keep root: %v", err)
	}
	keepFile := filepath.Join(keepRoot, "child", "data")
	if err := os.WriteFile(keepFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}

	if err := PruneEmptyDirs([]string{emptyRoot, keepRoot}); err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if _, err := os.Stat(emptyRoot); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("empty root not pruned: %v", err)
	}
	if _, err := os.Stat(keepRoot); err != nil {
		t.Fatalf("keep root should remain: %v", err)
	}
}
