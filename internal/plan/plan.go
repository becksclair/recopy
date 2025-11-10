package plan

import (
	"fmt"
	"path/filepath"
	"sort"

	"recopy/internal/fsprobe"
	"recopy/internal/remotepath"
)

// Kind represents a planned action.
type Kind string

const (
	// StepRename moves files via rename when both paths share a device.
	StepRename Kind = "rename"
	// StepReflink clones blocks via reflink when supported by the filesystem.
	StepReflink Kind = "reflink"
	// StepCopy performs local copy using copy_file_range and other kernel zero-copy mechanisms.
	StepCopy Kind = "copy"
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

// Build produces a deterministic Plan of Steps for the given Input.
// The planner emits one step per source (sorted deterministically) and selects the action based on source/destination characteristics:
// it may propose a Btrfs send/receive offer for Btrfs subvolumes (when transport is "auto"), use a rename for moves on the same device,
// use a reflink when both sides support it and reflinks are allowed, use a local copy engine (copy_file_range) for same-device copies
// when reflink is not used and rsync is not forced, and fall back to rsync otherwise.
// It returns an error if no sources are provided or if the destination path cannot be parsed.
func Build(in Input) (Plan, error) {
	if len(in.Sources) == 0 {
		return Plan{}, fmt.Errorf("plan: no sources provided")
	}
	_, destRemote, err := remotepath.Parse(in.Dest)
	if err != nil {
		return Plan{}, err
	}
	destParent := ""
	destReflink := false
	destBtrfs := fsprobe.Info{}
	if !destRemote {
		destParent = filepath.Dir(in.Dest)
		destReflink = fsprobe.SupportsReflink(destParent)
		destBtrfs, _ = fsprobe.BtrfsInfo(destParent)
	}

	sorted := append([]string(nil), in.Sources...)
	sort.Strings(sorted)

	var out Plan
	for _, src := range sorted {
		srcInfo := gatherSourceInfo(src, destParent, destRemote)
		if shouldOfferBtrfs(in, srcInfo, destBtrfs, destRemote) {
			out.Steps = append(out.Steps, Step{
				Kind:    StepBtrfsOffer,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "source + dest on btrfs subvols",
			})
		}

		// Check if user forced rsync transport explicitly
		forceRsync := in.Options.Transport == "rsync"

		switch {
		case !destRemote && !srcInfo.remote && in.Options.Move && srcInfo.sameDevice:
			out.Steps = append(out.Steps, Step{
				Kind:    StepRename,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "move on same device",
			})
		case !destRemote && !srcInfo.remote && srcInfo.sameDevice && !in.Options.NoReflink && srcInfo.reflinkOK && destReflink:
			out.Steps = append(out.Steps, Step{
				Kind:    StepReflink,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "reflink fast path",
			})
		case !destRemote && !srcInfo.remote && srcInfo.sameDevice && !forceRsync:
			// Local same-device copy without reflink support: use copy engine
			out.Steps = append(out.Steps, Step{
				Kind:    StepCopy,
				Sources: []string{src},
				Dest:    in.Dest,
				Reason:  "local copy engine (copy_file_range)",
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
	remote     bool
	sameDevice bool
	reflinkOK  bool
	btrfs      fsprobe.Info
}

func gatherSourceInfo(src, destParent string, destRemote bool) sourceInfo {
	if _, ok, err := remotepath.Parse(src); err == nil && ok {
		return sourceInfo{remote: true}
	}
	info := sourceInfo{}
	if destParent != "" && !destRemote {
		if same, err := fsprobe.SameDevice(src, destParent); err == nil {
			info.sameDevice = same
		}
	}
	info.reflinkOK = fsprobe.SupportsReflink(src)
	info.btrfs, _ = fsprobe.BtrfsInfo(src)
	return info
}

func shouldOfferBtrfs(in Input, src sourceInfo, dest fsprobe.Info, destRemote bool) bool {
	if in.Options.Transport != "auto" {
		return false
	}
	if src.remote || destRemote {
		return false
	}
	if !src.btrfs.IsBtrfs || !dest.IsBtrfs {
		return false
	}
	return src.btrfs.IsSubvolume
}