package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathsCleansInputs(t *testing.T) {
	src := []string{"./foo/../foo//bar/"}
	dest := "./dest//sub"

	cleanedSrc, cleanedDest, err := normalizePaths(src, dest)
	if err != nil {
		t.Fatalf("normalize paths failed: %v", err)
	}
	expectedSrc := filepath.Join("foo", "bar")
	if cleanedSrc[0] != expectedSrc {
		t.Fatalf("expected %s, got %s", expectedSrc, cleanedSrc[0])
	}
	expectedDest := filepath.Join("dest", "sub")
	if cleanedDest != expectedDest {
		t.Fatalf("expected dest %s, got %s", expectedDest, cleanedDest)
	}
}

func TestNormalizePathsRejectsDestInsideSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(src, "nested")
	cleaned, _, err := normalizePaths([]string{src}, dest)
	if err == nil {
		t.Fatalf("expected error for dest inside source, got cleaned=%v", cleaned)
	}
	if !strings.Contains(err.Error(), "inside source") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizePathsRemote(t *testing.T) {
	src := []string{"user@host:/data"}
	dest := "/local"
	cleanedSrc, cleanedDest, err := normalizePaths(src, dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanedSrc[0] != "user@host:/data" {
		t.Fatalf("remote source mutated: %s", cleanedSrc[0])
	}
	if cleanedDest != dest {
		t.Fatalf("dest changed: %s", cleanedDest)
	}
}
