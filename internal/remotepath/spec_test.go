package remotepath

import "testing"

func TestParseRemote(t *testing.T) {
	cases := []struct {
		input  string
		remote bool
		user   string
		host   string
		path   string
	}{
		{"user@example.com:/data", true, "user", "example.com", "/data"},
		{"host:/var/tmp", true, "", "host", "/var/tmp"},
		{"[2001:db8::1]:/srv", true, "", "[2001:db8::1]", "/srv"},
		{"user@[2001:db8::1]:~/dir", true, "user", "[2001:db8::1]", "~/dir"},
		{"/local/path", false, "", "", ""},
		{"C:/windows", false, "", "", ""},
	}
	for _, tc := range cases {
		spec, ok, err := Parse(tc.input)
		if err != nil {
			t.Fatalf("parse %q failed: %v", tc.input, err)
		}
		if ok != tc.remote {
			t.Fatalf("parse %q remote=%v want %v", tc.input, ok, tc.remote)
		}
		if !ok {
			continue
		}
		if spec.User != tc.user || spec.Host != tc.host || spec.Path != tc.path {
			t.Fatalf("parse %q mismatch: %+v", tc.input, spec)
		}
	}
}

func TestParseRejectsEmptyPath(t *testing.T) {
	if _, _, err := Parse("user@host:"); err == nil {
		t.Fatalf("expected error for missing path")
	}
}

func TestJoinAndBase(t *testing.T) {
	spec := Spec{User: "a", Host: "b", Path: "/root"}
	joined := Join(spec, "file.txt")
	if joined.Path != "/root/file.txt" {
		t.Fatalf("unexpected join: %s", joined.Path)
	}
	if base := Base(joined); base != "file.txt" {
		t.Fatalf("unexpected base: %s", base)
	}
}
