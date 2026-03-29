package usecase

import (
	"testing"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// MockDayRecordRepository is a simple in-memory repository for testing.
type MockDayRecordRepository struct {
	records map[string]model.DayRecord
}

func NewMockDayRecordRepository() *MockDayRecordRepository {
	return &MockDayRecordRepository{
		records: make(map[string]model.DayRecord),
	}
}

func (m *MockDayRecordRepository) FindByDate(date string) (model.DayRecord, error) {
	if rec, ok := m.records[date]; ok {
		return rec, nil
	}
	// Return empty record if not found
	return model.DayRecord{
		Date:    date,
		Entries: []model.WorkEntry{},
	}, nil
}

func (m *MockDayRecordRepository) FindAll() ([]model.DayRecord, error) {
	var result []model.DayRecord
	for _, rec := range m.records {
		result = append(result, rec)
	}
	return result, nil
}

func (m *MockDayRecordRepository) Save(record model.DayRecord) error {
	m.records[record.Date] = record
	return nil
}

func (m *MockDayRecordRepository) Delete(date string) error {
	delete(m.records, date)
	return nil
}

func (m *MockDayRecordRepository) Exists(date string) (bool, error) {
	_, ok := m.records[date]
	return ok, nil
}

var _ repository.DayRecordRepository = (*MockDayRecordRepository)(nil)

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestAddWorkEntry_Success(t *testing.T) {
	repo := NewMockDayRecordRepository()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	useCase := NewAddWorkEntryUseCase(repo, timeProvider)

	input := AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Code Review",
		Project:     "Backend",
		DurationMin: 60,
		IsBreak:     false,
	}

	output, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.EntryID == "" {
		t.Error("expected entry ID to be generated")
	}
	if output.RecordDate != "2026-03-28" {
		t.Errorf("expected record date '2026-03-28', got %s", output.RecordDate)
	}
	if output.TotalWork != 60 {
		t.Errorf("expected total work 60, got %d", output.TotalWork)
	}
	if output.TotalBreaks != 0 {
		t.Errorf("expected total breaks 0, got %d", output.TotalBreaks)
	}

	// Verify record was saved
	saved, err := repo.FindByDate("2026-03-28")
	if err != nil {
		t.Fatalf("failed to load saved record: %v", err)
	}
	if len(saved.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(saved.Entries))
	}
	if saved.Entries[0].Task != "Code Review" {
		t.Errorf("expected task 'Code Review', got %s", saved.Entries[0].Task)
	}
}

func TestAddWorkEntry_MultipleEntries(t *testing.T) {
	repo := NewMockDayRecordRepository()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	useCase := NewAddWorkEntryUseCase(repo, timeProvider)

	// Add first entry
	_, err := useCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Work",
		DurationMin: 60,
		IsBreak:     false,
	})
	if err != nil {
		t.Fatalf("first entry failed: %v", err)
	}

	// Add second entry (break)
	output, err := useCase.Execute(AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Lunch",
		DurationMin: 30,
		IsBreak:     true,
	})
	if err != nil {
		t.Fatalf("second entry failed: %v", err)
	}

	if output.TotalWork != 60 {
		t.Errorf("expected total work 60, got %d", output.TotalWork)
	}
	if output.TotalBreaks != 30 {
		t.Errorf("expected total breaks 30, got %d", output.TotalBreaks)
	}

	// Verify both entries saved
	saved, err := repo.FindByDate("2026-03-28")
	if err != nil {
		t.Fatalf("failed to load saved record: %v", err)
	}
	if len(saved.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(saved.Entries))
	}
}

func TestAddWorkEntry_ValidationErrors(t *testing.T) {
	repo := NewMockDayRecordRepository()
	timeProvider := newTestTimeProviderAt(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	useCase := NewAddWorkEntryUseCase(repo, timeProvider)

	tests := []struct {
		name  string
		input AddWorkEntryInput
	}{
		{
			name: "missing date",
			input: AddWorkEntryInput{
				Date:        "",
				Task:        "Task",
				DurationMin: 60,
			},
		},
		{
			name: "missing task",
			input: AddWorkEntryInput{
				Date:        "2026-03-28",
				Task:        "",
				DurationMin: 60,
			},
		},
		{
			name: "zero duration",
			input: AddWorkEntryInput{
				Date:        "2026-03-28",
				Task:        "Task",
				DurationMin: 0,
			},
		},
		{
			name: "negative duration",
			input: AddWorkEntryInput{
				Date:        "2026-03-28",
				Task:        "Task",
				DurationMin: -10,
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

func TestAddWorkEntry_UniqueIDs(t *testing.T) {
	repo := NewMockDayRecordRepository()
	// Use monotonic test provider to ensure unique IDs.
	timeProvider := newTestMonotonicTimeProvider()
	useCase := NewAddWorkEntryUseCase(repo, timeProvider)

	input := AddWorkEntryInput{
		Date:        "2026-03-28",
		Task:        "Task",
		DurationMin: 30,
		IsBreak:     false,
	}

	// Add two entries quickly
	output1, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("first entry failed: %v", err)
	}

	// Add tiny delay to mimic rapid consecutive calls.
	time.Sleep(1 * time.Millisecond)

	output2, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("second entry failed: %v", err)
	}

	// IDs should be different
	if output1.EntryID == output2.EntryID {
		t.Error("expected unique entry IDs, got duplicates")
	}
}
