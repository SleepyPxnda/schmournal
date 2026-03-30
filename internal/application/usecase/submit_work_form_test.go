package usecase

import (
	"testing"
	"time"
)

func TestSubmitWorkForm_AddWorkSplitsProjects(t *testing.T) {
	repo := NewMockDayRecordRepository()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC))
	uc := NewSubmitWorkFormUseCase(repo, timeProvider)

	out, err := uc.Execute(SubmitWorkFormInput{
		Date:       "2026-03-30",
		Task:       "Feature work",
		ProjectRaw: "API, UI",
		Duration:   "1h 1m",
		IsBreak:    false,
		EditEntry:  -1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Label != "✓ Work entries split across projects" {
		t.Fatalf("unexpected label: %q", out.Label)
	}
	if len(out.Record.Entries) != 2 {
		t.Fatalf("expected 2 split entries, got %d", len(out.Record.Entries))
	}
	if out.Record.Entries[0].DurationMin+out.Record.Entries[1].DurationMin != 61 {
		t.Fatalf("expected total split duration to equal 61, got %d", out.Record.Entries[0].DurationMin+out.Record.Entries[1].DurationMin)
	}
	if out.SelectedEntryIdx != 1 {
		t.Fatalf("expected selected index 1, got %d", out.SelectedEntryIdx)
	}
}

func TestSubmitWorkForm_AddBreakMergesCaseInsensitiveTask(t *testing.T) {
	repo := NewMockDayRecordRepository()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC))
	if err := repo.Save(mapDayRecordDTOToDomain(DayRecordDTO{
		Date: "2026-03-30",
		Entries: []WorkEntryDTO{
			{ID: "b1", Task: "Lunch", DurationMin: 30, IsBreak: true},
		},
	})); err != nil {
		t.Fatalf("failed to seed repo: %v", err)
	}

	uc := NewSubmitWorkFormUseCase(repo, timeProvider)
	out, err := uc.Execute(SubmitWorkFormInput{
		Date:      "2026-03-30",
		Task:      "lunch",
		Duration:  "15m",
		IsBreak:   true,
		EditEntry: -1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Label != "✓ Break logged" {
		t.Fatalf("unexpected label: %q", out.Label)
	}
	if len(out.Record.Entries) != 1 {
		t.Fatalf("expected merged break entry, got %d entries", len(out.Record.Entries))
	}
	if out.Record.Entries[0].DurationMin != 45 {
		t.Fatalf("expected merged duration 45, got %d", out.Record.Entries[0].DurationMin)
	}
	if out.SelectedEntryIdx != 0 {
		t.Fatalf("expected selected index 0, got %d", out.SelectedEntryIdx)
	}
}

func TestSubmitWorkForm_EditSingleWorkEntryReturnsUpdatedLabel(t *testing.T) {
	repo := NewMockDayRecordRepository()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC))
	if err := repo.Save(mapDayRecordDTOToDomain(DayRecordDTO{
		Date: "2026-03-30",
		Entries: []WorkEntryDTO{
			{ID: "w1", Task: "Old task", Project: "Backend", DurationMin: 30, IsBreak: false},
		},
	})); err != nil {
		t.Fatalf("failed to seed repo: %v", err)
	}

	uc := NewSubmitWorkFormUseCase(repo, timeProvider)
	out, err := uc.Execute(SubmitWorkFormInput{
		Date:       "2026-03-30",
		Task:       "Updated task",
		ProjectRaw: "Backend",
		Duration:   "45m",
		IsBreak:    false,
		EditEntry:  0,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Label != "✓ Entry updated" {
		t.Fatalf("unexpected label: %q", out.Label)
	}
	if len(out.Record.Entries) != 1 || out.Record.Entries[0].Task != "Updated task" || out.Record.Entries[0].DurationMin != 45 {
		t.Fatalf("expected edited entry in output, got %+v", out.Record.Entries)
	}
	if out.SelectedEntryIdx != 0 {
		t.Fatalf("expected selected index 0, got %d", out.SelectedEntryIdx)
	}
}

