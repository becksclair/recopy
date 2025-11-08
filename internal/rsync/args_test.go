package rsync

import "testing"

func TestBuildArgsCompression(t *testing.T) {
	caps := Capabilities{SupportsZstd: true, SupportsPreallocate: true, SupportsMkpath: true}
	args, err := BuildArgs(ArgsOptions{Source: "/tmp/src", Dest: "/tmp/dest", Profile: "wan"}, caps)
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}
	wantFlags := []string{"--compress", "--compress-choice=zstd", "--zl=1"}
	for _, flag := range wantFlags {
		if !contains(args, flag) {
			t.Fatalf("missing flag %s in %v", flag, args)
		}
	}
}

func TestBuildArgsMirror(t *testing.T) {
	caps := Capabilities{}
	args, err := BuildArgs(ArgsOptions{Source: "a", Dest: "b", Mirror: true}, caps)
	if err != nil {
		t.Fatalf("build args failed: %v", err)
	}
	if !contains(args, "--delete") {
		t.Fatalf("expected --delete flag, got %v", args)
	}
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
