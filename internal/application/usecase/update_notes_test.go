package usecase

import (
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestUpdateNotes_SetNotes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: "Today was productive!\n\n- Completed feature X\n- Fixed bug Y",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.NotesLength == 0 {
		t.Error("expected notes length > 0")
	}

	// Verify saved
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Notes != input.Notes {
		t.Errorf("expected notes to match, got %s", saved.Notes)
	}
}

func TestUpdateNotes_UpdateExistingNotes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	// Create record with existing notes
	record := model.DayRecord{
		Date:  "2026-03-28",
		Notes: "Old notes",
	}
	_ = repo.Save(record)

	// Update notes
	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: "New notes - completely replaced",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify replaced
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Notes != "New notes - completely replaced" {
		t.Error("expected notes to be replaced")
	}

	if output.NotesLength != len("New notes - completely replaced") {
		t.Errorf("expected notes length %d, got %d", len("New notes - completely replaced"), output.NotesLength)
	}
}

func TestUpdateNotes_ClearNotes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	// Create record with notes
	record := model.DayRecord{
		Date:  "2026-03-28",
		Notes: "Notes to be cleared",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Task", DurationMin: 60},
		},
	}
	_ = repo.Save(record)

	// Clear notes using "-"
	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: "-",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.NotesLength != 0 {
		t.Error("expected notes length to be 0")
	}

	// Verify cleared
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Notes != "" {
		t.Error("expected notes to be cleared")
	}
	// Entry should still be there
	if len(saved.Entries) != 1 {
		t.Error("expected entry to remain")
	}
}

func TestUpdateNotes_ClearNotes_DeletesEmptyRecord(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	// Create record with only notes (no entries, no times)
	record := model.DayRecord{
		Date:  "2026-03-28",
		Notes: "Only notes, nothing else",
	}
	_ = repo.Save(record)

	// Clear notes using "-"
	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: "-",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.NotesLength != 0 {
		t.Error("expected notes length to be 0")
	}

	// Verify record was deleted (should return empty)
	saved, _ := repo.FindByDate("2026-03-28")
	if len(saved.Entries) != 0 || saved.Notes != "" {
		t.Error("expected record to be deleted")
	}
}

func TestUpdateNotes_EmptyStringNotes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	// Set notes to empty string (not "-")
	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: "",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.NotesLength != 0 {
		t.Error("expected notes length to be 0")
	}

	// Empty string should also clear notes
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Notes != "" {
		t.Error("expected empty notes")
	}
}

func TestUpdateNotes_PreservesEntries(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	// Create record with entries
	record := model.DayRecord{
		Date: "2026-03-28",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Task 1", DurationMin: 60},
			{ID: "2", Task: "Task 2", DurationMin: 30},
		},
	}
	_ = repo.Save(record)

	// Add notes
	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: "Added notes after entries",
	}

	_, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify entries are preserved
	saved, _ := repo.FindByDate("2026-03-28")
	if len(saved.Entries) != 2 {
		t.Error("expected entries to be preserved")
	}
	if saved.Notes != "Added notes after entries" {
		t.Error("expected notes to be added")
	}
}

func TestUpdateNotes_ValidationErrors(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	// Missing date
	input := UpdateNotesInput{
		Date:  "",
		Notes: "Some notes",
	}

	_, err := useCase.Execute(input)
	if err == nil {
		t.Error("expected error for missing date")
	}
}

func TestUpdateNotes_MultilineNotes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewUpdateNotesUseCase(repo)

	multilineNotes := `# Daily Summary

## Accomplishments
- Feature X implemented
- Bug Y fixed
- Code review completed

## Challenges
- Integration test failures
- Had to refactor module Z

## Tomorrow
- Deploy to staging
- Continue with feature W`

	input := UpdateNotesInput{
		Date:  "2026-03-28",
		Notes: multilineNotes,
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.NotesLength != len(multilineNotes) {
		t.Errorf("expected notes length %d, got %d", len(multilineNotes), output.NotesLength)
	}

	// Verify multiline content preserved
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.Notes != multilineNotes {
		t.Error("expected multiline notes to be preserved exactly")
	}
}
