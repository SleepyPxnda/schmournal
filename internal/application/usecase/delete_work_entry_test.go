package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestDeleteWorkEntry_Success(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewDeleteWorkEntryUseCase(repo)

	// Add two entries
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	addUseCase := NewAddWorkEntryUseCase(repo, timeProvider)
	output1, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Task 1",
		DurationMin: 60,
	})
	output2, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Task 2",
		DurationMin: 30,
	})

	// Delete the first entry
	input := DeleteWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: output1.EntryID,
	}

	deleteOutput, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if deleteOutput.RemainingCount != 1 {
		t.Errorf("expected 1 remaining entry, got %d", deleteOutput.RemainingCount)
	}
	if deleteOutput.TotalWork != 30 {
		t.Errorf("expected total work 30, got %d", deleteOutput.TotalWork)
	}

	// Verify the correct entry was deleted
	saved, _ := repo.FindByDate("2026-03-28")
	if len(saved.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(saved.Entries))
	}
	if saved.Entries[0].ID != output2.EntryID {
		t.Error("expected Task 2 to remain")
	}
}

func TestDeleteWorkEntry_LastEntry_DeletesRecord(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewDeleteWorkEntryUseCase(repo)

	// Add one entry
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	addUseCase := NewAddWorkEntryUseCase(repo, timeProvider)
	output, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Only Task",
		DurationMin: 60,
	})

	// Delete the only entry
	input := DeleteWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: output.EntryID,
	}

	deleteOutput, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if deleteOutput.RemainingCount != 0 {
		t.Errorf("expected 0 remaining entries, got %d", deleteOutput.RemainingCount)
	}

	// Verify record was deleted (should return empty record)
	saved, _ := repo.FindByDate("2026-03-28")
	if len(saved.Entries) != 0 {
		t.Error("expected record to be deleted")
	}
}

func TestDeleteWorkEntry_LastEntry_KeepsRecordWithNotes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewDeleteWorkEntryUseCase(repo)

	// Create a record with one entry and notes
	record := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "entry-1", Task: "Task", DurationMin: 60},
		},
		Notes: "Important notes!",
	}
	_ = repo.Save(record)

	// Delete the only entry
	input := DeleteWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: "entry-1",
	}

	deleteOutput, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if deleteOutput.RemainingCount != 0 {
		t.Errorf("expected 0 remaining entries, got %d", deleteOutput.RemainingCount)
	}

	// Verify record was NOT deleted (notes should keep it alive)
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Notes != "Important notes!" {
		t.Error("expected record to be kept because of notes")
	}
	if len(saved.Entries) != 0 {
		t.Error("expected entries to be empty")
	}
}

func TestDeleteWorkEntry_LastEntry_KeepsRecordWithTimes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewDeleteWorkEntryUseCase(repo)

	// Create a record with one entry and day times
	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
		Entries: []model.WorkEntry{
			{ID: "entry-1", Task: "Task", DurationMin: 60},
		},
	}
	_ = repo.Save(record)

	// Delete the only entry
	input := DeleteWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: "entry-1",
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify record was NOT deleted (times should keep it alive)
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.StartTime != "09:00" || saved.EndTime != "17:00" {
		t.Error("expected record to be kept because of times")
	}
}

func TestDeleteWorkEntry_EntryNotFound(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewDeleteWorkEntryUseCase(repo)

	// Add a record with one entry
	record := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "existing-id", Task: "Task", DurationMin: 60},
		},
	}
	_ = repo.Save(record)

	// Try to delete non-existent entry
	input := DeleteWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: "non-existent-id",
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestDeleteWorkEntry_ValidationErrors(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewDeleteWorkEntryUseCase(repo)

	tests := []struct {
		name  string
		input DeleteWorkEntryInput
	}{
		{
			name: "missing date",
			input: DeleteWorkEntryInput{
				Date:    "",
				EntryID: "id",
			},
		},
		{
			name: "missing entry ID",
			input: DeleteWorkEntryInput{
				Date:    "2026-03-28",
				EntryID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := useCase.Execute(tt.input)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}
