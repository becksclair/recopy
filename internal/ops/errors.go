package ops

import (
	"errors"
	"fmt"

	"recopy/internal/plan"
)

// StepError wraps failures for a specific plan step and carries source/dest context.
type StepError struct {
	Kind   plan.Kind
	Source string
	Dest   string
	Err    error
}

func (e StepError) Error() string {
	return fmt.Sprintf("%s step %s -> %s failed: %v", e.Kind, e.Source, e.Dest, e.Err)
}

func (e StepError) Unwrap() error {
	return e.Err
}

// ExitCode attempts to surface a process exit code when available.
func (e StepError) ExitCode() int {
	var coder interface{ ExitCode() int }
	if errors.As(e.Err, &coder) {
		return coder.ExitCode()
	}
	return 1
}

// CommandError captures failures from external binaries (rsync/btrfs/etc).
type CommandError struct {
	Name string
	Args []string
	Code int
	Err  error
}

func (e CommandError) Error() string {
	return fmt.Sprintf("%s exited with code %d", e.Name, e.ExitCode())
}

func (e CommandError) Unwrap() error {
	return e.Err
}

func (e CommandError) ExitCode() int {
	if e.Code <= 0 {
		return 1
	}
	return e.Code
}

func wrapStepError(kind plan.Kind, src, dest string, err error) error {
	if err == nil {
		return nil
	}
	return StepError{Kind: kind, Source: src, Dest: dest, Err: err}
}
