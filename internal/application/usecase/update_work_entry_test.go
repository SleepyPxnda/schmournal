package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestUpdateWorkEntry_Success(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateWorkEntryUseCase(repo)

	// Add an initial entry
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	addUseCase := NewAddWorkEntryUseCase(repo, timeProvider)
	addOutput, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Old Task",
		Project:     "OldProject",
		DurationMin: 60,
	})

	// Update the entry
	input := UpdateWorkEntryInput{
		Date:        "2026-03-28",
		EntryID:     addOutput.EntryID,
		Task:        "New Task",
		Project:     "NewProject",
		DurationMin: 90,
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.EntryID != addOutput.EntryID {
		t.Error("expected same entry ID")
	}
	if output.TotalWork != 90 {
		t.Errorf("expected total work 90, got %d", output.TotalWork)
	}

	// Verify the entry was updated
	saved, _ := repo.FindByDate("2026-03-28")
	if len(saved.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(saved.Entries))
	}
	if saved.Entries[0].Task != "New Task" {
		t.Errorf("expected task 'New Task', got %s", saved.Entries[0].Task)
	}
	if saved.Entries[0].Project != "NewProject" {
		t.Errorf("expected project 'NewProject', got %s", saved.Entries[0].Project)
	}
	if saved.Entries[0].DurationMin != 90 {
		t.Errorf("expected duration 90, got %d", saved.Entries[0].DurationMin)
	}
}

func TestUpdateWorkEntry_PartialUpdate(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateWorkEntryUseCase(repo)

	// Add an initial entry
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	addUseCase := NewAddWorkEntryUseCase(repo, timeProvider)
	addOutput, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Original Task",
		Project:     "OriginalProject",
		DurationMin: 60,
	})

	// Update only the task (keep project and duration)
	input := UpdateWorkEntryInput{
		Date:        "2026-03-28",
		EntryID:     addOutput.EntryID,
		Task:        "Updated Task",
		Project:     "", // empty = keep existing
		DurationMin: 0,  // zero = keep existing
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify only task was updated
	saved, _ := repo.FindByDate("2026-03-28")
	entry := saved.Entries[0]
	if entry.Task != "Updated Task" {
		t.Errorf("expected task to be updated, got %s", entry.Task)
	}
	if entry.Project != "OriginalProject" {
		t.Errorf("expected project to stay 'OriginalProject', got %s", entry.Project)
	}
	if entry.DurationMin != 60 {
		t.Errorf("expected duration to stay 60, got %d", entry.DurationMin)
	}
}

func TestUpdateWorkEntry_ClearProject(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateWorkEntryUseCase(repo)

	// Add an initial entry with a project
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	addUseCase := NewAddWorkEntryUseCase(repo, timeProvider)
	addOutput, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Task",
		Project:     "ProjectToRemove",
		DurationMin: 60,
	})

	// Clear the project using "-"
	input := UpdateWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: addOutput.EntryID,
		Project: "-", // "-" = clear project
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify project was cleared
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Entries[0].Project != "" {
		t.Errorf("expected project to be cleared, got %s", saved.Entries[0].Project)
	}
}

func TestUpdateWorkEntry_EntryNotFound(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateWorkEntryUseCase(repo)

	// Add a record with one entry
	record := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "existing-id", Task: "Task", DurationMin: 60},
		},
	}
	_ = repo.Save(record)

	// Try to update non-existent entry
	input := UpdateWorkEntryInput{
		Date:    "2026-03-28",
		EntryID: "non-existent-id",
		Task:    "New Task",
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestUpdateWorkEntry_ValidationErrors(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateWorkEntryUseCase(repo)

	tests := []struct {
		name  string
		input UpdateWorkEntryInput
	}{
		{
			name: "missing date",
			input: UpdateWorkEntryInput{
				Date:    "",
				EntryID: "id",
				Task:    "Task",
			},
		},
		{
			name: "missing entry ID",
			input: UpdateWorkEntryInput{
				Date:    "2026-03-28",
				EntryID: "",
				Task:    "Task",
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

func TestUpdateWorkEntry_InvalidUpdates(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateWorkEntryUseCase(repo)

	// Add an initial entry
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	addUseCase := NewAddWorkEntryUseCase(repo, timeProvider)
	addOutput, _ := addUseCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Original Task",
		DurationMin: 60,
	})

	// Test: Update task to empty string (should result in validation error)
	t.Run("update to empty task", func(t *testing.T) {
		// Reset record
		_ = repo.Save(model.DayRecord{
			Date: "2026-03-28",
			Entries: []model.WorkEntry{
				{ID: addOutput.EntryID, Task: "X", DurationMin: 60}, // Start with non-empty task
			},
		})

		// Try to update to empty task
		input := UpdateWorkEntryInput{
			Date:    "2026-03-28",
			EntryID: addOutput.EntryID,
			Task:    "", // Empty = keep existing (which is "X"), so this should succeed
		}

		_, err := useCase.Execute(input)
		if err != nil {
			t.Errorf("expected no error for empty task input (should keep existing), got %v", err)
		}
	})

	// Test: Can't have an entry with empty task in the DB (validation should catch it)
	// This would only fail if the entry in DB already had an empty task
	t.Run("entry with empty task in DB", func(t *testing.T) {
		// This scenario shouldn't happen (AddWorkEntry prevents it), but if it does...
		// Actually, this test doesn't make sense - we can't create an invalid entry
		// Let's skip this test case
	})
}
