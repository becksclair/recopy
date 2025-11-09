package rsync

import "testing"

func TestBuildArgsDryRunMirror(t *testing.T) {
	args, err := BuildArgs(ArgsOptions{
		Source: "src",
		Dest:   "dest",
		Mirror: true,
		DryRun: true,
	}, Capabilities{SupportsMkpath: true})
	if err != nil {
		t.Fatalf("BuildArgs failed: %v", err)
	}
	if !contains(args, "-n") {
		t.Fatalf("expected -n for dry-run: %v", args)
	}
	if !contains(args, "--delete-delay") {
		t.Fatalf("expected --delete-delay for mirror: %v", args)
	}
	if contains(args, "--delete") {
		t.Fatalf("did not expect --delete when using delete-delay: %v", args)
	}
}

func contains(list []string, needle string) bool {
	for _, item := range list {
		if item == needle {
			return true
		}
	}
	return false
}
