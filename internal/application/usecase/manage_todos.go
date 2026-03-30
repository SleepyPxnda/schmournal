package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/repository"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

// ArchiveCompletedTodosInput contains the workspace for TODO archiving.
type ArchiveCompletedTodosInput struct {
	Workspace string
}

// ArchiveCompletedTodosOutput contains the result of archiving.
type ArchiveCompletedTodosOutput struct {
	ArchivedCount  int
	RemainingCount int
	ArchivedTodos  []TodoDTO
}

// ManageTodosUseCase handles TODO operations like archiving and pruning.
// This use case orchestrates:
// 1. Loading workspace TODOs
// 2. Identifying fully completed TODOs
// 3. Moving them to archive
// 4. Pruning from active list
// 5. Saving updated TODOs
type ManageTodosUseCase struct {
	todoRepo repository.TodoRepository
	todoOps  *service.TodoOperations
}

// NewManageTodosUseCase creates a new ManageTodosUseCase.
func NewManageTodosUseCase(
	todoRepo repository.TodoRepository,
	todoOps *service.TodoOperations,
) *ManageTodosUseCase {
	return &ManageTodosUseCase{
		todoRepo: todoRepo,
		todoOps:  todoOps,
	}
}

// ArchiveCompletedTodos moves fully completed TODOs to the archive.
// This is typically called when the user leaves the day view.
func (uc *ManageTodosUseCase) ArchiveCompletedTodos(input ArchiveCompletedTodosInput) (*ArchiveCompletedTodosOutput, error) {
	// Validate input
	if input.Workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	// Load workspace TODOs
	workspaceTodos, err := uc.todoRepo.Load(input.Workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to load TODOs: %w", err)
	}

	// Collect completed TODOs with contextual parent trees for display.
	completedTodos := uc.todoOps.CollectCompletedWithContext(workspaceTodos.Todos)
	archivedCount := len(uc.todoOps.CollectFullyCompleted(workspaceTodos.Todos))

	// Prune completed TODOs from active list
	workspaceTodos.Todos = uc.todoOps.PruneCompleted(workspaceTodos.Todos)

	// Save updated TODOs
	if err := uc.todoRepo.Save(input.Workspace, workspaceTodos); err != nil {
		return nil, fmt.Errorf("failed to save TODOs: %w", err)
	}

	return &ArchiveCompletedTodosOutput{
		ArchivedCount:  archivedCount,
		RemainingCount: len(workspaceTodos.Todos),
		ArchivedTodos:  mapDomainTodosToDTO(completedTodos),
	}, nil
}
