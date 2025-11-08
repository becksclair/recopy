package plan

import (
	"fmt"
	"path/filepath"
	"sort"

	"recopy/internal/fsprobe"
)

// Kind represents a planned action.
type Kind string

const (
	// StepRename moves files via rename when both paths share a device.
	StepRename Kind = "rename"
	// StepReflink clones blocks via reflink when supported by the filesystem.
	StepReflink Kind = "reflink"
	// StepRsync copies data via rsync.
	StepRsync Kind = "rsync"
	// StepBtrfsOffer proposes a send/receive fast path for subvolumes.
	StepBtrfsOffer Kind = "btrfs-offer"
)

// Step is a single unit of work for the executor.
type Step struct {
	Kind    Kind
	Sources []string
	Dest    string
	Reason  string
}

// Options influences planning decisions derived from CLI flags.
type Options struct {
	Move      bool
	NoReflink bool
	Transport string
}

// Input is the payload for Build.
type Input struct {
	Sources []string
	Dest    string
	Options Options
}

// Plan is the ordered list of steps.
type Plan struct {
	Steps []Step
}

// Build produces a deterministic set of steps for the provided inputs.
func Build(in Input) (Plan, error) {
	if len(in.Sources) == 0 {
		return Plan{}, fmt.Errorf("plan: no sources provided")
	}
	destParent := filepath.Dir(in.Dest)

	destReflink := fsprobe.SupportsReflink(destParent)
	destBtrfs, _ := fsprobe.BtrfsInfo(destParent)

	sorted := append([]string(nil), in.Sources...)
	sort.Strings(sorted)

	var out Plan
	for _, src := range sorted {
		srcInfo := gatherSourceInfo(src, destParent)
		if shouldOfferBtrfs(in, srcInfo, destBtrfs) {
			out.Steps = append(out.Steps, Step{
				Kind:    StepBtrfsOffer,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "source + dest on btrfs subvols",
			})
		}

		switch {
		case in.Options.Move && srcInfo.sameDevice:
			out.Steps = append(out.Steps, Step{
				Kind:    StepRename,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "move on same device",
			})
		case srcInfo.sameDevice && !in.Options.NoReflink && srcInfo.reflinkOK && destReflink:
			out.Steps = append(out.Steps, Step{
				Kind:    StepReflink,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "reflink fast path",
			})
		default:
			out.Steps = append(out.Steps, Step{
				Kind:    StepRsync,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "rsync fallback",
			})
		}
	}

	return out, nil
}

type sourceInfo struct {
	sameDevice bool
	reflinkOK  bool
	btrfs      fsprobe.Info
}

func gatherSourceInfo(src, destParent string) sourceInfo {
	same, err := fsprobe.SameDevice(src, destParent)
	if err != nil {
		same = false
	}
	reflink := fsprobe.SupportsReflink(src)
	btrfs, _ := fsprobe.BtrfsInfo(src)
	return sourceInfo{sameDevice: same, reflinkOK: reflink, btrfs: btrfs}
}

func shouldOfferBtrfs(in Input, src sourceInfo, dest fsprobe.Info) bool {
	if in.Options.Transport != "auto" {
		return false
	}
	if !src.btrfs.IsBtrfs || !dest.IsBtrfs {
		return false
	}
	return src.btrfs.IsSubvolume
}
