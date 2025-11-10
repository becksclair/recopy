package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"recopy/internal/remotepath"
)

func TestParseSetsExpectedOptions(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dest := filepath.Join(tmp, "dest")
	args := []string{
		"--move",
		"--dry-run",
		"--mirror",
		"--profile", string(ProfileWAN),
		"--no-reflink",
		"--inplace",
		"--parallel", "3",
		"--transport", "rsync",
		"--prescan",
		"--verify",
		"--one-file-system",
		"--no-ui",
		src,
		dest,
	}
	opts, err := Parse(args)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !opts.Move || !opts.DryRun || !opts.Mirror {
		t.Fatalf("boolean flags not propagated: %+v", opts)
	}
	if opts.Profile != ProfileWAN {
		t.Fatalf("expected wan profile, got %s", opts.Profile)
	}
	if opts.Parallel != 3 {
		t.Fatalf("expected parallel 3, got %d", opts.Parallel)
	}
	if opts.Transport != "rsync" {
		t.Fatalf("unexpected transport %q", opts.Transport)
	}
	if !opts.NoReflink || !opts.Inplace || !opts.Prescan || !opts.Verify || !opts.OneFileSystem || !opts.NoUI {
		t.Fatalf("flags missing in options: %+v", opts)
	}
	if opts.Remote.Side != remotepath.SideNone {
		t.Fatalf("expected local layout, got %+v", opts.Remote)
	}
}

func TestParseRemoteDestLayout(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dest := "bex@host:/var/lib/data"
	opts, err := Parse([]string{src, dest})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if opts.Remote.Side != remotepath.SideDest {
		t.Fatalf("expected remote dest side, got %+v", opts.Remote)
	}
	if opts.Remote.Spec.Host != "host" || opts.Remote.Spec.User != "bex" {
		t.Fatalf("remote spec not captured: %+v", opts.Remote.Spec)
	}
}

func TestParseRejectsInvalidProfile(t *testing.T) {
	_, err := Parse([]string{"--profile", "fast", "src", "dest"})
	if err == nil || !strings.Contains(err.Error(), "invalid profile") {
		t.Fatalf("expected invalid profile error, got %v", err)
	}
}

func TestParseRejectsMixedRemoteSources(t *testing.T) {
	_, err := Parse([]string{"user@host:/src", "local", "dest"})
	if err == nil || !strings.Contains(err.Error(), "cannot mix remote and local sources") {
		t.Fatalf("expected mixed sources error, got %v", err)
	}
}
