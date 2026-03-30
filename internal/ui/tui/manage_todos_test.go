package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/application/usecase"
	domainmodel "github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

type mockTodoRepo struct {
	workspaces map[string]domainmodel.WorkspaceTodos
}

func newMockTodoRepo() *mockTodoRepo {
	return &mockTodoRepo{workspaces: make(map[string]domainmodel.WorkspaceTodos)}
}

func (m *mockTodoRepo) Load(workspace string) (domainmodel.WorkspaceTodos, error) {
	if wt, ok := m.workspaces[workspace]; ok {
		return wt, nil
	}
	return domainmodel.WorkspaceTodos{
		Todos:    []domainmodel.Todo{},
		Archived: []domainmodel.Todo{},
	}, nil
}

func (m *mockTodoRepo) Save(workspace string, todos domainmodel.WorkspaceTodos) error {
	m.workspaces[workspace] = todos
	return nil
}

func (m *mockTodoRepo) Delete(workspace string) error {
	delete(m.workspaces, workspace)
	return nil
}

func newUseCasesWithMockTodos(t *testing.T, workspaceTodos domainmodel.WorkspaceTodos) *UseCases {
	t.Helper()

	repo := newMockTodoRepo()
	if err := repo.Save("default", workspaceTodos); err != nil {
		t.Fatalf("failed to seed mock todo repository: %v", err)
	}

	return NewUseCases(UseCaseSet{
		LoadWorkspaceTodos: usecase.NewLoadWorkspaceTodosUseCase(repo),
		ManageTodos:        usecase.NewManageTodosUseCase(repo, service.NewTodoOperations()),
	}, nil)
}

func TestArchiveCompletedTodosCmdArchivesAndReloadsWorkspaceTodos(t *testing.T) {
	uc := newUseCasesWithMockTodos(t, domainmodel.WorkspaceTodos{
		Todos: []domainmodel.Todo{
			{ID: "a", Title: "Done", Completed: true},
			{ID: "b", Title: "Keep", Completed: false},
		},
		Archived: []domainmodel.Todo{},
	})
	m := newDayViewTestModel(t)
	m.context.UseCases = uc
	m.context.ActiveWorkspace = "default"

	msg := m.archiveCompletedTodosCmd("")()
	managed, ok := msg.(workspaceTodosManagedMsg)
	if !ok {
		t.Fatalf("expected workspaceTodosManagedMsg, got %T", msg)
	}

	if len(managed.todos.Todos) != 1 || managed.todos.Todos[0].ID != "b" {
		t.Fatalf("expected remaining incomplete todo only, got %+v", managed.todos.Todos)
	}
	if len(managed.todos.Archived) != 1 || managed.todos.Archived[0].ID != "a" {
		t.Fatalf("expected completed todo archived, got %+v", managed.todos.Archived)
	}
}

func TestClearArchiveCmdClearsArchiveAndReturnsLabel(t *testing.T) {
	uc := newUseCasesWithMockTodos(t, domainmodel.WorkspaceTodos{
		Todos: []domainmodel.Todo{},
		Archived: []domainmodel.Todo{
			{ID: "a", Title: "Archived", Completed: true},
		},
	})
	m := newDayViewTestModel(t)
	m.context.UseCases = uc
	m.context.ActiveWorkspace = "default"

	msg := m.clearArchiveCmd("✓ Archive cleared")()
	managed, ok := msg.(workspaceTodosManagedMsg)
	if !ok {
		t.Fatalf("expected workspaceTodosManagedMsg, got %T", msg)
	}

	if len(managed.todos.Archived) != 0 {
		t.Fatalf("expected archive to be cleared, got %+v", managed.todos.Archived)
	}
	if managed.label != "✓ Archive cleared" {
		t.Fatalf("expected status label to round-trip, got %q", managed.label)
	}
}

func TestUpdateWorkspaceTodosManagedMsgRefreshesDayViewportAndStatus(t *testing.T) {
	m := newDayViewTestModel(t)
	m.ui.Current = stateDayView
	m.day.Selection.DayTab = 0

	updated, _ := m.Update(workspaceTodosManagedMsg{
		todos: WorkspaceTodos{
			Todos:    []Todo{{ID: "1", Title: "Updated", Completed: false}},
			Archived: []Todo{},
		},
		label: "✓ Todos updated",
	})
	got := updated.(Model)

	if len(got.workspace.Todos) != 1 || got.workspace.Todos[0].Title != "Updated" {
		t.Fatalf("expected workspace todos updated from managed msg, got %+v", got.workspace.Todos)
	}
	if got.status.Message != "✓ Todos updated" || got.status.IsError {
		t.Fatalf("expected success status from managed msg, got message=%q err=%v", got.status.Message, got.status.IsError)
	}
	if got.day.Viewport.View() == "" {
		t.Fatalf("expected day viewport content to be refreshed")
	}
}

func TestUpdateWorkspaceTodosManagedMsgTracksTodayDoneOnDayRecord(t *testing.T) {
	m := newDayViewTestModel(t)
	m.ui.Current = stateList

	updated, cmd := m.Update(workspaceTodosManagedMsg{
		todos: WorkspaceTodos{
			Todos:    []Todo{},
			Archived: []Todo{{ID: "arch", Title: "Archived", Completed: true}},
		},
		archivedToday: []Todo{{ID: "done", Title: "Done today", Completed: true}},
	})
	got := updated.(Model)

	if len(got.day.Record.TodayDone) != 1 || got.day.Record.TodayDone[0].ID != "done" {
		t.Fatalf("expected today done to be appended on day record, got %+v", got.day.Record.TodayDone)
	}
	if cmd == nil {
		t.Fatalf("expected save-day command when archivedToday is present")
	}
}

func TestDayEscUsesManageTodosWhenConfigured(t *testing.T) {
	uc := newUseCasesWithMockTodos(t, domainmodel.WorkspaceTodos{
		Todos: []domainmodel.Todo{
			{ID: "a", Title: "Done", Completed: true},
			{ID: "b", Title: "Keep", Completed: false},
		},
		Archived: []domainmodel.Todo{},
	})

	m := newDayViewTestModel(t)
	m.context.UseCases = uc
	m.context.ActiveWorkspace = "default"
	m.clock.Running = true

	updated, cmd := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)

	if got.ui.Current != stateList {
		t.Fatalf("expected esc to return to list, got state=%v", got.ui.Current)
	}
	if cmd == nil {
		t.Fatalf("expected esc to return batched command")
	}

	// Managed-todo path should defer todo mutation to command messages
	// instead of mutating local workspace state immediately.
	if len(got.workspace.Todos) != 0 || len(got.workspace.Archived) != 0 {
		t.Fatalf("expected no immediate local todo mutation when manage use case is configured, got todos=%+v archived=%+v", got.workspace.Todos, got.workspace.Archived)
	}
}
