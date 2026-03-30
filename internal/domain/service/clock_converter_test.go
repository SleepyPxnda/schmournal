package service

import (
	"testing"
	"time"
)

func TestClockConverter_SingleProject(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	entries := converter.ConvertToEntries("feature dev", "Backend", 90*time.Minute)
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

func TestClockConverter_NoProject(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	entries := converter.ConvertToEntries("meeting", "", 30*time.Minute)
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

func TestClockConverter_TwoProjects(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	entries := converter.ConvertToEntries("stand-up", "Alpha, Beta", 60*time.Minute)
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

func TestClockConverter_ThreeProjects(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	entries := converter.ConvertToEntries("planning", "A, B, C", 90*time.Minute)
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

func TestClockConverter_RemainderDistribution(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	// 91 minutes split across 2 projects: 46m + 45m
	entries := converter.ConvertToEntries("coding", "P1, P2", 91*time.Minute)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// First project gets the extra minute
	if entries[0].DurationMin != 46 {
		t.Errorf("first entry DurationMin = %d, want 46", entries[0].DurationMin)
	}
	if entries[1].DurationMin != 45 {
		t.Errorf("second entry DurationMin = %d, want 45", entries[1].DurationMin)
	}
}

func TestClockConverter_ZeroMinutes(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	// 30 seconds = 0 minutes (rounded down)
	entries := converter.ConvertToEntries("quick check", "P", 30*time.Second)
	if entries != nil {
		t.Errorf("expected nil entries for <1 minute, got %d entries", len(entries))
	}
}

func TestClockConverter_WhitespaceInProjects(t *testing.T) {
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	converter := NewClockConverter(timeProvider)

	entries := converter.ConvertToEntries("task", "  Alpha  ,  Beta  ", 60*time.Minute)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Project != "Alpha" {
		t.Errorf("first project = %q, want %q", entries[0].Project, "Alpha")
	}
	if entries[1].Project != "Beta" {
		t.Errorf("second project = %q, want %q", entries[1].Project, "Beta")
	}
}
