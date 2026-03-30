package model

import (
	"fmt"
	"time"
)

// DayRecord represents a single work day with entries, times, and notes.
// This is a pure domain entity with no infrastructure dependencies.
//
// Design Decision: Todos are kept separate in WorkspaceTodos (they are
// workspace-level, not day-level) and are not persisted with DayRecord.
type DayRecord struct {
	Date      string
	StartTime string // HH:MM format
	EndTime   string // HH:MM format
	Entries   []WorkEntry
	Notes     string
}

// WorkTotals calculates the total work time, break time, and overall time
// from all entries in this day record.
// This is core business logic that belongs in the domain.
func (r DayRecord) WorkTotals() (work, breaks, total time.Duration) {
	for _, e := range r.Entries {
		if e.IsBreak {
			breaks += e.Duration()
		} else {
			work += e.Duration()
		}
	}
	total = work + breaks
	return
}

// DayDuration calculates the total duration between StartTime and EndTime.
// Returns (duration, ok) where ok=false if times are invalid or missing.
func (r DayRecord) DayDuration() (time.Duration, bool) {
	if r.StartTime == "" || r.EndTime == "" {
		return 0, false
	}
	s, err1 := time.Parse("15:04", r.StartTime)
	e, err2 := time.Parse("15:04", r.EndTime)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	d := e.Sub(s)
	if d <= 0 {
		return 0, false
	}
	return d, true
}

// Summary returns a human-readable summary of this day record.
// Requires a DurationFormatter to format durations (Dependency Inversion).
func (r DayRecord) Summary(formatter DurationFormatter) string {
	n := len(r.Entries)
	if n == 0 {
		return "No entries"
	}
	word := "entries"
	if n == 1 {
		word = "entry"
	}
	work, breaks, _ := r.WorkTotals()
	s := fmt.Sprintf("%d %s", n, word)
	if work > 0 {
		s += "  ·  Work: " + formatter.Format(work)
	}
	if breaks > 0 {
		s += "  ·  Breaks: " + formatter.Format(breaks)
	}
	return s
}

// ParseDate parses the Date field into a time.Time.
func (r DayRecord) ParseDate() (time.Time, error) {
	return time.Parse("2006-01-02", r.Date)
}

// Validate checks if this DayRecord is valid according to business rules.
func (r DayRecord) Validate() error {
	if r.Date == "" {
		return fmt.Errorf("date is required")
	}
	if _, err := r.ParseDate(); err != nil {
		return fmt.Errorf("invalid date format %q: %w", r.Date, err)
	}
	for i, e := range r.Entries {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("entry %d invalid: %w", i, err)
		}
	}
	// Validate time formats if provided
	if r.StartTime != "" {
		if _, err := time.Parse("15:04", r.StartTime); err != nil {
			return fmt.Errorf("invalid start time %q: %w", r.StartTime, err)
		}
	}
	if r.EndTime != "" {
		if _, err := time.Parse("15:04", r.EndTime); err != nil {
			return fmt.Errorf("invalid end time %q: %w", r.EndTime, err)
		}
	}
	return nil
}

// AddEntry adds a work entry to this day record.
// This is a convenience method for domain operations.
func (r *DayRecord) AddEntry(entry WorkEntry) error {
	if err := entry.Validate(); err != nil {
		return fmt.Errorf("cannot add invalid entry: %w", err)
	}
	r.Entries = append(r.Entries, entry)
	return nil
}

// RemoveEntry removes an entry by ID.
// Returns true if an entry was removed, false if not found.
func (r *DayRecord) RemoveEntry(id string) bool {
	for i, e := range r.Entries {
		if e.ID == id {
			r.Entries = append(r.Entries[:i], r.Entries[i+1:]...)
			return true
		}
	}
	return false
}

// FindEntryByID finds an entry by ID.
// Returns nil if not found.
func (r *DayRecord) FindEntryByID(id string) *WorkEntry {
	for i := range r.Entries {
		if r.Entries[i].ID == id {
			return &r.Entries[i]
		}
	}
	return nil
}
