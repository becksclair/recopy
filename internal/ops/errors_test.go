package ops

import (
	"testing"

	"recopy/internal/plan"
)

func TestStepErrorExitCode(t *testing.T) {
	stepErr := StepError{
		Kind:   plan.StepRsync,
		Source: "src",
		Dest:   "dest",
		Err:    CommandError{Name: "rsync", Code: 23},
	}
	if got := stepErr.ExitCode(); got != 23 {
		t.Fatalf("expected exit code 23, got %d", got)
	}
}
