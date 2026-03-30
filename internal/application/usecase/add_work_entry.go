package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

// AddWorkEntryInput contains the data needed to add a work entry.
type AddWorkEntryInput struct {
	Date        string // YYYY-MM-DD format
	Task        string
	Project     string // optional
	DurationMin int
	IsBreak     bool
}

// AddWorkEntryOutput contains the result of adding a work entry.
type AddWorkEntryOutput struct {
	EntryID     string
	RecordDate  string
	TotalWork   int // total work minutes in the day
	TotalBreaks int // total break minutes in the day
}

// AddWorkEntryUseCase handles adding a work entry to a day record.
// This use case orchestrates:
// 1. Loading the day record (or creating a new one)
// 2. Validating the input
// 3. Creating a new work entry with a unique ID
// 4. Adding it to the record
// 5. Saving the updated record
// 6. Returning summary statistics
type AddWorkEntryUseCase struct {
	dayRepo      repository.DayRecordRepository
	timeProvider service.TimeProvider
}

// NewAddWorkEntryUseCase creates a new AddWorkEntryUseCase.
func NewAddWorkEntryUseCase(
	dayRepo repository.DayRecordRepository,
	timeProvider service.TimeProvider,
) *AddWorkEntryUseCase {
	return &AddWorkEntryUseCase{
		dayRepo:      dayRepo,
		timeProvider: timeProvider,
	}
}

// Execute adds a work entry to the specified day.
func (uc *AddWorkEntryUseCase) Execute(input AddWorkEntryInput) (*AddWorkEntryOutput, error) {
	// Validate input
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	if input.Task == "" {
		return nil, fmt.Errorf("task is required")
	}
	if input.DurationMin <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}

	// Load or create day record
	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load day record: %w", err)
	}

	// Create new work entry with unique ID
	entryID := uc.timeProvider.GenerateID()
	newEntry := model.WorkEntry{
		ID:          entryID,
		Task:        input.Task,
		Project:     input.Project,
		DurationMin: input.DurationMin,
		IsBreak:     input.IsBreak,
	}

	// Validate the entry itself (basic validation)
	if newEntry.Task == "" {
		return nil, fmt.Errorf("task cannot be empty")
	}
	if newEntry.DurationMin <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}

	// Add to record
	record.Entries = append(record.Entries, newEntry)

	// Save updated record
	if err := uc.dayRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save day record: %w", err)
	}

	// Calculate totals for output
	workMin, breakMin, _ := record.WorkTotals()

	return &AddWorkEntryOutput{
		EntryID:     entryID,
		RecordDate:  record.Date,
		TotalWork:   int(workMin.Minutes()),
		TotalBreaks: int(breakMin.Minutes()),
	}, nil
}
