package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DurationParser provides flexible duration string parsing.
// This is a domain service - it contains business logic for parsing
// user-friendly duration inputs like "1h 30m", "90m", "1.5h".
//
// Design Decision: This is a stateless service, implemented as functions.
// No need for a struct since there's no state to maintain.
type DurationParser struct{}

// NewDurationParser creates a new DurationParser.
func NewDurationParser() *DurationParser {
	return &DurationParser{}
}

// Parse parses flexible duration strings like "1h 30m", "90m", "1.5h", "45m".
// This supports multiple formats for better UX:
// - Standard Go format: "1h30m", "90m"
// - Spaced format: "1h 30m", "1h 30min"
// - Decimal hours: "1.5h"
// - Minutes only: "90", "90m", "90min"
func (p *DurationParser) Parse(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "—" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try standard time.ParseDuration first (handles "1h30m", "90m", etc.)
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try "1h 30m" format (with space)
	var hours, mins float64
	if n, _ := fmt.Sscanf(s, "%fh %fm", &hours, &mins); n == 2 {
		return time.Duration((hours*60+mins)*float64(time.Minute)), nil
	}
	// Try "1h 30min" format
	if n, _ := fmt.Sscanf(s, "%fh %fmin", &hours, &mins); n == 2 {
		return time.Duration((hours*60+mins)*float64(time.Minute)), nil
	}

	// Try hours-only with decimal (e.g. "1.5h")
	if strings.HasSuffix(s, "h") {
		if h, err := strconv.ParseFloat(strings.TrimSuffix(s, "h"), 64); err == nil {
			return time.Duration(h * float64(time.Hour)), nil
		}
	}

	// Try minutes-only (e.g. "90m", "90min", or just "90")
	s2 := strings.TrimSuffix(strings.TrimSuffix(s, "min"), "m")
	if m, err := strconv.ParseFloat(strings.TrimSpace(s2), 64); err == nil && m > 0 {
		return time.Duration(m * float64(time.Minute)), nil
	}

	return 0, fmt.Errorf("cannot parse duration %q – try: 1h 30m, 45m, 2h", s)
}

// DurationFormatter formats durations as human-readable strings.
// This implements the DurationFormatter interface from the model package.
type DurationFormatter struct{}

// NewDurationFormatter creates a new DurationFormatter.
func NewDurationFormatter() *DurationFormatter {
	return &DurationFormatter{}
}

// Format converts a duration to a human-readable string like "1h 30m".
func (f *DurationFormatter) Format(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}
