package rsync

import "testing"

const sampleVersion = `rsync  version 3.2.7  protocol version 31
Copyright (C) 1996-2022 by Andrew Tridgell, Wayne Davison, and others.
` + "more" + `
`

func TestParseCapabilities(t *testing.T) {
	caps, err := parseCapabilities([]byte(sampleVersion))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !caps.Version.AtLeast(3, 2, 7) {
		t.Fatalf("unexpected version: %+v", caps.Version)
	}
	if !caps.SupportsZstd || !caps.SupportsMkpath || !caps.SupportsChecksumChoice {
		t.Fatalf("expected advanced capabilities, got %+v", caps)
	}
}

func TestIntersectCaps(t *testing.T) {
	local := Capabilities{Version: Version{3, 2, 7}, SupportsZstd: true, SupportsMkpath: true, SupportsChecksumChoice: true, SupportsPreallocate: true}
	remote := Capabilities{Version: Version{3, 1, 3}, SupportsZstd: false, SupportsMkpath: true, SupportsChecksumChoice: false, SupportsPreallocate: true}
	merged := IntersectCaps(local, remote)
	if merged.Version != remote.Version {
		t.Fatalf("expected min version, got %+v", merged.Version)
	}
	if merged.SupportsZstd || !merged.SupportsMkpath || merged.SupportsChecksumChoice || !merged.SupportsPreallocate {
		t.Fatalf("unexpected capabilities: %+v", merged)
	}
}
