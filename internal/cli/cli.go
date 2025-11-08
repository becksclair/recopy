package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
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

	fmt.Fprintf(os.Stdout, "recopy stub — sources: %s\n", strings.Join(opts.Sources, ", "))
	fmt.Fprintf(os.Stdout, "dest: %s, profile: %s, move=%t, dry-run=%t, transport=%s\n",
		opts.Dest, opts.Profile, opts.Move, opts.DryRun, opts.Transport)
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
	opts.Dest = remaining[len(remaining)-1]
	opts.Sources = remaining[:len(remaining)-1]

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
