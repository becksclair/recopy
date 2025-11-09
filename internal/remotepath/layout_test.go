package remotepath

import "testing"

func TestClassifyPaths(t *testing.T) {
	sources := []string{"user@host:/src/a", "user@host:/src/b"}
	l, err := ClassifyPaths(sources, "/dest")
	if err != nil {
		t.Fatalf("classify failed: %v", err)
	}
	if l.Side != SideSource {
		t.Fatalf("expected source remote, got %v", l.Side)
	}
	if l.Spec.Host != "host" || l.Spec.User != "user" {
		t.Fatalf("unexpected spec: %+v", l.Spec)
	}
}

func TestClassifyRejectsMixedSources(t *testing.T) {
	sources := []string{"user@host:/src", "local"}
	if _, err := ClassifyPaths(sources, "/dest"); err == nil {
		t.Fatalf("expected error for mixed sources")
	}
}

func TestClassifyDestRemote(t *testing.T) {
	sources := []string{"local"}
	l, err := ClassifyPaths(sources, "host:/dest")
	if err != nil {
		t.Fatalf("classify failed: %v", err)
	}
	if l.Side != SideDest {
		t.Fatalf("expected dest remote, got %v", l.Side)
	}
}
