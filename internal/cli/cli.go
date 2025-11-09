package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"recopy/internal/fsprobe"
	"recopy/internal/ops"
	"recopy/internal/plan"
	"recopy/internal/remotepath"
	"recopy/internal/rsync"
	"recopy/internal/tui"
)

// Profile represents the performance presets exposed on the CLI.
type Profile string

const (
	ProfileAuto Profile = "auto"
	ProfileLAN  Profile = "lan"
	ProfileWAN  Profile = "wan"
)

// Options captures the normalized CLI inputs.
type Options struct {
	Move          bool
	DryRun        bool
	Mirror        bool
	Profile       Profile
	NoReflink     bool
	Inplace       bool
	Parallel      int
	Transport     string
	Prescan       bool
	Verify        bool
	OneFileSystem bool
	NoUI          bool
	Sources       []string
	Dest          string
	Remote        remotepath.Layout
}

// Run parses CLI args, validates basic invariants, and prints a short summary.
func Run(args []string) int {
	opts, err := Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "recopy:", err)
		return 2
	}

	caps, err := rsync.DetectLocal(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "recopy: warning: rsync detection failed:", err)
	}
	if opts.Remote.Side != remotepath.SideNone {
		remoteCaps, rerr := rsync.DetectRemote(context.Background(), opts.Remote.Spec)
		if rerr != nil {
			fmt.Fprintln(os.Stderr, "recopy: warning: remote rsync detection failed:", rerr)
		} else {
			caps = rsync.IntersectCaps(caps, remoteCaps)
		}
	}

	planInput := plan.Input{
		Sources: opts.Sources,
		Dest:    opts.Dest,
		Options: plan.Options{
			Move:      opts.Move,
			NoReflink: opts.NoReflink,
			Transport: opts.Transport,
		},
	}
	executionPlan, err := plan.Build(planInput)
	if err != nil {
		fmt.Fprintln(os.Stderr, "recopy: plan build failed:", err)
		return 2
	}

	if opts.DryRun && shouldUseTUI(opts) {
		uiErr := tui.Run(context.Background(), tui.RunOptions{
			Plan:      executionPlan,
			Mode:      modeLabel(opts),
			Profile:   string(opts.Profile),
			Transport: opts.Transport,
			Mirror:    opts.Mirror,
			DryRun:    opts.DryRun,
		})
		if uiErr != nil {
			fmt.Fprintln(os.Stderr, "recopy: warning: tui failed, falling back to plain output:", uiErr)
			printPlanSummary(os.Stdout, executionPlan, opts, caps)
		}
		return 0
	}
	printPlanSummary(os.Stdout, executionPlan, opts, caps)
	if opts.DryRun {
		return 0
	}
	executor := ops.NewExecutor(caps)
	btrfsDecider := newBtrfsDecider(executionPlan)
	execOpts := ops.Options{
		Sources:      opts.Sources,
		Dest:         opts.Dest,
		Profile:      string(opts.Profile),
		Mirror:       opts.Mirror,
		Move:         opts.Move,
		Inplace:      opts.Inplace,
		DryRun:       opts.DryRun,
		BtrfsDecider: btrfsDecider,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	}
	if err := executor.Run(context.Background(), executionPlan, execOpts); err != nil {
		fmt.Fprintln(os.Stderr, "recopy: execution failed:", err)
		return 1
	}
	fmt.Fprintln(os.Stdout, "recopy: completed")
	return 0
}

