package rsync

import (
	"bufio"
	"math"
	"os"
	"testing"
	"time"
)

func TestParseProgress2(t *testing.T) {
	line := "  5,242,880  12%   32.61MB/s    0:00:03 (xfr#2, to-chk=3/5)"
	sample, ok := parseProgress2(line)
	if !ok {
		t.Fatalf("expected progress sample")
	}
	if sample.Bytes != 5242880 {
		t.Fatalf("bytes mismatch: %d", sample.Bytes)
	}
	if sample.Percent != 12 {
		t.Fatalf("percent mismatch: %d", sample.Percent)
	}
	if math.Abs(sample.RateBytesPerSec-34194063.36) > 1 {
		t.Fatalf("rate mismatch: %f", sample.RateBytesPerSec)
	}
	if sample.Eta != 3*time.Second {
		t.Fatalf("eta mismatch: %s", sample.Eta)
	}
	if sample.TransferCount != 2 || sample.ToCheckRemaining != 3 || sample.ToCheckTotal != 5 {
		t.Fatalf("counts mismatch: %+v", sample)
	}
}

func TestParseItemEvent(t *testing.T) {
	line := ">f+++++++++|123|dir/file"
	event, ok := parseItemEvent(line)
	if !ok {
		t.Fatalf("expected item event")
	}
	if event.Change != ">f+++++++++" || event.Length != 123 || event.Path != "dir/file" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestParseItemEventSymlink(t *testing.T) {
	line := "cL+++++++++|1|link -> target"
	event, ok := parseItemEvent(line)
	if !ok {
		t.Fatalf("expected event")
	}
	if event.Path != "link" || event.Target != "target" {
		t.Fatalf("unexpected symlink parse: %+v", event)
	}
}

func TestParserFeed(t *testing.T) {
	parser := Parser{}
	lines := []string{
		"sending incremental file list",
		">f+++++++++|11|docs/README.md",
		"  5,242,880  12%   32.61MB/s    0:00:03 (xfr#1, to-chk=4/5)",
		">f..t......|2048|docs/link -> ../target",
	}
	var items int
	var progressSeen bool
	for _, line := range lines {
		event, ok := parser.Feed(line)
		if !ok {
			continue
		}
		if event.Item != nil {
			items++
		}
		if event.Progress != nil {
			progressSeen = true
		}
	}
	if items != 2 || !progressSeen {
		t.Fatalf("expected 2 items + progress, got items=%d progress=%v", items, progressSeen)
	}
}

func TestParserFeedGolden(t *testing.T) {
	f, err := os.Open("testdata/session.log")
	if err != nil {
		t.Fatalf("open session log: %v", err)
	}
	defer f.Close()

	parser := Parser{}
	scanner := bufio.NewScanner(f)
	var progressSamples int
	var itemPaths []string
	for scanner.Scan() {
		line := scanner.Text()
		event, ok := parser.Feed(line)
		if !ok {
			continue
		}
		if event.Progress != nil {
			progressSamples++
		}
		if event.Item != nil {
			itemPaths = append(itemPaths, event.Item.Path)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if progressSamples != 2 {
		t.Fatalf("expected 2 progress samples, got %d", progressSamples)
	}
	if len(itemPaths) != 2 {
		t.Fatalf("expected 2 item events, got %d", len(itemPaths))
	}
	if itemPaths[0] != "docs/README.md" || itemPaths[1] != "docs/link" {
		t.Fatalf("unexpected paths: %v", itemPaths)
	}
}
