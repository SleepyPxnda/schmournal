package repository

import (
	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// TodoRepository defines the interface for persisting workspace-level TODOs.
// This is a domain interface - implementations live in infrastructure layer.
type TodoRepository interface {
	// Load reads workspace TODOs for the given workspace.
	Load(workspace string) (model.WorkspaceTodos, error)

	// Save persists workspace TODOs for the given workspace.
	Save(workspace string, todos model.WorkspaceTodos) error

	// Delete removes the TODOs for the given workspace.
	Delete(workspace string) error
}
