package service

import (
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// Test helpers
func makeWorkEntry(task, project string, mins int) model.WorkEntry {
	return model.WorkEntry{
		ID:          "test-id",
		Task:        task,
		Project:     project,
		DurationMin: mins,
		IsBreak:     false,
	}
}

func TestEntryConsolidator_Consolidate_NoDuplicates(t *testing.T) {
	consolidator := NewEntryConsolidator()

	entries := []model.WorkEntry{
		makeWorkEntry("TaskA", "", 30),
		makeWorkEntry("TaskB", "", 60),
	}

	got := consolidator.Consolidate(entries)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Task != "TaskA" || got[0].DurationMin != 30 {
		t.Errorf("first entry = %+v, want TaskA 30m", got[0])
	}
	if got[1].Task != "TaskB" || got[1].DurationMin != 60 {
		t.Errorf("second entry = %+v, want TaskB 60m", got[1])
	}
}

func TestEntryConsolidator_Consolidate_Duplicates(t *testing.T) {
	consolidator := NewEntryConsolidator()

	entries := []model.WorkEntry{
		makeWorkEntry("TaskA", "", 30),
		makeWorkEntry("TaskA", "", 45),
	}

	got := consolidator.Consolidate(entries)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Task != "TaskA" {
		t.Errorf("task = %q, want %q", got[0].Task, "TaskA")
	}
	if got[0].DurationMin != 75 {
		t.Errorf("duration = %d, want 75", got[0].DurationMin)
	}
}

func TestEntryConsolidator_Consolidate_MultipleDuplicates(t *testing.T) {
	consolidator := NewEntryConsolidator()

	entries := []model.WorkEntry{
		makeWorkEntry("TaskA", "", 10),
		makeWorkEntry("TaskB", "", 20),
		makeWorkEntry("TaskA", "", 15),
		makeWorkEntry("TaskC", "", 5),
		makeWorkEntry("TaskB", "", 25),
		makeWorkEntry("TaskA", "", 20),
	}

	got := consolidator.Consolidate(entries)

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}

	// Check TaskA: 10 + 15 + 20 = 45
	var foundA, foundB, foundC bool
	for _, e := range got {
		switch e.Task {
		case "TaskA":
			foundA = true
			if e.DurationMin != 45 {
				t.Errorf("TaskA duration = %d, want 45", e.DurationMin)
			}
		case "TaskB":
			foundB = true
			if e.DurationMin != 45 {
				t.Errorf("TaskB duration = %d, want 45", e.DurationMin)
			}
		case "TaskC":
			foundC = true
			if e.DurationMin != 5 {
				t.Errorf("TaskC duration = %d, want 5", e.DurationMin)
			}
		}
	}

	if !foundA || !foundB || !foundC {
		t.Errorf("missing tasks: foundA=%v, foundB=%v, foundC=%v", foundA, foundB, foundC)
	}
}

func TestEntryConsolidator_Consolidate_EmptyInput(t *testing.T) {
	consolidator := NewEntryConsolidator()

	got := consolidator.Consolidate([]model.WorkEntry{})

	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(got))
	}
}

func TestEntryConsolidator_Consolidate_CaseSensitive(t *testing.T) {
	consolidator := NewEntryConsolidator()

	entries := []model.WorkEntry{
		makeWorkEntry("taskA", "", 30),
		makeWorkEntry("TaskA", "", 40),
	}

	got := consolidator.Consolidate(entries)

	// Should NOT consolidate - case sensitive
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (case sensitive)", len(got))
	}
}

func TestEntryConsolidator_ConsolidateByProject(t *testing.T) {
	consolidator := NewEntryConsolidator()

	entries := []model.WorkEntry{
		makeWorkEntry("Review", "ProjectA", 30),
		makeWorkEntry("Review", "ProjectA", 20),
		makeWorkEntry("Review", "ProjectB", 10),
		makeWorkEntry("Coding", "ProjectA", 40),
	}

	got := consolidator.ConsolidateByProject(entries)

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}

	// Verify consolidation by project+task
	for _, e := range got {
		if e.Project == "ProjectA" && e.Task == "Review" {
			if e.DurationMin != 50 {
				t.Errorf("ProjectA Review duration = %d, want 50", e.DurationMin)
			}
		} else if e.Project == "ProjectB" && e.Task == "Review" {
			if e.DurationMin != 10 {
				t.Errorf("ProjectB Review duration = %d, want 10", e.DurationMin)
			}
		} else if e.Project == "ProjectA" && e.Task == "Coding" {
			if e.DurationMin != 40 {
				t.Errorf("ProjectA Coding duration = %d, want 40", e.DurationMin)
			}
		} else {
			t.Errorf("unexpected entry: %+v", e)
		}
	}
}

func TestEntryConsolidator_ConsolidateByProject_EmptyProject(t *testing.T) {
	consolidator := NewEntryConsolidator()

	entries := []model.WorkEntry{
		makeWorkEntry("TaskA", "", 30),
		makeWorkEntry("TaskA", "", 20),
		makeWorkEntry("TaskA", "ProjectX", 10),
	}

	got := consolidator.ConsolidateByProject(entries)

	// Empty project and "ProjectX" should be treated differently
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	for _, e := range got {
		if e.Project == "" && e.Task == "TaskA" {
			if e.DurationMin != 50 {
				t.Errorf("empty project TaskA duration = %d, want 50", e.DurationMin)
			}
		} else if e.Project == "ProjectX" && e.Task == "TaskA" {
			if e.DurationMin != 10 {
				t.Errorf("ProjectX TaskA duration = %d, want 10", e.DurationMin)
			}
		} else {
			t.Errorf("unexpected entry: %+v", e)
		}
	}
}
