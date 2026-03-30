package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
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

	// Collect fully completed TODOs
	completedTodos := uc.todoOps.CollectFullyCompleted(workspaceTodos.Todos)
	archivedCount := len(completedTodos)

	// Prune completed TODOs from active list
	workspaceTodos.Todos = uc.todoOps.PruneCompleted(workspaceTodos.Todos)

	// Add completed TODOs to archive
	workspaceTodos.Archived = append(workspaceTodos.Archived, completedTodos...)

	// Save updated TODOs
	if err := uc.todoRepo.Save(input.Workspace, workspaceTodos); err != nil {
		return nil, fmt.Errorf("failed to save TODOs: %w", err)
	}

	return &ArchiveCompletedTodosOutput{
		ArchivedCount:  archivedCount,
		RemainingCount: len(workspaceTodos.Todos),
	}, nil
}

// ClearArchive removes all archived TODOs.
func (uc *ManageTodosUseCase) ClearArchive(input ArchiveCompletedTodosInput) error {
	// Validate input
	if input.Workspace == "" {
		return fmt.Errorf("workspace is required")
	}

	// Load workspace TODOs
	workspaceTodos, err := uc.todoRepo.Load(input.Workspace)
	if err != nil {
		return fmt.Errorf("failed to load TODOs: %w", err)
	}

	// Clear archive
	workspaceTodos.Archived = []model.Todo{}

	// Save updated TODOs
	if err := uc.todoRepo.Save(input.Workspace, workspaceTodos); err != nil {
		return fmt.Errorf("failed to save TODOs: %w", err)
	}

	return nil
}
