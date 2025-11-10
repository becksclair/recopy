package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIIntegrationLocalDryRun(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "sample"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	output := runRecopyCmd(t, "--dry-run", src, dest)
	if !strings.Contains(output, "recopy: dry-run completed") {
		t.Fatalf("expected dry-run completion, got output:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("-> %s", dest)) {
		t.Fatalf("expected dest mention, got:\n%s", output)
	}
}

func TestCLIIntegrationRemoteDestDryRun(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "data"), []byte("dat"), 0o644); err != nil {
		t.Fatalf("write data: %v", err)
	}
	dest := "user@remote:/tmp/dest"

	output := runRecopyCmd(t, "--dry-run", src, dest)
	if !strings.Contains(output, "recopy: dry-run completed") {
		t.Fatalf("expected dry-run completion, got output:\n%s", output)
	}
	if !strings.Contains(output, dest) {
		t.Fatalf("expected remote dest mention, got:\n%s", output)
	}
}

func runRecopyCmd(t *testing.T, args ...string) string {
	t.Helper()
	root := repoRoot(t)
	cmdArgs := append([]string{"run", "./cmd/recopy"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = root
	cmd.Env = stubbedEnv(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("recopy command failed: %v\noutput:\n%s", err, string(out))
	}
	return string(out)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found in repo tree")
		}
		dir = parent
	}
}

func stubbedEnv(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	binDir := filepath.Join(root, "testdata", "bin")
	pathEntry := fmt.Sprintf("PATH=%s%c%s", binDir, os.PathListSeparator, os.Getenv("PATH"))
	env := os.Environ()
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = pathEntry
			return env
		}
	}
	env = append(env, pathEntry)
	return env
}
