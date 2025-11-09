package ops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathResolverMultiSourceRequiresDir(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dest")
	if _, err := newPathResolver([]string{"a", "b"}, dest); err == nil {
		t.Fatalf("expected error when dest missing for multi-source copy")
	}

	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	resolver, err := newPathResolver([]string{"a", "b"}, dest)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	target, err := resolver.TargetFor("foo.txt")
	if err != nil {
		t.Fatalf("target err: %v", err)
	}
	expect := filepath.Join(dest, "foo.txt")
	if target != expect {
		t.Fatalf("want %s, got %s", expect, target)
	}
}

func TestPathResolverSingleSourceFile(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.bin")
	resolver, err := newPathResolver([]string{"src.bin"}, dest)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	target, err := resolver.TargetFor("src.bin")
	if err != nil {
		t.Fatalf("target err: %v", err)
	}
	if target != dest {
		t.Fatalf("want %s, got %s", dest, target)
	}
}

func TestPathResolverRemoteDestMultiSource(t *testing.T) {
	resolver, err := newPathResolver([]string{"local1", "local2"}, "user@host:/uploads/")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	target, err := resolver.TargetFor("local1")
	if err != nil {
		t.Fatalf("target err: %v", err)
	}
	if target != "user@host:/uploads/local1" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestPathResolverRemoteDestSingleFile(t *testing.T) {
	resolver, err := newPathResolver([]string{"local"}, "host:/tmp/out.bin")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	target, err := resolver.TargetFor("local")
	if err != nil {
		t.Fatalf("target err: %v", err)
	}
	if target != "host:/tmp/out.bin" {
		t.Fatalf("unexpected target: %s", target)
	}
}
