package ops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"recopy/internal/fsprobe"
	"recopy/internal/plan"
	"recopy/internal/rsync"
)

// Executor runs plan steps using reflink and rsync helpers.
type Executor struct {
	caps rsync.Capabilities
}

// NewExecutor constructs an Executor with rsync capabilities.
func NewExecutor(caps rsync.Capabilities) Executor {
	return Executor{caps: caps}
}

// Run executes the plan in order.
func (e Executor) Run(ctx context.Context, pl plan.Plan, opts Options) error {
	resolver, err := newPathResolver(opts.Sources, opts.Dest)
	if err != nil {
		return err
	}

	for _, step := range pl.Steps {
		if len(step.Sources) != 1 {
			return fmt.Errorf("step %s expects single source", step.Kind)
		}
		src := step.Sources[0]
		target, err := resolver.TargetFor(src)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.stdout(), "[%s] %s -> %s\n", step.Kind, src, target)
		switch step.Kind {
		case plan.StepReflink:
			if err := e.handleReflink(ctx, src, target, opts); err != nil {
				if errors.Is(err, errReflinkUnsupported) {
					fmt.Fprintf(opts.stdout(), "reflink unsupported for %s, falling back to rsync\n", src)
					if err := e.handleRsync(ctx, src, target, opts); err != nil {
						return err
					}
					continue
				}
				return fmt.Errorf("reflink %s -> %s: %w", src, target, err)
			}
		case plan.StepRsync:
			if err := e.handleRsync(ctx, src, target, opts); err != nil {
				return err
			}
		case plan.StepRename:
			return fmt.Errorf("rename steps not implemented yet")
		case plan.StepBtrfsOffer:
			fmt.Fprintf(opts.stdout(), "skipping btrfs-offer step for now: %s\n", src)
		default:
			return fmt.Errorf("unsupported step kind %s", step.Kind)
		}
	}
	return nil
}

func (e Executor) handleReflink(ctx context.Context, src, dest string, opts Options) error {
	info, err := statPath(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errReflinkUnsupported
	}
	return reflinkCopy(src, dest, info)
}

func (e Executor) handleRsync(ctx context.Context, src, dest string, opts Options) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := ensureParentDir(dest); err != nil {
		return err
	}
	preferSparse := false
	sourceArg := src
	if info, err := statPath(src); err == nil {
		if info.Mode().IsRegular() {
			if ok, err := fsprobe.HasSparseData(src); err == nil {
				preferSparse = ok
			}
		} else if info.IsDir() {
			sourceArg = filepath.Clean(src) + string(os.PathSeparator)
		}
	}
	args, err := rsync.BuildArgs(rsync.ArgsOptions{
		Source:       sourceArg,
		Dest:         dest,
		Mirror:       opts.Mirror,
		RemoveSource: opts.Move,
		Inplace:      opts.Inplace,
		Profile:      opts.Profile,
		PreferSparse: preferSparse,
	}, e.caps)
	if err != nil {
		return err
	}
	return runCommand(ctx, args, opts.stdout(), opts.stderr())
}

func statPath(path string) (fileInfo, error) {
	return getFileInfo(path)
}

func ensureParentDir(path string) error {
	return mkdirAll(parentDir(path))
}

var errReflinkUnsupported = errors.New("reflink unsupported")
