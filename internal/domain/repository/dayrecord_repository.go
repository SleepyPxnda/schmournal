package repository

import (
	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// DayRecordRepository defines the interface for persisting DayRecords.
// This is a domain interface - implementations live in infrastructure layer.
type DayRecordRepository interface {
	// FindByDate loads a DayRecord for the given date (YYYY-MM-DD).
	// Returns an empty DayRecord (not an error) if the date doesn't exist.
	FindByDate(date string) (model.DayRecord, error)

	// FindAll loads all DayRecords, sorted newest first.
	FindAll() ([]model.DayRecord, error)

	// Save persists a DayRecord. Creates new or updates existing.
	Save(record model.DayRecord) error

	// Delete removes a DayRecord for the given date.
	Delete(date string) error

	// Exists checks if a DayRecord exists for the given date.
	Exists(date string) (bool, error)
}