// Parse converts CLI args into Options, enforcing cp-style SRC... DEST semantics.
func Parse(args []string) (Options, error) {
	opts := Options{Profile: ProfileAuto, Parallel: 1, Transport: "auto"}

	fs := flag.NewFlagSet("recopy", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), usageSynopsis())
		fs.PrintDefaults()
	}

	fs.BoolVar(&opts.Move, "move", false, "rename when possible, fallback to copy")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "plan only")
	fs.BoolVar(&opts.Mirror, "mirror", false, "delete extraneous destination data")
	profile := fs.String("profile", string(ProfileAuto), "auto|lan|wan tuning presets")
	fs.BoolVar(&opts.NoReflink, "no-reflink", false, "disable reflink fast path")
	fs.BoolVar(&opts.Inplace, "inplace", false, "rsync --inplace copy strategy")
	fs.IntVar(&opts.Parallel, "parallel", 1, "max concurrent workers")
	fs.StringVar(&opts.Transport, "transport", "auto", "auto|rsync|btrfs")
	fs.BoolVar(&opts.Prescan, "prescan", false, "enable metadata prescan")
	fs.BoolVar(&opts.Verify, "verify", false, "re-verify data after transfer")
	fs.BoolVar(&opts.OneFileSystem, "one-file-system", false, "stay on same filesystem")
	fs.BoolVar(&opts.NoUI, "no-ui", false, "disable Bubble Tea TUI")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	normalizedProfile := Profile(*profile)
	if err := validateProfile(normalizedProfile); err != nil {
		return Options{}, err
	}
	opts.Profile = normalizedProfile

	if opts.Parallel < 1 {
		return Options{}, errors.New("--parallel must be >= 1")
	}

	remaining := fs.Args()
	if len(remaining) < 2 {
		return Options{}, errors.New("expected SRC... DEST paths")
	}
	rawDest := remaining[len(remaining)-1]
	rawSources := remaining[:len(remaining)-1]
	cleanedSources, cleanedDest, err := normalizePaths(rawSources, rawDest)
	if err != nil {
		return Options{}, err
	}
	opts.Sources = cleanedSources
	opts.Dest = cleanedDest
	layout, err := remotepath.ClassifyPaths(opts.Sources, opts.Dest)
	if err != nil {
		return Options{}, err
	}
	opts.Remote = layout

	return opts, nil
}

func validateProfile(p Profile) error {
	switch p {
	case ProfileAuto, ProfileLAN, ProfileWAN:
		return nil
	default:
		return fmt.Errorf("invalid profile %q", p)
	}
}

func usageSynopsis() string {
	return "Usage: recopy [--move] [--dry-run] [--mirror]\n" +
		"               [--profile auto|lan|wan]\n" +
		"               [--no-reflink] [--inplace]\n" +
		"               [--parallel N]\n" +
		"               [--transport auto|rsync|btrfs]\n" +
		"               [--prescan]\n" +
		"               [--verify]\n" +
		"               [--one-file-system]\n" +
		"               [--no-ui]\n" +
		"               SRC... DEST\n\n"
}

func shouldUseTUI(opts Options) bool {
	if opts.NoUI {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func modeLabel(opts Options) string {
	if opts.Move {
		return "move"
	}
	return "copy"
}

func newBtrfsDecider(pl plan.Plan) ops.BtrfsDecider {
	if !hasBtrfsOffer(pl) {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "recopy: btrfs fast path available but stdin is not a TTY; skipping offer")
		return nil
	}
	reader := bufio.NewReader(os.Stdin)
	cache := make(map[string]bool)
	return func(src, dest string) (bool, error) {
		key := src + "->" + dest
		if val, ok := cache[key]; ok {
			return val, nil
		}
		fmt.Fprintf(os.Stdout, "Btrfs fast path available for %s -> %s. Use it? [y/N]: ", src, dest)
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		accept := answer == "y" || answer == "yes"
		cache[key] = accept
		return accept, nil
	}
}

func hasBtrfsOffer(pl plan.Plan) bool {
	for _, step := range pl.Steps {
		if step.Kind == plan.StepBtrfsOffer {
			return true
		}
	}
	return false
}

func printPlanSummary(w io.Writer, executionPlan plan.Plan, opts Options, caps rsync.Capabilities) {
	fmt.Fprintf(w, "recopy plan (%d steps):\n", len(executionPlan.Steps))
	for i, step := range executionPlan.Steps {
		fmt.Fprintf(w, "%02d. %-12s %s -> %s [%s]\n", i+1, step.Kind, strings.Join(step.Sources, ","), step.Dest, step.Reason)
		if step.Kind == plan.StepRsync {
			var sparse bool
			if len(step.Sources) == 1 {
				if info, err := os.Stat(step.Sources[0]); err == nil && info.Mode().IsRegular() {
					if ok, err := fsprobe.HasSparseData(step.Sources[0]); err == nil {
						sparse = ok
					} else if !errors.Is(err, fsprobe.ErrSparseUnsupported) {
						fmt.Fprintf(os.Stderr, "recopy: warning: sparse probe failed for %s: %v\n", step.Sources[0], err)
					}
				}
			}
			args, err := rsync.BuildArgs(rsync.ArgsOptions{
				Source:       step.Sources[0],
				Dest:         step.Dest,
				Mirror:       opts.Mirror,
				RemoveSource: opts.Move,
				Inplace:      opts.Inplace,
				Profile:      string(opts.Profile),
				PreferSparse: sparse,
			}, caps)
			if err == nil {
				fmt.Fprintf(w, "    rsync: %s\n", strings.Join(args, " "))
			}
		}
	}
}
