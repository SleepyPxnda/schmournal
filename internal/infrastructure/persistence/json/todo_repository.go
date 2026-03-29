package json

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// FileSystemTodoRepository implements repository.TodoRepository
// using JSON files on the filesystem.
type FileSystemTodoRepository struct {
	storage *StorageManager
}

// NewFileSystemTodoRepository creates a new FileSystemTodoRepository.
func NewFileSystemTodoRepository(storage *StorageManager) repository.TodoRepository {
	return &FileSystemTodoRepository{
		storage: storage,
	}
}

// Load reads workspace TODOs (active + archived) for the given workspace.
func (r *FileSystemTodoRepository) Load(workspace string) (model.WorkspaceTodos, error) {
	path, err := r.storage.TodosPath()
	if err != nil {
		return model.WorkspaceTodos{}, fmt.Errorf("failed to get todos path: %w", err)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Return empty TODOs if file doesn't exist.
		return model.WorkspaceTodos{
			Todos:    []model.Todo{},
			Archived: []model.Todo{},
		}, nil
	}
	if err != nil {
		return model.WorkspaceTodos{}, fmt.Errorf("failed to read todos file: %w", err)
	}

	var todos model.WorkspaceTodos
	if err := json.Unmarshal(data, &todos); err != nil {
		return model.WorkspaceTodos{}, fmt.Errorf("failed to unmarshal todos: %w", err)
	}

	// Normalize todos (ensure no nil slices)
	todos = normalizeWorkspaceTodos(todos)

	return todos, nil
}

// Save persists workspace TODOs for the given workspace.
func (r *FileSystemTodoRepository) Save(workspace string, todos model.WorkspaceTodos) error {
	path, err := r.storage.TodosPath()
	if err != nil {
		return fmt.Errorf("failed to get todos path: %w", err)
	}

	// Normalize before saving
	todos = normalizeWorkspaceTodos(todos)

	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal todos: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write todos file: %w", err)
	}

	return nil
}

// Delete removes the TODOs for the given workspace.
func (r *FileSystemTodoRepository) Delete(workspace string) error {
	path, err := r.storage.TodosPath()
	if err != nil {
		return fmt.Errorf("failed to get todos path: %w", err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete todos file: %w", err)
	}

	return nil
}

// normalizeWorkspaceTodos ensures no nil slices in the TODO structure.
func normalizeWorkspaceTodos(w model.WorkspaceTodos) model.WorkspaceTodos {
	w.Todos = normalizeTodos(w.Todos)
	w.Archived = normalizeTodos(w.Archived)
	return w
}

// normalizeTodos recursively normalizes a slice of TODOs.
func normalizeTodos(todos []model.Todo) []model.Todo {
	if todos == nil {
		return []model.Todo{}
	}
	out := make([]model.Todo, len(todos))
	for i, t := range todos {
		t.Subtodos = normalizeTodos(t.Subtodos)
		out[i] = t
	}
	return out
}
