package json

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StorageManager manages filesystem paths for journal data.
// Replaces the global storagePath variable with an injectable struct.
//
// Design Decision: This encapsulates all path logic in one place,
// making it easy to switch storage backends later (e.g., SQLite, Cloud).
type StorageManager struct {
	basePath string
}

// NewStorageManager creates a new StorageManager.
// basePath supports ~ expansion (e.g. "~/.journal").
func NewStorageManager(basePath string) (*StorageManager, error) {
	if basePath == "" {
		// Default to ~/.journal
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(home, ".journal")
	}

	expanded, err := expandPath(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path: %w", err)
	}

	return &StorageManager{basePath: expanded}, nil
}

// Dir returns (and creates if necessary) the journal storage directory.
func (s *StorageManager) Dir() (string, error) {
	return s.basePath, os.MkdirAll(s.basePath, 0o755)
}

// PathForDate returns the file path for a given date (YYYY-MM-DD).
func (s *StorageManager) PathForDate(date string) (string, error) {
	dir, err := s.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, date+".json"), nil
}

// TodosPath returns the file path for workspace-level TODOs.
func (s *StorageManager) TodosPath() (string, error) {
	dir, err := s.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "todos.json"), nil
}

// TodosPathForWorkspace returns the file path for workspace-specific TODOs.
// NOTE: This is a temporary implementation. In the current codebase, workspaces
// are stored in the user's home directory under different folders.
// For now, we use the same folder structure as the existing implementation.
func (s *StorageManager) TodosPathForWorkspace(workspace string) (string, error) {
	// For "default" workspace, use the regular todos.json
	if workspace == "default" || workspace == "" {
		return s.TodosPath()
	}

	// For other workspaces, use workspace-specific path
	// This matches the existing behavior where workspaces are in ~/.journal-{workspace}/
	dir, err := s.Dir()
	if err != nil {
		return "", err
	}

	// Replace the directory with workspace-specific directory
	// e.g., ~/.journal/ → ~/.journal-{workspace}/
	baseDir := filepath.Dir(dir)
	workspaceDir := filepath.Join(baseDir, ".journal-"+workspace)

	// Ensure the workspace directory exists
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	return filepath.Join(workspaceDir, "todos.json"), nil
}

// ExportsDir returns the directory for markdown exports.
func (s *StorageManager) ExportsDir() (string, error) {
	dir, err := s.Dir()
	if err != nil {
		return "", err
	}
	exportsDir := filepath.Join(dir, "exports")
	return exportsDir, os.MkdirAll(exportsDir, 0o755)
}

// expandPath expands a leading ~ to the user's home directory.
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// TrimPrefix strips a leading "/" so that filepath.Join does not
	// interpret the remainder as an absolute path (e.g. "~/.journal" →
	// "<home>/.journal" rather than "/.journal").
	rest := strings.TrimPrefix(path[1:], "/")
	rest = strings.TrimPrefix(rest, "\\") // Windows support
	if rest == "" {
		return home, nil
	}
	return filepath.Join(home, rest), nil
}
