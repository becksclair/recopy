package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"recopy/internal/plan"
)

// RunOptions controls Bubble Tea shell behavior.
type RunOptions struct {
	Plan        plan.Plan
	Mode        string
	Profile     string
	Transport   string
	Mirror      bool
	DryRun      bool
	InitialLogs []string
}

// LogMsg appends a line to the drawer; future exec paths can emit this.
type LogMsg struct {
	Line string
}

// Run launches the Bubble Tea shell for the provided plan.
func Run(ctx context.Context, opts RunOptions) error {
	if len(opts.Plan.Steps) == 0 {
		return errors.New("tui: empty plan")
	}
	m := newModel(opts)
	program := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := program.Run()
	return err
}
