package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"recopy/internal/plan"
	"recopy/internal/rsync"
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

	fmt.Fprintf(os.Stdout, "recopy plan (%d steps):\n", len(executionPlan.Steps))
	for i, step := range executionPlan.Steps {
		fmt.Fprintf(os.Stdout, "%02d. %-12s %s -> %s [%s]\n", i+1, step.Kind, strings.Join(step.Sources, ","), step.Dest, step.Reason)
		if step.Kind == plan.StepRsync {
			args, err := rsync.BuildArgs(rsync.ArgsOptions{
				Source:       step.Sources[0],
				Dest:         step.Dest,
				Mirror:       opts.Mirror,
				RemoveSource: opts.Move,
				Inplace:      opts.Inplace,
				Profile:      string(opts.Profile),
			}, caps)
			if err == nil {
				fmt.Fprintf(os.Stdout, "    rsync: %s\n", strings.Join(args, " "))
			}
		}
	}
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
