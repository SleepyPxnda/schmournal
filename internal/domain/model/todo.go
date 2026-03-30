package model

import "fmt"

// Todo represents a single workspace-level TODO item.
// TODOs can be nested up to 3 levels deep (Subtodos contain Subtodos).
//
// Design Decision: TODOs are workspace-level (not tied to a specific day).
// This is a pure domain entity - no JSON tags, no persistence logic.
type Todo struct {
	ID        string
	Title     string
	Completed bool
	Subtodos  []Todo
}

// Validate checks if this TODO is valid according to business rules.
func (t Todo) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("todo id is required")
	}
	if t.Title == "" {
		return fmt.Errorf("todo title is required")
	}
	// Recursively validate subtodos
	for i, sub := range t.Subtodos {
		if err := sub.Validate(); err != nil {
			return fmt.Errorf("subtodo %d invalid: %w", i, err)
		}
	}
	return nil
}

// IsFullyCompleted checks if this TODO and ALL nested subtodos are completed.
// This is used for archive operations.
func (t Todo) IsFullyCompleted() bool {
	if !t.Completed {
		return false
	}
	for _, sub := range t.Subtodos {
		if !sub.IsFullyCompleted() {
			return false
		}
	}
	return true
}

// CountSubtodos returns the total number of subtodos (including nested).
func (t Todo) CountSubtodos() int {
	count := len(t.Subtodos)
	for _, sub := range t.Subtodos {
		count += sub.CountSubtodos()
	}
	return count
}

// WorkspaceTodos holds the global TODO list for a workspace.
// This includes both active TODOs and archived (completed) ones.
type WorkspaceTodos struct {
	Todos    []Todo
	Archived []Todo
}

// Validate checks if the workspace TODOs structure is valid.
func (w WorkspaceTodos) Validate() error {
	for i, t := range w.Todos {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("active todo %d invalid: %w", i, err)
		}
	}
	for i, t := range w.Archived {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("archived todo %d invalid: %w", i, err)
		}
	}
	return nil
}

// CountActiveTodos returns the total number of active TODOs (including nested).
func (w WorkspaceTodos) CountActiveTodos() int {
	count := len(w.Todos)
	for _, t := range w.Todos {
		count += t.CountSubtodos()
	}
	return count
}

// CountArchivedTodos returns the total number of archived TODOs (including nested).
func (w WorkspaceTodos) CountArchivedTodos() int {
	count := len(w.Archived)
	for _, t := range w.Archived {
		count += t.CountSubtodos()
	}
	return count
}
