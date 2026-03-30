package usecase

import (
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

// MockTodoRepository is a simple in-memory repository for testing.
type MockTodoRepository struct {
	workspaces map[string]model.WorkspaceTodos
}

func NewMockTodoRepository() *MockTodoRepository {
	return &MockTodoRepository{
		workspaces: make(map[string]model.WorkspaceTodos),
	}
}

func (m *MockTodoRepository) Load(workspace string) (model.WorkspaceTodos, error) {
	if wt, ok := m.workspaces[workspace]; ok {
		return wt, nil
	}
	// Return empty WorkspaceTodos if not found
	return model.WorkspaceTodos{
		Todos:    []model.Todo{},
		Archived: []model.Todo{},
	}, nil
}

func (m *MockTodoRepository) Save(workspace string, todos model.WorkspaceTodos) error {
	m.workspaces[workspace] = todos
	return nil
}

func (m *MockTodoRepository) Delete(workspace string) error {
	delete(m.workspaces, workspace)
	return nil
}

var _ repository.TodoRepository = (*MockTodoRepository)(nil)

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestManageTodos_ArchiveCompletedTodos_Success(t *testing.T) {
	repo := NewMockTodoRepository()
	todoOps := service.NewTodoOperations()
	useCase := NewManageTodosUseCase(repo, todoOps)

	// Create workspace with some TODOs
	workspace := model.WorkspaceTodos{
		Todos: []model.Todo{
			{ID: "1", Title: "Completed", Completed: true, Subtodos: []model.Todo{}},
			{ID: "2", Title: "Incomplete", Completed: false, Subtodos: []model.Todo{}},
		},
		Archived: []model.Todo{},
	}
	_ = repo.Save("default", workspace)

	// Archive completed TODOs
	input := ArchiveCompletedTodosInput{Workspace: "default"}
	output, err := useCase.ArchiveCompletedTodos(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.ArchivedCount != 1 {
		t.Errorf("expected 1 archived, got %d", output.ArchivedCount)
	}
	if output.RemainingCount != 1 {
		t.Errorf("expected 1 remaining, got %d", output.RemainingCount)
	}
	if len(output.ArchivedTodos) != 1 || output.ArchivedTodos[0].ID != "1" {
		t.Errorf("expected archived todo IDs in output, got %+v", output.ArchivedTodos)
	}

	// Verify state
	saved, err := repo.Load("default")
	if err != nil {
		t.Fatalf("failed to load saved todos: %v", err)
	}
	if len(saved.Todos) != 1 || saved.Todos[0].ID != "2" {
		t.Error("expected only incomplete TODO to remain")
	}
	if len(saved.Archived) != 1 || saved.Archived[0].ID != "1" {
		t.Error("expected completed TODO to be archived")
	}
}

func TestManageTodos_ArchiveCompletedTodos_NestedTodos(t *testing.T) {
	repo := NewMockTodoRepository()
	todoOps := service.NewTodoOperations()
	useCase := NewManageTodosUseCase(repo, todoOps)

	// Create workspace with nested TODOs
	workspace := model.WorkspaceTodos{
		Todos: []model.Todo{
			{
				ID:        "1",
				Title:     "Parent completed",
				Completed: true,
				Subtodos: []model.Todo{
					{ID: "1a", Title: "Child completed", Completed: true, Subtodos: []model.Todo{}},
				},
			},
			{
				ID:        "2",
				Title:     "Parent incomplete",
				Completed: false,
				Subtodos: []model.Todo{
					{ID: "2a", Title: "Child completed", Completed: true, Subtodos: []model.Todo{}},
				},
			},
		},
		Archived: []model.Todo{},
	}
	_ = repo.Save("default", workspace)

	// Archive completed TODOs
	input := ArchiveCompletedTodosInput{Workspace: "default"}
	output, err := useCase.ArchiveCompletedTodos(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// First todo is fully complete → archived
	// Second todo is incomplete but has a completed child:
	// - child should be pruned from active todos
	// - archived output should include contextual parent branch
	if output.ArchivedCount != 1 {
		t.Errorf("expected 1 archived (fully complete tree), got %d", output.ArchivedCount)
	}
	if len(output.ArchivedTodos) != 2 {
		t.Fatalf("expected fully done tree + contextual branch in output, got %+v", output.ArchivedTodos)
	}
	if output.ArchivedTodos[1].ID != "2" || output.ArchivedTodos[1].Completed {
		t.Fatalf("expected second archived output item to be incomplete parent context, got %+v", output.ArchivedTodos[1])
	}
	if len(output.ArchivedTodos[1].Subtodos) != 1 || output.ArchivedTodos[1].Subtodos[0].ID != "2a" {
		t.Fatalf("expected contextual parent to include completed child, got %+v", output.ArchivedTodos[1].Subtodos)
	}

	// Verify state
	saved, err := repo.Load("default")
	if err != nil {
		t.Fatalf("failed to load saved todos: %v", err)
	}

	// Only parent 2 should remain, but its completed child should be pruned
	if len(saved.Todos) != 1 {
		t.Fatalf("expected 1 remaining todo, got %d", len(saved.Todos))
	}
	if saved.Todos[0].ID != "2" {
		t.Error("expected parent 2 to remain")
	}
	if len(saved.Todos[0].Subtodos) != 0 {
		t.Errorf("expected completed child to be pruned, got %d subtodos", len(saved.Todos[0].Subtodos))
	}

	// Archived should contain both:
	// - fully complete tree (parent 1 + child 1a)
	// - contextual branch for parent 2 -> child 2a
	if len(saved.Archived) != 2 {
		t.Fatalf("expected 2 archived todos, got %d", len(saved.Archived))
	}
	if saved.Archived[0].ID != "1" {
		t.Error("expected parent 1 to be archived")
	}
	if len(saved.Archived[0].Subtodos) != 1 {
		t.Error("expected child 1a to be archived with parent")
	}
	if saved.Archived[1].ID != "2" || saved.Archived[1].Completed {
		t.Fatalf("expected parent 2 context to be archived as incomplete reference, got %+v", saved.Archived[1])
	}
	if len(saved.Archived[1].Subtodos) != 1 || saved.Archived[1].Subtodos[0].ID != "2a" {
		t.Fatalf("expected contextual branch to include completed child 2a, got %+v", saved.Archived[1].Subtodos)
	}
}

func TestManageTodos_ClearArchive_Success(t *testing.T) {
	repo := NewMockTodoRepository()
	todoOps := service.NewTodoOperations()
	useCase := NewManageTodosUseCase(repo, todoOps)

	// Create workspace with archived TODOs
	workspace := model.WorkspaceTodos{
		Todos: []model.Todo{},
		Archived: []model.Todo{
			{ID: "1", Title: "Archived 1", Completed: true, Subtodos: []model.Todo{}},
			{ID: "2", Title: "Archived 2", Completed: true, Subtodos: []model.Todo{}},
		},
	}
	_ = repo.Save("default", workspace)

	// Clear archive
	input := ArchiveCompletedTodosInput{Workspace: "default"}
	err := useCase.ClearArchive(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify archive is cleared
	saved, err := repo.Load("default")
	if err != nil {
		t.Fatalf("failed to load saved todos: %v", err)
	}
	if len(saved.Archived) != 0 {
		t.Errorf("expected archive to be empty, got %d items", len(saved.Archived))
	}
}

func TestManageTodos_ValidationErrors(t *testing.T) {
	repo := NewMockTodoRepository()
	todoOps := service.NewTodoOperations()
	useCase := NewManageTodosUseCase(repo, todoOps)

	// Missing workspace
	_, err := useCase.ArchiveCompletedTodos(ArchiveCompletedTodosInput{Workspace: ""})
	if err == nil {
		t.Error("expected error for missing workspace")
	}

	err = useCase.ClearArchive(ArchiveCompletedTodosInput{Workspace: ""})
	if err == nil {
		t.Error("expected error for missing workspace in clear archive")
	}
}
