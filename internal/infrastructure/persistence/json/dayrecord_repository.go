package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// FileSystemDayRecordRepository implements repository.DayRecordRepository
// using JSON files on the filesystem.
//
// Design Decision: This is a clean implementation of the Repository pattern.
// All filesystem logic is isolated here - domain/application never touch os/filepath.
type FileSystemDayRecordRepository struct {
	storage *StorageManager
}

// NewFileSystemDayRecordRepository creates a new FileSystemDayRecordRepository.
func NewFileSystemDayRecordRepository(
	storage *StorageManager,
) repository.DayRecordRepository {
	return &FileSystemDayRecordRepository{
		storage: storage,
	}
}

// FindByDate loads a DayRecord for the given date.
func (r *FileSystemDayRecordRepository) FindByDate(date string) (model.DayRecord, error) {
	path, err := r.storage.PathForDate(date)
	if err != nil {
		return model.DayRecord{}, fmt.Errorf("failed to get path for date %s: %w", date, err)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Return empty record with date filled in (matches existing behavior)
		return model.DayRecord{
			Date:    date,
			Entries: []model.WorkEntry{},
		}, nil
	}
	if err != nil {
		return model.DayRecord{}, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var rec persistedDayRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return model.DayRecord{}, fmt.Errorf("failed to unmarshal day record: %w", err)
	}

	return toDomainDayRecord(rec), nil
}

// FindAll loads all DayRecords, sorted newest first.
func (r *FileSystemDayRecordRepository) FindAll() ([]model.DayRecord, error) {
	dir, err := r.storage.Dir()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage directory: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "????-??-??.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob day records: %w", err)
	}

	var records []model.DayRecord
	for _, f := range files {
		rec, err := r.loadFromPath(f)
		if err != nil {
			// Skip unreadable files to keep listing robust.
			continue
		}
		records = append(records, rec)
	}

	// Sort newest first
	sort.Slice(records, func(i, j int) bool {
		return records[i].Date > records[j].Date
	})

	return records, nil
}

// Save persists a DayRecord.
func (r *FileSystemDayRecordRepository) Save(record model.DayRecord) error {
	path, err := r.storage.PathForDate(record.Date)
	if err != nil {
		return fmt.Errorf("failed to get path for date %s: %w", record.Date, err)
	}

	data, err := json.MarshalIndent(toPersistedDayRecord(record), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal day record: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// Delete removes a DayRecord.
func (r *FileSystemDayRecordRepository) Delete(date string) error {
	path, err := r.storage.PathForDate(date)
	if err != nil {
		return fmt.Errorf("failed to get path for date %s: %w", date, err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

// Exists checks if a DayRecord exists.
func (r *FileSystemDayRecordRepository) Exists(date string) (bool, error) {
	path, err := r.storage.PathForDate(date)
	if err != nil {
		return false, fmt.Errorf("failed to get path for date %s: %w", date, err)
	}

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// loadFromPath is a helper to load a DayRecord from a specific file path.
func (r *FileSystemDayRecordRepository) loadFromPath(path string) (model.DayRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.DayRecord{}, err
	}

	var rec persistedDayRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return model.DayRecord{}, err
	}

	return toDomainDayRecord(rec), nil
}

type persistedWorkEntry struct {
	ID          string `json:"id"`
	Project     string `json:"project"`
	Task        string `json:"task"`
	DurationMin int    `json:"duration_min"`
	IsBreak     bool   `json:"is_break"`
}

type persistedDayRecord struct {
	Date      string               `json:"date"`
	StartTime string               `json:"start_time,omitempty"`
	EndTime   string               `json:"end_time,omitempty"`
	Entries   []persistedWorkEntry `json:"entries"`
	Notes     string               `json:"notes,omitempty"`
}

func toDomainDayRecord(rec persistedDayRecord) model.DayRecord {
	entries := make([]model.WorkEntry, len(rec.Entries))
	for i, entry := range rec.Entries {
		entries[i] = model.WorkEntry{
			ID:          entry.ID,
			Project:     entry.Project,
			Task:        entry.Task,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		}
	}

	return model.DayRecord{
		Date:      rec.Date,
		StartTime: rec.StartTime,
		EndTime:   rec.EndTime,
		Entries:   entries,
		Notes:     rec.Notes,
	}
}

func toPersistedDayRecord(rec model.DayRecord) persistedDayRecord {
	entries := make([]persistedWorkEntry, len(rec.Entries))
	for i, entry := range rec.Entries {
		entries[i] = persistedWorkEntry{
			ID:          entry.ID,
			Project:     entry.Project,
			Task:        entry.Task,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		}
	}

	return persistedDayRecord{
		Date:      rec.Date,
		StartTime: rec.StartTime,
		EndTime:   rec.EndTime,
		Entries:   entries,
		Notes:     rec.Notes,
	}
}
