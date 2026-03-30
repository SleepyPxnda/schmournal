package service

import "github.com/sleepypxnda/schmournal/internal/domain/model"

// TodoOperations provides business logic for TODO management.
// This service handles complex operations like pruning, archiving, and validation.
//
// Design Decision: TODO manipulation is core business logic and belongs in domain.
// The UI should only orchestrate calls to this service.
type TodoOperations struct{}

// NewTodoOperations creates a new TodoOperations service.
func NewTodoOperations() *TodoOperations {
	return &TodoOperations{}
}

// IsFullyCompleted reports whether t and every nested subtodo are all completed.
// A TODO is considered "fully completed" when:
// - The TODO itself is marked as completed
// - AND all its subtodos (recursively) are marked as completed
//
// This is used to determine which TODOs should be archived or pruned.
func (s *TodoOperations) IsFullyCompleted(t model.Todo) bool {
	if !t.Completed {
		return false
	}
	for _, sub := range t.Subtodos {
		if !s.IsFullyCompleted(sub) {
			return false
		}
	}
	return true
}

// CollectFullyCompleted returns all top-level todos (and their subtree) for
// which IsFullyCompleted is true.
//
// This is used to collect TODOs that should be moved to the archive when
// the user leaves the day view. Only top-level TODOs are returned; their
// completed subtrees come along with them.
func (s *TodoOperations) CollectFullyCompleted(todos []model.Todo) []model.Todo {
	var result []model.Todo
	for _, t := range todos {
		if s.IsFullyCompleted(t) {
			result = append(result, t)
		}
	}
	return result
}

// CollectCompletedWithContext returns TODO trees that should be archived:
//   - Fully completed top-level TODOs are returned as-is.
//   - If a TODO is not fully completed but contains completed descendants,
//     it is returned as a contextual parent (Completed=false) containing only
//     the completed descendant branches.
func (s *TodoOperations) CollectCompletedWithContext(todos []model.Todo) []model.Todo {
	var result []model.Todo
	for _, t := range todos {
		if projected, ok := s.projectCompletedTree(t); ok {
			result = append(result, projected)
		}
	}
	return result
}

func (s *TodoOperations) projectCompletedTree(todo model.Todo) (model.Todo, bool) {
	if s.IsFullyCompleted(todo) {
		return todo, true
	}

	projectedChildren := make([]model.Todo, 0, len(todo.Subtodos))
	for _, sub := range todo.Subtodos {
		if projected, ok := s.projectCompletedTree(sub); ok {
			projectedChildren = append(projectedChildren, projected)
		}
	}
	if len(projectedChildren) == 0 {
		return model.Todo{}, false
	}

	todo.Completed = false
	todo.Subtodos = projectedChildren
	return todo, true
}

// PruneCompleted removes todos (at any depth) where the todo itself and
// all its descendants are completed.
//
// Partial branches (some children incomplete) are kept intact and have their
// subtodos recursively pruned. This ensures that incomplete work is never lost.
//
// Example:
// Input:  [TodoA(✓), TodoB(—) with SubB1(✓), SubB2(—)]
// Output: [TodoB(—) with SubB2(—)]
//
// TodoA is fully complete → removed
// TodoB is incomplete → kept, but SubB1 is complete → removed
// SubB2 is incomplete → kept
func (s *TodoOperations) PruneCompleted(todos []model.Todo) []model.Todo {
	result := make([]model.Todo, 0, len(todos))
	for _, t := range todos {
		if s.IsFullyCompleted(t) {
			continue
		}
		// Recursively prune subtodos
		t.Subtodos = s.PruneCompleted(t.Subtodos)
		result = append(result, t)
	}
	return result
}

// ValidateTodoTree performs deep validation of a TODO tree.
// Returns an error if:
// - A TODO title is empty
// - A TODO has invalid nesting (more than 2 levels deep)
// - Circular references exist (not yet implemented, would require ID tracking)
func (s *TodoOperations) ValidateTodoTree(todos []model.Todo, currentDepth int) error {
	const maxDepth = 2 // Support only 3 levels (0, 1, 2)

	for _, t := range todos {
		// Empty title validation
		if t.Title == "" {
			return &ValidationError{Message: "TODO title cannot be empty"}
		}

		// Depth validation
		if currentDepth > maxDepth {
			return &ValidationError{Message: "TODO nesting exceeds maximum depth of 3 levels"}
		}

		// Recursive validation of subtodos
		if len(t.Subtodos) > 0 {
			if err := s.ValidateTodoTree(t.Subtodos, currentDepth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidationError is returned when a TODO tree fails validation.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
