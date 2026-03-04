package journal

import (
	"testing"
	"time"
)

func TestClockEntriesSingleProject(t *testing.T) {
	entries := ClockEntries("feature dev", "Backend", 90*time.Minute)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Task != "feature dev" {
		t.Errorf("Task = %q, want %q", e.Task, "feature dev")
	}
	if e.Project != "Backend" {
		t.Errorf("Project = %q, want %q", e.Project, "Backend")
	}
	if e.DurationMin != 90 {
		t.Errorf("DurationMin = %d, want 90", e.DurationMin)
	}
	if e.IsBreak {
		t.Error("IsBreak should be false")
	}
}

func TestClockEntriesNoProject(t *testing.T) {
	entries := ClockEntries("meeting", "", 30*time.Minute)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Project != "" {
		t.Errorf("Project = %q, want empty", entries[0].Project)
	}
	if entries[0].DurationMin != 30 {
		t.Errorf("DurationMin = %d, want 30", entries[0].DurationMin)
	}
}

func TestClockEntriesTwoProjects(t *testing.T) {
	entries := ClockEntries("stand-up", "Alpha, Beta", 60*time.Minute)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	total := 0
	for _, e := range entries {
		if e.Task != "stand-up" {
			t.Errorf("Task = %q, want %q", e.Task, "stand-up")
		}
		total += e.DurationMin
	}
	if total != 60 {
		t.Errorf("total DurationMin = %d, want 60", total)
	}
}

func TestClockEntriesThreeProjects(t *testing.T) {
	entries := ClockEntries("planning", "A, B, C", 90*time.Minute)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	total := 0
	for _, e := range entries {
		total += e.DurationMin
	}
	if total != 90 {
		t.Errorf("total DurationMin = %d, want 90", total)
	}
}

func TestClockEntriesUnevenSplit(t *testing.T) {
	// 61 minutes across 2 projects: first gets 31, second gets 30.
	entries := ClockEntries("task", "A, B", 61*time.Minute)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].DurationMin != 31 {
		t.Errorf("first entry DurationMin = %d, want 31", entries[0].DurationMin)
	}
	if entries[1].DurationMin != 30 {
		t.Errorf("second entry DurationMin = %d, want 30", entries[1].DurationMin)
	}
	if entries[0].Project != "A" {
		t.Errorf("first entry Project = %q, want A", entries[0].Project)
	}
	if entries[1].Project != "B" {
		t.Errorf("second entry Project = %q, want B", entries[1].Project)
	}
}

func TestClockEntriesSubMinute(t *testing.T) {
	// elapsed < 1 minute → no entries
	entries := ClockEntries("quick task", "", 30*time.Second)
	if len(entries) != 0 {
		t.Errorf("expected no entries for sub-minute elapsed, got %d", len(entries))
	}
}

func TestClockEntriesZeroDuration(t *testing.T) {
	entries := ClockEntries("task", "Project", 0)
	if len(entries) != 0 {
		t.Errorf("expected no entries for zero elapsed, got %d", len(entries))
	}
}

func TestClockEntriesDurationTooShortForProjects(t *testing.T) {
	// 1 minute across 3 projects: base=0, remainder=1 → only first project gets 1 min.
	entries := ClockEntries("task", "A, B, C", 1*time.Minute)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (others dropped), got %d", len(entries))
	}
	if entries[0].Project != "A" {
		t.Errorf("Project = %q, want A", entries[0].Project)
	}
	if entries[0].DurationMin != 1 {
		t.Errorf("DurationMin = %d, want 1", entries[0].DurationMin)
	}
}

func TestClockEntriesWhitespaceProjects(t *testing.T) {
	// Extra whitespace around project names should be trimmed.
	entries := ClockEntries("work", "  Frontend ,  Backend  ", 60*time.Minute)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Project != "Frontend" {
		t.Errorf("first Project = %q, want Frontend", entries[0].Project)
	}
	if entries[1].Project != "Backend" {
		t.Errorf("second Project = %q, want Backend", entries[1].Project)
	}
}

func TestClockEntriesIDsAreUnique(t *testing.T) {
	entries := ClockEntries("task", "A, B, C", 90*time.Minute)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.ID == "" {
			t.Error("entry has empty ID")
		}
		if seen[e.ID] {
			t.Errorf("duplicate ID %q", e.ID)
		}
		seen[e.ID] = true
	}
}
