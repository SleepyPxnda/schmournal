package repository

import "github.com/sleepypxnda/schmournal/internal/domain/model"

// StateRepository manages application state persistence.
type StateRepository interface {
	// LoadState reads the persisted application state.
	// Returns an empty state if the file does not exist.
	LoadState() (model.AppState, error)

	// SaveState persists the given application state to disk.
	SaveState(state model.AppState) error
}
