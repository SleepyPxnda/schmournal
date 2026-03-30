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
		Todos: []domainmodel.Todo{},
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

func TestArchiveCompletedTodosCmdReturnsArchivedTodayPayload(t *testing.T) {
	uc := newUseCasesWithMockTodos(t, domainmodel.WorkspaceTodos{
		Todos: []domainmodel.Todo{
			{ID: "a", Title: "Done", Completed: true},
			{ID: "b", Title: "Keep", Completed: false},
		},
	})
	m := newDayViewTestModel(t)
	m.context.UseCases = uc
	m.context.ActiveWorkspace = "default"

	msg := m.archiveCompletedTodosCmd("")()
	managed, ok := msg.(workspaceTodosManagedMsg)
	if !ok {
		t.Fatalf("expected workspaceTodosManagedMsg, got %T", msg)
	}

	if len(managed.archivedToday) != 1 || managed.archivedToday[0].ID != "a" {
		t.Fatalf("expected completed todo in archivedToday payload, got %+v", managed.archivedToday)
	}
}

func TestUpdateWorkspaceTodosManagedMsgRefreshesDayViewportAndStatus(t *testing.T) {
	m := newDayViewTestModel(t)
	m.ui.Current = stateDayView
	m.day.Selection.DayTab = 0

	updated, _ := m.Update(workspaceTodosManagedMsg{
		todos: WorkspaceTodos{Todos: []Todo{{ID: "1", Title: "Updated", Completed: false}}},
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
		todos:         WorkspaceTodos{Todos: []Todo{}},
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

func TestUpdateWorkspaceTodosManagedMsgResolvesTodayDonePlaceholderWhenParentCompletes(t *testing.T) {
	m := newDayViewTestModel(t)
	m.day.Record.TodayDone = []Todo{
		{
			ID:        "parent",
			Title:     "Parent",
			Completed: false,
			Subtodos: []Todo{
				{ID: "child", Title: "Child", Completed: true},
			},
		},
	}

	updated, cmd := m.Update(workspaceTodosManagedMsg{
		todos: WorkspaceTodos{Todos: []Todo{}},
		archivedToday: []Todo{
			{
				ID:        "parent",
				Title:     "Parent",
				Completed: true,
				Subtodos: []Todo{
					{ID: "child", Title: "Child", Completed: true},
				},
			},
		},
	})
	got := updated.(Model)

	if len(got.day.Record.TodayDone) != 1 {
		t.Fatalf("expected merged today done to keep one parent entry, got %+v", got.day.Record.TodayDone)
	}
	if !got.day.Record.TodayDone[0].Completed {
		t.Fatalf("expected placeholder parent to resolve to completed, got %+v", got.day.Record.TodayDone[0])
	}
	if len(got.day.Record.TodayDone[0].Subtodos) != 1 || got.day.Record.TodayDone[0].Subtodos[0].ID != "child" {
		t.Fatalf("expected merged child subtree to be preserved, got %+v", got.day.Record.TodayDone[0].Subtodos)
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
	if len(got.workspace.Todos) != 0 {
		t.Fatalf("expected no immediate local todo mutation when manage use case is configured, got todos=%+v", got.workspace.Todos)
	}
}
