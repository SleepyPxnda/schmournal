package model

import (
	"fmt"
	"time"
)

// DurationFormatter defines how durations should be formatted for display.
// This interface allows the domain to format durations without depending
// on specific formatting implementations (which may vary by locale, etc.).
//
// Design Decision: Domain entities need to format durations for Summary(),
// but formatting logic should be injectable (Dependency Inversion).
type DurationFormatter interface {
	Format(d time.Duration) string
}

// DefaultDurationFormatter provides the standard "1h 30m" format.
type DefaultDurationFormatter struct{}

// Format converts a duration to "1h 30m" format.
func (f DefaultDurationFormatter) Format(d time.Duration) string {
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
