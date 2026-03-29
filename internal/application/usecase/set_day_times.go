package usecase

import (
	"fmt"
	"regexp"

	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// SetDayTimesInput contains the data needed to set day start/end times.
type SetDayTimesInput struct {
	Date      string // YYYY-MM-DD format
	StartTime string // HH:MM format (optional, use "-" to clear)
	EndTime   string // HH:MM format (optional, use "-" to clear)
}

// SetDayTimesOutput contains the result of setting day times.
type SetDayTimesOutput struct {
	RecordDate    string
	StartTime     string
	EndTime       string
	DayDurationMin int  // 0 if times are incomplete
	HasDuration   bool // true if both times are set
}

// SetDayTimesUseCase handles setting start/end times for a day.
// This use case orchestrates:
// 1. Loading the day record (or creating empty if needed)
// 2. Validating time format (HH:MM)
// 3. Updating start/end times
// 4. Saving the record
// 5. Returning the updated times and calculated duration
type SetDayTimesUseCase struct {
	dayRepo repository.DayRecordRepository
}

// NewSetDayTimesUseCase creates a new SetDayTimesUseCase.
func NewSetDayTimesUseCase(dayRepo repository.DayRecordRepository) *SetDayTimesUseCase {
	return &SetDayTimesUseCase{
		dayRepo: dayRepo,
	}
}

// Execute sets the start/end times for the specified day.
func (uc *SetDayTimesUseCase) Execute(input SetDayTimesInput) (*SetDayTimesOutput, error) {
	// Validate input
	if input.Date == "" {
		return nil, fmt.Errorf("date is required")
	}

	// At least one time must be provided
	if input.StartTime == "" && input.EndTime == "" {
		return nil, fmt.Errorf("at least one time (start or end) must be provided")
	}

	// Load or create day record
	record, err := uc.dayRepo.FindByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to load day record: %w", err)
	}

	// Update start time
	if input.StartTime != "" {
		if input.StartTime == "-" {
			record.StartTime = ""
		} else {
			if err := validateTimeFormat(input.StartTime); err != nil {
				return nil, fmt.Errorf("invalid start time: %w", err)
			}
			record.StartTime = input.StartTime
		}
	}

	// Update end time
	if input.EndTime != "" {
		if input.EndTime == "-" {
			record.EndTime = ""
		} else {
			if err := validateTimeFormat(input.EndTime); err != nil {
				return nil, fmt.Errorf("invalid end time: %w", err)
			}
			record.EndTime = input.EndTime
		}
	}

	// Save updated record
	if err := uc.dayRepo.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save day record: %w", err)
	}

	// Calculate duration
	duration, hasDuration := record.DayDuration()

	return &SetDayTimesOutput{
		RecordDate:     record.Date,
		StartTime:      record.StartTime,
		EndTime:        record.EndTime,
		DayDurationMin: int(duration.Minutes()),
		HasDuration:    hasDuration,
	}, nil
}

// validateTimeFormat validates that a time string is in HH:MM format.
var timeFormatRegex = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

func validateTimeFormat(timeStr string) error {
	if !timeFormatRegex.MatchString(timeStr) {
		return fmt.Errorf("time must be in HH:MM format (e.g., 09:00 or 17:30), got %s", timeStr)
	}
	return nil
}
