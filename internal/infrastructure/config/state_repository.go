package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// stateFileName is the name of the file used to persist runtime application
// state between sessions.
const stateFileName = "schmournal.state"

// FileSystemStateRepository implements repository.StateRepository using
// the file system for persistence.
type FileSystemStateRepository struct {
	configDir string
}

// NewFileSystemStateRepository creates a new state repository.
// configDir is the directory where the state file will be stored.
func NewFileSystemStateRepository(configDir string) repository.StateRepository {
	return &FileSystemStateRepository{
		configDir: configDir,
	}
}

// statePath returns the path to the application state file.
func (r *FileSystemStateRepository) statePath() string {
	return filepath.Join(r.configDir, stateFileName)
}

// LoadState reads the persisted application state.
// Returns an empty state if the file does not exist.
func (r *FileSystemStateRepository) LoadState() (model.AppState, error) {
	path := r.statePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return model.AppState{}, nil
	}
	if err != nil {
		return model.AppState{}, err
	}

	var state model.AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return model.AppState{}, err
	}
	return state, nil
}

// SaveState persists the given application state to disk.
func (r *FileSystemStateRepository) SaveState(state model.AppState) error {
	path := r.statePath()

	// Ensure config directory exists
	if err := os.MkdirAll(r.configDir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
