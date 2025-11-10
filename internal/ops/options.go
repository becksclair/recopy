package ops

import "io"

// Options configure executor behavior.
type Options struct {
	Sources []string
	Dest    string

	Profile      string
	Mirror       bool
	Move         bool
	Inplace      bool
	DryRun       bool
	Parallel     int
	BtrfsDecider BtrfsDecider

	Stdout io.Writer
	Stderr io.Writer
}

// BtrfsDecider decides whether to accept a btrfs offer for a source/dest pair.
type BtrfsDecider func(src, dest string) (bool, error)

func (o Options) stdout() io.Writer {
	if o.Stdout != nil {
		return o.Stdout
	}
	return io.Discard
}

func (o Options) stderr() io.Writer {
	if o.Stderr != nil {
		return o.Stderr
	}
	return io.Discard
}
