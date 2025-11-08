package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	opts   RunOptions
	cursor int
}

func newModel(opts RunOptions) model {
	return model{opts: opts}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.opts.Plan.Steps)-1 {
				m.cursor++
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "MODE %s • PROFILE %s • TRANSPORT %s", strings.ToUpper(m.opts.Mode), strings.ToUpper(m.opts.Profile), strings.ToUpper(m.opts.Transport))
	if m.opts.Mirror {
		b.WriteString(" • MIRROR")
	}
	if m.opts.DryRun {
		b.WriteString(" • DRY-RUN")
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", 72))
	b.WriteString("\n")

	if len(m.opts.Plan.Steps) == 0 {
		b.WriteString("no planned steps\n")
		return tea.NewView(b.String())
	}

	b.WriteString("Steps:\n")
	for i, step := range m.opts.Plan.Steps {
		cursor := " "
		if i == m.cursor {
			cursor = "›"
		}
		sources := strings.Join(step.Sources, ",")
		fmt.Fprintf(&b, "%s %02d %-12s %s -> %s\n", cursor, i+1, step.Kind, shorthand(sources), step.Dest)
		if step.Reason != "" {
			fmt.Fprintf(&b, "    reason: %s\n", step.Reason)
		}
	}

	b.WriteString("\nKeys: ↑/↓ select • q quit\n")
	return tea.NewView(b.String())
}

func shorthand(path string) string {
	const max = 32
	if len(path) <= max {
		return path
	}
	if max <= 3 {
		return path[:max]
	}
	head := max/2 - 1
	tail := max - head - 3
	return path[:head] + "..." + path[len(path)-tail:]
}

var _ tea.Model = (*model)(nil)
