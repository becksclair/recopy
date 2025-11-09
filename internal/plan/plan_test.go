package plan

import (
	"os"
	"path/filepath"
	"testing"

	"recopy/internal/fsprobe"
)

func TestBuildRenamePreferred(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	destDir := filepath.Join(dir, "dest")

	mustWriteFile(t, src)
	if err := mkdir(destDir); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	same, err := fsprobe.SameDevice(src, destDir)
	if err != nil {
		t.Fatalf("same device probe failed: %v", err)
	}
	if !same {
		t.Skip("filesystem does not support same-device fast path in test env")
	}

	result, err := Build(Input{
		Sources: []string{src},
		Dest:    filepath.Join(destDir, "out"),
		Options: Options{Move: true, Transport: "auto"},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(result.Steps) == 0 || result.Steps[0].Kind != StepRename {
		t.Fatalf("expected rename step, got %#v", result.Steps)
	}
}

func TestBuildDeterministicOrdering(t *testing.T) {
	dir := t.TempDir()
	if err := mkdir(filepath.Join(dir, "dest")); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	srcB := filepath.Join(dir, "b")
	srcA := filepath.Join(dir, "a")
	mustWriteFile(t, srcB)
	mustWriteFile(t, srcA)

	res, err := Build(Input{
		Sources: []string{srcB, srcA},
		Dest:    filepath.Join(dir, "dest", "out"),
		Options: Options{NoReflink: true, Transport: "rsync"},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(res.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(res.Steps))
	}
	if res.Steps[0].Sources[0] != srcA {
		t.Fatalf("expected srcA first, got %s", res.Steps[0].Sources[0])
	}
	if res.Steps[0].Kind != StepRsync || res.Steps[1].Kind != StepRsync {
		t.Fatalf("expected rsync steps, got %#v", res.Steps)
	}
}

func TestBuildRemoteDestinationForcesRsync(t *testing.T) {
	res, err := Build(Input{
		Sources: []string{"/tmp/src"},
		Dest:    "user@host:/data",
		Options: Options{Move: true, Transport: "auto"},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(res.Steps) != 1 || res.Steps[0].Kind != StepRsync {
		t.Fatalf("expected rsync step for remote dest, got %#v", res.Steps)
	}
}

func TestBuildRemoteSourcesRequireRsync(t *testing.T) {
	res, err := Build(Input{
		Sources: []string{"host:/src"},
		Dest:    "/tmp/dest",
		Options: Options{Transport: "auto"},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(res.Steps) != 1 || res.Steps[0].Kind != StepRsync {
		t.Fatalf("expected rsync step, got %#v", res.Steps)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := mkdir(filepath.Dir(path)); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := osWriteFile(path, []byte("recopy"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0o755)
}

var osWriteFile = os.WriteFile
