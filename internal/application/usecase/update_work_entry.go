package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// UpdateWorkEntryInput contains the data needed to update a work entry.
type UpdateWorkEntryInput struct {
	Date        string // YYYY-MM-DD format
	EntryID     string // ID of the entry to update
	Task        string // optional - if empty, keeps existing
	Project     string // optional - if empty, keeps existing (use "-" to clear)
	DurationMin int    // optional - if 0, keeps existing
}

// UpdateWorkEntryOutput contains the result of updating a work entry.
type UpdateWorkEntryOutput struct {
	EntryID     string
	RecordDate  string
	TotalWork   int
	TotalBreaks int
}

// UpdateWorkEntryUseCase handles updating an existing work entry.
// This use case orchestrates:
// 1. Loading the day record
// 2. Finding the entry by ID
// 3. Updating the entry fields (only non-empty values)
// 4. Validating the updated entry
// 5. Saving the record
// 6. Returning summary statistics
type UpdateWorkEntryUseCase struct {
	dayRepo repository.DayRecordRepository
}

// NewUpdateWorkEntryUseCase creates a new UpdateWorkEntryUseCase.
func NewUpdateWorkEntryUseCase(dayRepo repository.DayRecordRepository) *UpdateWorkEntryUseCase {
	return &UpdateWorkEntryUseCase{
		dayRepo: dayRepo,
	}
}

// Execute updates a work entry in the specified day.
func (uc *UpdateWorkEntryUseCase) Execute(input UpdateWorkEntryInput) (*UpdateWorkEntryOutput, error) {
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

	// Find entry by ID
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

	// Update entry fields (only non-empty/non-zero values)
	entry := &record.Entries[entryIndex]

	if input.Task != "" {
		entry.Task = input.Task
	}

	// Handle project update
	// Empty string means "keep existing"
	// "-" means "clear project"
	if input.Project == "-" {
		entry.Project = ""
	} else if input.Project != "" {
		entry.Project = input.Project
	}

	if input.DurationMin > 0 {
		entry.DurationMin = input.DurationMin
	}

	// Validate updated entry
	if entry.Task == "" {
		return nil, fmt.Errorf("task cannot be empty")
	}
	if entry.DurationMin <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}

	// Save updated record
	if err := uc.dayRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save day record: %w", err)
	}

	// Calculate totals for output
	workMin, breakMin, _ := record.WorkTotals()

	return &UpdateWorkEntryOutput{
		EntryID:     input.EntryID,
		RecordDate:  record.Date,
		TotalWork:   int(workMin.Minutes()),
		TotalBreaks: int(breakMin.Minutes()),
	}, nil
}
