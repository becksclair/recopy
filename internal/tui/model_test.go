package tui

import (
	"fmt"
	"strings"
	"testing"

	"recopy/internal/plan"
)

func TestViewDisplaysWorkerCount(t *testing.T) {
	pl := plan.Plan{Steps: []plan.Step{{Kind: plan.StepRsync, Sources: []string{"src"}, Dest: "dest"}}}
	m := newModel(RunOptions{Plan: pl, Mode: "copy", Profile: "auto", Transport: "auto", Workers: 4})
	layer, ok := m.View().Layer.(fmt.Stringer)
	if !ok {
		t.Fatalf("view layer missing string content")
	}
	view := layer.String()
	if !strings.Contains(view, "WORKERS 4") {
		t.Fatalf("expected worker count in view, got %q", view)
	}
}
