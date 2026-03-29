package usecase

import (
	"strings"
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestSetDayTimes_SetBothTimes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	input := SetDayTimesInput{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StartTime != "09:00" {
		t.Errorf("expected start time '09:00', got %s", output.StartTime)
	}
	if output.EndTime != "17:00" {
		t.Errorf("expected end time '17:00', got %s", output.EndTime)
	}
	if !output.HasDuration {
		t.Error("expected HasDuration to be true")
	}
	if output.DayDurationMin != 480 { // 8 hours = 480 minutes
		t.Errorf("expected day duration 480 minutes, got %d", output.DayDurationMin)
	}

	// Verify saved
	saved, _ := repo.FindByDate("2026-03-28")
	if saved.StartTime != "09:00" || saved.EndTime != "17:00" {
		t.Error("expected times to be saved")
	}
}

func TestSetDayTimes_SetStartTimeOnly(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	// First set end time
	record := model.DayRecord{
		Date:    "2026-03-28",
		EndTime: "17:00",
	}
	_ = repo.Save(record)

	// Now set start time
	input := SetDayTimesInput{
		Date:      "2026-03-28",
		StartTime: "09:00",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StartTime != "09:00" {
		t.Errorf("expected start time '09:00', got %s", output.StartTime)
	}
	if output.EndTime != "17:00" {
		t.Error("expected end time to be preserved")
	}
	if !output.HasDuration {
		t.Error("expected HasDuration to be true")
	}
}

func TestSetDayTimes_SetEndTimeOnly(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	// Set existing start time
	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
	}
	_ = repo.Save(record)

	// Set end time
	input := SetDayTimesInput{
		Date:    "2026-03-28",
		EndTime: "17:00",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StartTime != "09:00" {
		t.Error("expected start time to be preserved")
	}
	if output.EndTime != "17:00" {
		t.Errorf("expected end time '17:00', got %s", output.EndTime)
	}
}

func TestSetDayTimes_ClearStartTime(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	// Set both times first
	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
	}
	_ = repo.Save(record)

	// Clear start time using "-"
	input := SetDayTimesInput{
		Date:      "2026-03-28",
		StartTime: "-",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StartTime != "" {
		t.Errorf("expected start time to be cleared, got %s", output.StartTime)
	}
	if output.EndTime != "17:00" {
		t.Error("expected end time to be preserved")
	}
	if output.HasDuration {
		t.Error("expected HasDuration to be false after clearing start time")
	}
}

func TestSetDayTimes_ClearEndTime(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	// Set both times first
	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
	}
	_ = repo.Save(record)

	// Clear end time using "-"
	input := SetDayTimesInput{
		Date:    "2026-03-28",
		EndTime: "-",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StartTime != "09:00" {
		t.Error("expected start time to be preserved")
	}
	if output.EndTime != "" {
		t.Errorf("expected end time to be cleared, got %s", output.EndTime)
	}
	if output.HasDuration {
		t.Error("expected HasDuration to be false after clearing end time")
	}
}

func TestSetDayTimes_InvalidTimeFormat(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	tests := []struct {
		name      string
		startTime string
		endTime   string
	}{
		{
			name:      "invalid start time - no colon",
			startTime: "0900",
			endTime:   "",
		},
		{
			name:      "invalid start time - wrong format",
			startTime: "9:00",
			endTime:   "",
		},
		{
			name:      "invalid start time - out of range hour",
			startTime: "25:00",
			endTime:   "",
		},
		{
			name:      "invalid start time - out of range minute",
			startTime: "09:60",
			endTime:   "",
		},
		{
			name:      "invalid end time",
			startTime: "",
			endTime:   "17:99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := SetDayTimesInput{
				Date:      "2026-03-28",
				StartTime: tt.startTime,
				EndTime:   tt.endTime,
			}

			_, err := useCase.Execute(input)
			if err == nil {
				t.Error("expected validation error for invalid time format")
			}
			if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "format") {
				t.Errorf("expected format error, got %v", err)
			}
		})
	}
}

func TestSetDayTimes_ValidationErrors(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	tests := []struct {
		name  string
		input SetDayTimesInput
	}{
		{
			name: "missing date",
			input: SetDayTimesInput{
				Date:      "",
				StartTime: "09:00",
			},
		},
		{
			name: "no times provided",
			input: SetDayTimesInput{
				Date:      "2026-03-28",
				StartTime: "",
				EndTime:   "",
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

func TestSetDayTimes_UpdateExistingTimes(t *testing.T) {
	repo := NewMockDayRecordRepository()
	useCase := NewSetDayTimesUseCase(repo)

	// Set initial times
	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "08:00",
		EndTime:   "16:00",
	}
	_ = repo.Save(record)

	// Update both times
	input := SetDayTimesInput{
		Date:      "2026-03-28",
		StartTime: "09:30",
		EndTime:   "17:45",
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.StartTime != "09:30" {
		t.Errorf("expected updated start time '09:30', got %s", output.StartTime)
	}
	if output.EndTime != "17:45" {
		t.Errorf("expected updated end time '17:45', got %s", output.EndTime)
	}

	// Verify calculation (09:30 to 17:45 = 8 hours 15 minutes = 495 minutes)
	if output.DayDurationMin != 495 {
		t.Errorf("expected day duration 495 minutes, got %d", output.DayDurationMin)
	}
}
