package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"recopy/internal/plan"
)

type model struct {
	opts        RunOptions
	cursor      int
	logs        []string
	showLog     bool
	showHelp    bool
	frozen      bool
	offers      []offerState
	showOffer   bool
	activeOffer int
}

func newModel(opts RunOptions) model {
	m := model{opts: opts, logs: append([]string(nil), opts.InitialLogs...)}
	m.offers = collectOffers(opts.Plan)
	if len(m.offers) > 0 {
		m.showOffer = true
		m.activeOffer = nextPendingOffer(m.offers)
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if m.showOffer {
			if handled := m.handleOfferKey(key); handled {
				return m, nil
			}
		}
		switch key {
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
		case "v":
			m.showLog = !m.showLog
		case "?":
			m.showHelp = !m.showHelp
		case "F":
			m.frozen = !m.frozen
		case "o":
			if len(m.offers) > 0 {
				m.showOffer = true
				m.activeOffer = nextPendingOffer(m.offers)
			}
		case "esc":
			m.showHelp = false
			m.showOffer = false
		}
	case LogMsg:
		if msg.Line != "" {
			m.logs = append(m.logs, msg.Line)
			if len(m.logs) > maxLogLines {
				m.logs = m.logs[len(m.logs)-maxLogLines:]
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "MODE %s • PROFILE %s • TRANSPORT %s", strings.ToUpper(m.opts.Mode), strings.ToUpper(m.opts.Profile), strings.ToUpper(m.opts.Transport))
	workers := m.opts.Workers
	if workers <= 0 {
		workers = 1
	}
	fmt.Fprintf(&b, " • WORKERS %d", workers)
	if m.opts.Mirror {
		b.WriteString(" • MIRROR")
	}
	if m.opts.DryRun {
		b.WriteString(" • DRY-RUN")
	}
	if m.frozen {
		b.WriteString(" • FROZEN")
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

	b.WriteString("\n")
	if m.showHelp {
		b.WriteString(renderHelp())
		b.WriteString("\n")
	}
	if m.showOffer {
		b.WriteString(renderOfferModal(m.currentOffer()))
	}
	b.WriteString("Keys: ↑/↓ select • v log • F freeze • o offers • ? help • q quit\n")
	if m.showLog {
		b.WriteString(renderLogs(m.logs))
	}
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

const maxLogLines = 200

func renderHelp() string {
	return "Help:\n" +
		"  q / ctrl+c  quit\n" +
		"  v           toggle log drawer\n" +
		"  F           freeze screen (placeholder)\n" +
		"  ?           toggle this help\n" +
		"  ↑/↓, j/k    navigate plan items\n"
}

func renderLogs(logs []string) string {
	var b strings.Builder
	b.WriteString("\nLog drawer:\n")
	if len(logs) == 0 {
		b.WriteString("  (no log lines yet)\n")
		return b.String()
	}
	start := 0
	const maxDisplay = 5
	if len(logs) > maxDisplay {
		start = len(logs) - maxDisplay
	}
	for _, line := range logs[start:] {
		b.WriteString("  · ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

type offerState struct {
	Step     plan.Step
	Decision decision
}

type decision int

const (
	decisionPending decision = iota
	decisionAccepted
	decisionDeclined
)

func collectOffers(pl plan.Plan) []offerState {
	var out []offerState
	for _, step := range pl.Steps {
		if step.Kind == plan.StepBtrfsOffer {
			out = append(out, offerState{Step: step, Decision: decisionPending})
		}
	}
	return out
}

func nextPendingOffer(offers []offerState) int {
	for i, off := range offers {
		if off.Decision == decisionPending {
			return i
		}
	}
	return 0
}

func (m *model) handleOfferKey(key string) bool {
	if len(m.offers) == 0 || !m.showOffer {
		return false
	}
	switch key {
	case "y", "Y":
		m.setOfferDecision(decisionAccepted)
		return true
	case "n", "N":
		m.setOfferDecision(decisionDeclined)
		return true
	case "tab":
		m.activeOffer = (m.activeOffer + 1) % len(m.offers)
		return true
	case "esc":
		m.showOffer = false
		return true
	}
	return false
}

func (m *model) setOfferDecision(dec decision) {
	if m.activeOffer < 0 || m.activeOffer >= len(m.offers) {
		return
	}
	m.offers[m.activeOffer].Decision = dec
	if m.opts.OnBtrfsDecision != nil {
		accepted := dec == decisionAccepted
		src := m.offers[m.activeOffer].Step.Sources[0]
		dest := m.offers[m.activeOffer].Step.Dest
		m.opts.OnBtrfsDecision(src, dest, accepted)
	}
	if idx := nextPendingOffer(m.offers); m.offers[idx].Decision == decisionPending {
		m.activeOffer = idx
	} else {
		m.showOffer = false
	}
}

func (m model) currentOffer() *offerState {
	if m.activeOffer < 0 || m.activeOffer >= len(m.offers) {
		return nil
	}
	return &m.offers[m.activeOffer]
}

func renderOfferModal(off *offerState) string {
	if off == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nBTRFS OFFER\n")
	b.WriteString(strings.Repeat("-", 20))
	b.WriteString("\n")
	fmt.Fprintf(&b, "Source: %s\nDest:   %s\n", off.Step.Sources[0], off.Step.Dest)
	b.WriteString("[y] accept • [n] decline • [tab] next offer • [esc] close\n")
	switch off.Decision {
	case decisionAccepted:
		b.WriteString("Decision: accepted\n")
	case decisionDeclined:
		b.WriteString("Decision: declined\n")
	default:
		b.WriteString("Decision: pending\n")
	}
	return b.String()
}
