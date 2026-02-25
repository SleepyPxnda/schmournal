package journal

import (
	"fmt"
	"time"
)

// WorkEntry is a single work or break item within a day.
type WorkEntry struct {
	ID          string `json:"id"`
	Project     string `json:"project"`
	Task        string `json:"task"`
	DurationMin int    `json:"duration_min"`
	IsBreak     bool   `json:"is_break"`
}

func (e WorkEntry) Duration() time.Duration { return time.Duration(e.DurationMin) * time.Minute }

// DayRecord holds all data for a single work day.
type DayRecord struct {
	Date      string      `json:"date"`
	StartTime string      `json:"start_time"`
	EndTime   string      `json:"end_time"`
	Entries   []WorkEntry `json:"entries"`
	Notes     string      `json:"notes"`
	Path      string      `json:"-"` // runtime only
}

func (r DayRecord) ParseDate() (time.Time, error) { return time.Parse("2006-01-02", r.Date) }

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

func (r DayRecord) Summary() string {
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
		s += "  ·  Work: " + FormatDuration(work)
	}
	if breaks > 0 {
		s += "  ·  Breaks: " + FormatDuration(breaks)
	}
	return s
}
