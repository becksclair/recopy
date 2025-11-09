package rsync

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"regexp"

	"recopy/internal/remotepath"
)

// Version holds parsed rsync version numbers.
type Version struct {
	Major int
	Minor int
	Patch int
}

// Capabilities models compiled-in rsync features we care about.
type Capabilities struct {
	Version                Version
	SupportsZstd           bool
	SupportsMkpath         bool
	SupportsChecksumChoice bool
	SupportsPreallocate    bool
}

var versionRegex = regexp.MustCompile(`rsync\s+version\s+(\d+)\.(\d+)\.(\d+)`)

// DetectLocal executes `rsync --version` and infers feature gates.
func DetectLocal(ctx context.Context) (Capabilities, error) {
	cmd := exec.CommandContext(ctx, "rsync", "--version")
	out, err := cmd.Output()
	if err != nil {
		return Capabilities{}, err
	}
	return parseCapabilities(out)
}

// DetectRemote probes rsync capabilities on a remote host via ssh.
func DetectRemote(ctx context.Context, spec remotepath.Spec) (Capabilities, error) {
	addr := spec.Address()
	cmd := exec.CommandContext(ctx, "ssh", addr, "LC_ALL=C", "rsync", "--version")
	out, err := cmd.Output()
	if err != nil {
		return Capabilities{}, err
	}
	return parseCapabilities(out)
}

// parseCapabilities builds feature info from raw --version output.
func parseCapabilities(out []byte) (Capabilities, error) {
	ver, err := parseVersion(out)
	if err != nil {
		return Capabilities{}, err
	}
	caps := Capabilities{Version: ver}
	if ver.AtLeast(3, 2, 3) {
		caps.SupportsZstd = true
		caps.SupportsMkpath = true
	}
	if ver.AtLeast(3, 2, 4) {
		caps.SupportsChecksumChoice = true
	}
	if ver.AtLeast(3, 2, 0) {
		caps.SupportsPreallocate = true
	}
	return caps, nil
}

func parseVersion(out []byte) (Version, error) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		match := versionRegex.FindStringSubmatch(line)
		if len(match) == 4 {
			return Version{
				Major: atoi(match[1]),
				Minor: atoi(match[2]),
				Patch: atoi(match[3]),
			}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return Version{}, err
	}
	return Version{}, errors.New("rsync: unable to parse version output")
}

// AtLeast reports whether v >= major.minor.patch.
func (v Version) AtLeast(major, minor, patch int) bool {
	if v.Major != major {
		return v.Major > major
	}
	if v.Minor != minor {
		return v.Minor > minor
	}
	return v.Patch >= patch
}

// LessThan reports whether v < other.
func (v Version) LessThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

// IntersectCaps returns the shared feature set between two endpoints.
func IntersectCaps(a, b Capabilities) Capabilities {
	result := a
	if b.Version.LessThan(a.Version) {
		result.Version = b.Version
	}
	result.SupportsZstd = a.SupportsZstd && b.SupportsZstd
	result.SupportsMkpath = a.SupportsMkpath && b.SupportsMkpath
	result.SupportsChecksumChoice = a.SupportsChecksumChoice && b.SupportsChecksumChoice
	result.SupportsPreallocate = a.SupportsPreallocate && b.SupportsPreallocate
	return result
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}
