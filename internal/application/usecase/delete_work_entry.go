package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// DeleteWorkEntryInput contains the data needed to delete a work entry.
type DeleteWorkEntryInput struct {
	Date    string // YYYY-MM-DD format
	EntryID string // ID of the entry to delete
}

// DeleteWorkEntryOutput contains the result of deleting a work entry.
type DeleteWorkEntryOutput struct {
	RecordDate     string
	RemainingCount int // number of entries remaining
	TotalWork      int
	TotalBreaks    int
}

// DeleteWorkEntryUseCase handles deleting a work entry.
// This use case orchestrates:
// 1. Loading the day record
// 2. Finding and removing the entry by ID
// 3. Saving the updated record (or deleting if empty)
// 4. Returning summary statistics
type DeleteWorkEntryUseCase struct {
	dayRepo repository.DayRecordRepository
}

// NewDeleteWorkEntryUseCase creates a new DeleteWorkEntryUseCase.
func NewDeleteWorkEntryUseCase(dayRepo repository.DayRecordRepository) *DeleteWorkEntryUseCase {
	return &DeleteWorkEntryUseCase{
		dayRepo: dayRepo,
	}
}

// Execute deletes a work entry from the specified day.
func (uc *DeleteWorkEntryUseCase) Execute(input DeleteWorkEntryInput) (*DeleteWorkEntryOutput, error) {
	// Validate input
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	if input.EntryID == "" {
		return nil, fmt.Errorf("entry ID is required")
	}

	// Load day record
	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load day record: %w", err)
	}

	// Find and remove entry by ID
	entryIndex := -1
	for i, entry := range record.Entries {
		if entry.ID == input.EntryID {
			entryIndex = i
			break
		}
	}

	if entryIndex == -1 {
		return nil, fmt.Errorf("entry with ID %s not found", input.EntryID)
	}

	// Remove the entry (preserve order)
	record.Entries = append(record.Entries[:entryIndex], record.Entries[entryIndex+1:]...)

	// If record is now completely empty, delete it
	if len(record.Entries) == 0 && record.Notes == "" && record.StartTime == "" && record.EndTime == "" {
		if err := uc.dayRepo.Delete(input.Date); err != nil {
			return nil, fmt.Errorf("failed to delete empty day record: %w", err)
		}

		return &DeleteWorkEntryOutput{
			RecordDate:     input.Date,
			RemainingCount: 0,
			TotalWork:      0,
			TotalBreaks:    0,
		}, nil
	}

	// Save updated record
	if err := uc.dayRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save day record: %w", err)
	}

	// Calculate totals for output
	workMin, breakMin, _ := record.WorkTotals()

	return &DeleteWorkEntryOutput{
		RecordDate:     record.Date,
		RemainingCount: len(record.Entries),
		TotalWork:      int(workMin.Minutes()),
		TotalBreaks:    int(breakMin.Minutes()),
	}, nil
}
