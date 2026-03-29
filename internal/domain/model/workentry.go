package model

import (
	"fmt"
	"time"
)

// WorkEntry represents a single work or break item within a day.
// This is a pure domain entity with no infrastructure dependencies.
//
// Design Decision: No JSON tags here - serialization is handled by
// infrastructure layer DTOs. This keeps the domain clean and independent.
type WorkEntry struct {
	ID          string
	Project     string
	Task        string
	DurationMin int
	IsBreak     bool
}

// Duration returns the duration as time.Duration.
func (e WorkEntry) Duration() time.Duration {
	return time.Duration(e.DurationMin) * time.Minute
}

// Validate checks if this WorkEntry is valid according to business rules.
func (e WorkEntry) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("id is required")
	}
	if e.Task == "" {
		return fmt.Errorf("task is required")
	}
	if e.DurationMin <= 0 {
		return fmt.Errorf("duration must be positive, got %d", e.DurationMin)
	}
	return nil
}

// IsWork returns true if this is a work entry (not a break).
func (e WorkEntry) IsWork() bool {
	return !e.IsBreak
}
