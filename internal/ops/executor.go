package ops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"recopy/internal/copy"
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
	parallel := opts.Parallel
	if parallel < 1 {
		parallel = 1
	}
	if parallel == 1 {
		return e.runSequential(ctx, pl, opts, resolver)
	}
	return e.runParallel(ctx, pl, opts, resolver, parallel)
}

func (e Executor) runSequential(ctx context.Context, pl plan.Plan, opts Options, resolver *pathResolver) error {
	dispatch := func(src, dest string) error {
		return e.handleRsync(ctx, src, dest, opts)
	}
	return e.runWithDispatcher(ctx, pl, opts, resolver, dispatch)
}

func (e Executor) runParallel(ctx context.Context, pl plan.Plan, opts Options, resolver *pathResolver, workers int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	tasks := make(chan rsyncTask)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				if ctx.Err() != nil {
					return
				}
				if err := e.handleRsync(ctx, task.src, task.dest, opts); err != nil {
					if errors.Is(err, context.Canceled) {
						cancel()
						return
					}
					select {
					case errCh <- err:
					default:
					}
					cancel()
					return
				}
			}
		}()
	}

	dispatch := func(src, dest string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tasks <- rsyncTask{src: src, dest: dest}:
			return nil
		}
	}

	runErr := e.runWithDispatcher(ctx, pl, opts, resolver, dispatch)
	close(tasks)
	wg.Wait()
	if runErr != nil {
		return runErr
	}
	select {
	case workerErr := <-errCh:
		if workerErr != nil {
			return workerErr
		}
	default:
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (e Executor) runWithDispatcher(ctx context.Context, pl plan.Plan, opts Options, resolver *pathResolver, dispatch func(src, dest string) error) error {
	for _, step := range pl.Steps {
		if err := ctx.Err(); err != nil {
			return err
		}
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
					if err := dispatch(src, target); err != nil {
						return err
					}
					continue
				}
				return err
			}
		case plan.StepCopy:
			if err := e.handleCopy(ctx, src, target, opts); err != nil {
				return err
			}
		case plan.StepRsync:
			if e.completed[key] {
				fmt.Fprintf(opts.stdout(), "rsync skipped, already handled via btrfs for %s\n", src)
				continue
			}
			if err := dispatch(src, target); err != nil {
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

type rsyncTask struct {
	src  string
	dest string
}

func (e Executor) handleReflink(ctx context.Context, src, dest string, opts Options) error {
	if opts.DryRun {
		fmt.Fprintf(opts.stdout(), "dry-run: would reflink %s -> %s\n", src, dest)
		return nil
	}
	info, err := statPath(src)
	if err != nil {
		return wrapStepError(plan.StepReflink, src, dest, err)
	}
	if info.IsDir() {
		return errReflinkUnsupported
	}
	if err := reflinkCopy(src, dest, info); err != nil {
		return wrapStepError(plan.StepReflink, src, dest, err)
	}
	return nil
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
		if errors.Is(err, context.Canceled) {
			return err
		}
		return wrapStepError(plan.StepRsync, src, dest, err)
	}
	if opts.Move && !srcRemote && !opts.DryRun {
		if err := mv.PruneEmptyDirs([]string{src}); err != nil {
			return wrapStepError(plan.StepRsync, src, dest, fmt.Errorf("prune %s: %w", src, err))
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
		return wrapStepError(plan.StepRename, src, dest, fmt.Errorf("rename %s -> %s: %w", src, dest, err))
	}
	return nil
}

func (e Executor) handleCopy(ctx context.Context, src, dest string, opts Options) error {
	if opts.DryRun {
		fmt.Fprintf(opts.stdout(), "dry-run: would copy %s -> %s\n", src, dest)
		return nil
	}

	info, err := statPath(src)
	if err != nil {
		return wrapStepError(plan.StepCopy, src, dest, err)
	}

	if info.IsDir() {
		// For directories, walk and copy all files
		return e.copyDirectory(ctx, src, dest, opts)
	}

	// Single file copy
	if err := ensureParentDir(dest); err != nil {
		return wrapStepError(plan.StepCopy, src, dest, err)
	}

	// Detect sparse files
	sparse := false
	if info.Mode().IsRegular() {
		if ok, err := fsprobe.HasSparseData(src); err == nil {
			sparse = ok
		}
	}

	copyOpts := copy.Options{
		PreserveAll: true, // Always preserve metadata like cp -a
		Sparse:      sparse,
	}

	if err := copy.File(src, dest, info, copyOpts); err != nil {
		return wrapStepError(plan.StepCopy, src, dest, err)
	}

	return nil
}

func (e Executor) copyDirectory(ctx context.Context, src, dest string, opts Options) error {
	// Walk the source directory and copy all files
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Compute relative path and destination
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, relPath)

		// Create directories
		if info.IsDir() {
			return mkdirAll(targetPath)
		}

		// Copy files
		copyOpts := copy.Options{
			PreserveAll: true,
			Sparse:      false,
		}

		if err := copy.File(path, targetPath, info, copyOpts); err != nil {
			return fmt.Errorf("copy %s: %w", path, err)
		}

		return nil
	})
}

// statPath returns file information for the given path.
// It returns a fileInfo describing the file or directory at path, or an error if the path cannot be accessed.
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