package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"recopy/internal/rsync"
)

type doctorCheck struct {
	Name   string
	Advice string
	Run    func(context.Context) (string, error)
}

func runDoctor(ctx context.Context) int {
	checks := []doctorCheck{
		{
			Name:   "rsync",
			Advice: "install rsync 3.2.7+ so recopy can orchestrate transfers",
			Run: func(ctx context.Context) (string, error) {
				caps, err := rsync.DetectLocal(ctx)
				if err != nil {
					return "", err
				}
				ver := caps.Version
				extras := []string{}
				if caps.SupportsZstd {
					extras = append(extras, "zstd")
				}
				if caps.SupportsMkpath {
					extras = append(extras, "mkpath")
				}
				if caps.SupportsChecksumChoice {
					extras = append(extras, "checksum-choice")
				}
				if caps.SupportsPreallocate {
					extras = append(extras, "preallocate")
				}
				return fmt.Sprintf("version %d.%d.%d (%s)", ver.Major, ver.Minor, ver.Patch, strings.Join(extras, ",")), nil
			},
		},
		{
			Name:   "ssh",
			Advice: "install openssh-client 9.6+ for remote copies",
			Run: func(ctx context.Context) (string, error) {
				cmd := exec.CommandContext(ctx, "ssh", "-V")
				out, err := cmd.CombinedOutput()
				if err != nil {
					return "", err
				}
				return strings.TrimSpace(string(out)), nil
			},
		},
		{
			Name:   "btrfs",
			Advice: "install btrfs-progs if you plan to use btrfs send/receive",
			Run: func(context.Context) (string, error) {
				if _, err := exec.LookPath("btrfs"); err != nil {
					return "", err
				}
				return "btrfs-progs present", nil
			},
		},
		{
			Name:   "hyperfine",
			Advice: "install hyperfine to run `mise run bench`",
			Run: func(context.Context) (string, error) {
				if _, err := exec.LookPath("hyperfine"); err != nil {
					return "", err
				}
				return "hyperfine available", nil
			},
		},
		{
			Name:   "perf",
			Advice: "install linux-tools-common for profiling (see docs/perf.md)",
			Run: func(context.Context) (string, error) {
				if _, err := exec.LookPath("perf"); err != nil {
					return "", err
				}
				return "perf available", nil
			},
		},
		{
			Name:   "kernel: copy_file_range",
			Advice: "upgrade to kernel ≥5.10 for zero-copy support",
			Run: func(context.Context) (string, error) {
				if supported, err := probeCopyFileRange(); err != nil {
					return "", err
				} else if !supported {
					return "", fmt.Errorf("not supported")
				}
				return "supported", nil
			},
		},
		{
			Name:   "kernel: io_uring",
			Advice: "upgrade to kernel ≥5.10 for advanced async I/O (optional)",
			Run: func(context.Context) (string, error) {
				if supported, err := probeIOUring(); err != nil {
					return "", err
				} else if !supported {
					return "", fmt.Errorf("not supported")
				}
				return "supported", nil
			},
		},
	}

	fmt.Println("recopy doctor:")
	var failed bool
	for _, check := range checks {
		msg, err := check.Run(ctx)
		if err != nil {
			failed = true
			fmt.Printf("[warn] %s: %v\n", check.Name, err)
			if check.Advice != "" {
				fmt.Printf("       hint: %s\n", check.Advice)
			}
			continue
		}
		if msg != "" {
			fmt.Printf("[ ok ] %s: %s\n", check.Name, msg)
		} else {
			fmt.Printf("[ ok ] %s\n", check.Name)
		}
	}
	if failed {
		return 1
	}
	return 0
}
