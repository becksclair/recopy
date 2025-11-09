package rsync

import "fmt"

// ArgsOptions describes high-level options for rsync argument synthesis.
type ArgsOptions struct {
	Source       string
	Dest         string
	Mirror       bool
	RemoveSource bool
	Inplace      bool
	Profile      string // auto|lan|wan
	PreferSparse bool
	DryRun       bool
}

// BuildArgs returns the argv slice for invoking rsync.
func BuildArgs(opts ArgsOptions, caps Capabilities) ([]string, error) {
	if opts.Source == "" || opts.Dest == "" {
		return nil, fmt.Errorf("rsync: source and dest must be set")
	}
	args := []string{
		"rsync",
		"-aHAX",
		"--info=progress2",
		"--itemize-changes",
		"--out-format=%i|%l|%n%L",
		"--outbuf=L",
		"--human-readable",
		"--partial",
		"--partial-dir=.rsync-partial",
	}
	if caps.SupportsPreallocate {
		args = append(args, "--preallocate")
	}
	if opts.Mirror {
		args = append(args, "--delete-delay")
	}
	if opts.Inplace {
		args = append(args, "--inplace")
	}
	if opts.RemoveSource {
		args = append(args, "--remove-source-files")
	}
	if opts.PreferSparse {
		args = append(args, "--sparse")
	}
	if opts.DryRun {
		args = append(args, "-n")
	}
	compression := compressionMode(opts.Profile)
	switch compression {
	case "zstd":
		args = append(args, "--compress")
		if caps.SupportsZstd {
			args = append(args, "--compress-choice=zstd", "--zl=1")
		}
	}
	if caps.SupportsMkpath {
		args = append(args, "--mkpath")
	}
	args = append(args, opts.Source, opts.Dest)
	return args, nil
}

func compressionMode(profile string) string {
	switch profile {
	case "wan":
		return "zstd"
	default:
		return "off"
	}
}
