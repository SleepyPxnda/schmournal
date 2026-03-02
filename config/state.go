package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// stateFileName is the name of the file used to persist runtime application
// state between sessions.
const stateFileName = "schmournal.state"

// AppState holds lightweight runtime state that is persisted across sessions.
type AppState struct {
	ActiveWorkspace string `json:"active_workspace"`
}

// statePath returns the path to the application state file.
func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", stateFileName), nil
}

// LoadState reads the persisted AppState. If the file does not exist an empty
// state is returned without error.
func LoadState() (AppState, error) {
	path, err := statePath()
	if err != nil {
		return AppState{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return AppState{}, nil
	}
	if err != nil {
		return AppState{}, err
	}
	var s AppState
	if err := json.Unmarshal(data, &s); err != nil {
		return AppState{}, err
	}
	return s, nil
}

// SaveState persists the given AppState to disk.
func SaveState(s AppState) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
