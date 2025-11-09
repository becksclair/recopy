package ops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"recopy/internal/fsprobe"
	"recopy/internal/mv"
	"recopy/internal/plan"
	"recopy/internal/remotepath"
	"recopy/internal/rsync"
)

// Executor runs plan steps using reflink and rsync helpers.
type Executor struct {
	caps      rsync.Capabilities
	completed map[string]bool
}

// NewExecutor constructs an Executor with rsync capabilities.
func NewExecutor(caps rsync.Capabilities) Executor {
	return Executor{caps: caps, completed: make(map[string]bool)}
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
		key := stepKey(src, target)

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
			if e.completed[key] {
				fmt.Fprintf(opts.stdout(), "rsync skipped, already handled via btrfs for %s\n", src)
				continue
			}
			if err := e.handleRsync(ctx, src, target, opts); err != nil {
				return err
			}
		case plan.StepRename:
			if err := e.handleRename(src, target, opts); err != nil {
				return err
			}
		case plan.StepBtrfsOffer:
			if err := e.handleBtrfsOffer(ctx, src, target, opts); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported step kind %s", step.Kind)
		}
	}
	return nil
}

func (e Executor) handleReflink(ctx context.Context, src, dest string, opts Options) error {
	if opts.DryRun {
		fmt.Fprintf(opts.stdout(), "dry-run: would reflink %s -> %s\n", src, dest)
		return nil
	}
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
	srcRemote := remotepath.IsRemote(src)
	destRemote := remotepath.IsRemote(dest)
	if !destRemote {
		if err := ensureParentDir(dest); err != nil {
			return err
		}
	}
	preferSparse := false
	sourceArg := src
	if !srcRemote {
		if info, err := statPath(src); err == nil {
			if info.Mode().IsRegular() {
				if ok, err := fsprobe.HasSparseData(src); err == nil {
					preferSparse = ok
				}
			} else if info.IsDir() {
				sourceArg = filepath.Clean(src) + string(os.PathSeparator)
			}
		}
	}
	removeSource := opts.Move && !opts.DryRun
	args, err := rsync.BuildArgs(rsync.ArgsOptions{
		Source:       sourceArg,
		Dest:         dest,
		Mirror:       opts.Mirror,
		RemoveSource: removeSource,
		Inplace:      opts.Inplace,
		Profile:      opts.Profile,
		PreferSparse: preferSparse,
		DryRun:       opts.DryRun,
	}, e.caps)
	if err != nil {
		return err
	}
	if err := runCommand(ctx, args, opts.stdout(), opts.stderr()); err != nil {
		return err
	}
	if opts.Move && !srcRemote && !opts.DryRun {
		if err := mv.PruneEmptyDirs([]string{src}); err != nil {
			return fmt.Errorf("prune %s: %w", src, err)
		}
	}
	return nil
}

func (e Executor) handleRename(src, dest string, opts Options) error {
	if opts.DryRun {
		fmt.Fprintf(opts.stdout(), "dry-run: would rename %s -> %s\n", src, dest)
		return nil
	}
	if err := mv.Rename(src, dest); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", src, dest, err)
	}
	return nil
}

func statPath(path string) (fileInfo, error) {
	return getFileInfo(path)
}

func ensureParentDir(path string) error {
	return mkdirAll(parentDir(path))
}

var errReflinkUnsupported = errors.New("reflink unsupported")

func stepKey(src, dest string) string {
	return src + "->" + dest
}
