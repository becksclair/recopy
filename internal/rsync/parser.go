package rsync

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parser consumes rsync progress2 and out-format lines, emitting structured events.
type Parser struct{}

// Event represents either a progress sample or an item event.
type Event struct {
	Progress *ProgressSample
	Item     *ItemEvent
}

// ProgressSample captures aggregated stats from --info=progress2.
type ProgressSample struct {
	Bytes            int64
	Percent          int
	RateBytesPerSec  float64
	Eta              time.Duration
	TransferCount    int
	ToCheckRemaining int
	ToCheckTotal     int
}

// ItemEvent mirrors fields from --out-format=%i|%l|%n%L.
type ItemEvent struct {
	Change string
	Length int64
	Path   string
	Target string
}

var progress2Regex = regexp.MustCompile(`^\s*([\d,]+)\s+(\d+)%\s+([\d.]+)([kMGT]?)(?:i?B)/s\s+(\d+):(\d+):(\d+)\s+\(xfr#(\d+),\s+to-chk=(\d+)/(\d+)\)`)

// Feed ingests a single line of rsync output and emits an Event if recognized.
func (Parser) Feed(line string) (Event, bool) {
	line = strings.TrimLeft(line, "\r")
	if progress, ok := parseProgress2(line); ok {
		return Event{Progress: &progress}, true
	}
	if item, ok := parseItemEvent(line); ok {
		return Event{Item: &item}, true
	}
	return Event{}, false
}

func parseProgress2(line string) (ProgressSample, bool) {
	matches := progress2Regex.FindStringSubmatch(line)
	if matches == nil {
		return ProgressSample{}, false
	}
	bytesVal, err := parseInt(strings.ReplaceAll(matches[1], ",", ""))
	if err != nil {
		return ProgressSample{}, false
	}
	percent, err := strconv.Atoi(matches[2])
	if err != nil {
		return ProgressSample{}, false
	}
	rate, err := parseRate(matches[3], matches[4])
	if err != nil {
		return ProgressSample{}, false
	}
	hours, _ := strconv.Atoi(matches[5])
	minutes, _ := strconv.Atoi(matches[6])
	seconds, _ := strconv.Atoi(matches[7])
	eta := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	transferCount, _ := strconv.Atoi(matches[8])
	toChkRem, _ := strconv.Atoi(matches[9])
	toChkTotal, _ := strconv.Atoi(matches[10])
	return ProgressSample{
		Bytes:            bytesVal,
		Percent:          percent,
		RateBytesPerSec:  rate,
		Eta:              eta,
		TransferCount:    transferCount,
		ToCheckRemaining: toChkRem,
		ToCheckTotal:     toChkTotal,
	}, true
}

func parseRate(value, unit string) (float64, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	switch strings.ToUpper(unit) {
	case "K":
		f *= 1024
	case "M":
		f *= 1024 * 1024
	case "G":
		f *= 1024 * 1024 * 1024
	case "T":
		f *= 1024 * 1024 * 1024 * 1024
	}
	return f, nil
}

func parseItemEvent(line string) (ItemEvent, bool) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) != 3 {
		return ItemEvent{}, false
	}
	length, err := parseInt(parts[1])
	if err != nil {
		return ItemEvent{}, false
	}
	path := parts[2]
	target := ""
	if idx := strings.Index(path, " -> "); idx != -1 {
		target = strings.TrimSpace(path[idx+4:])
		path = path[:idx]
	}
	return ItemEvent{
		Change: parts[0],
		Length: length,
		Path:   path,
		Target: target,
	}, true
}

func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
