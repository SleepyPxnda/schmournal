package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// UpdateNotesInput contains the data needed to update day notes.
type UpdateNotesInput struct {
	Date  string // YYYY-MM-DD format
	Notes string // Freeform markdown text (use "-" to clear)
}

// UpdateNotesOutput contains the result of updating notes.
type UpdateNotesOutput struct {
	RecordDate  string
	NotesLength int
}

// UpdateNotesUseCase handles updating the notes for a day.
// This use case orchestrates:
// 1. Loading the day record (or creating empty if needed)
// 2. Updating the notes field
// 3. Saving the record (or deleting if completely empty)
// 4. Returning confirmation
type UpdateNotesUseCase struct {
	dayRepo repository.DayRecordRepository
}

// NewUpdateNotesUseCase creates a new UpdateNotesUseCase.
func NewUpdateNotesUseCase(dayRepo repository.DayRecordRepository) *UpdateNotesUseCase {
	return &UpdateNotesUseCase{
		dayRepo: dayRepo,
	}
}

// Execute updates the notes for the specified day.
func (uc *UpdateNotesUseCase) Execute(input UpdateNotesInput) (*UpdateNotesOutput, error) {
	// Validate input
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}

	// Load or create day record
	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load day record: %w", err)
	}

	// Update notes
	if input.Notes == "-" {
		record.Notes = ""
	} else {
		record.Notes = input.Notes
	}

	// If record is now completely empty, delete it
	if len(record.Entries) == 0 && record.Notes == "" && record.StartTime == "" && record.EndTime == "" {
		if err := uc.dayRepo.Delete(input.Date); err != nil {
			return nil, fmt.Errorf("failed to delete empty day record: %w", err)
		}

		return &UpdateNotesOutput{
			RecordDate:  input.Date,
			NotesLength: 0,
		}, nil
	}

	// Save updated record
	if err := uc.dayRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save day record: %w", err)
	}

	return &UpdateNotesOutput{
		RecordDate:  record.Date,
		NotesLength: len(record.Notes),
	}, nil
}
