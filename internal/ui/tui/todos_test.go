package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestTodoCursorsIncludeThirdLevel(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{
				ID:    "p1",
				Title: "Parent",
				Subtodos: []Todo{
					{
						ID:    "c1",
						Title: "Child",
						Subtodos: []Todo{
							{ID: "g1", Title: "Grandchild"},
						},
					},
				},
			},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: -1, Sub2: -1},
	}

	m.todoMove(2)

	if m.todoSelection.Top != 0 || m.todoSelection.Sub != 0 || m.todoSelection.Sub2 != 0 {
		t.Fatalf("expected to navigate to third-level todo, got top=%d sub=%d sub2=%d", m.todoSelection.Top, m.todoSelection.Sub, m.todoSelection.Sub2)
	}
}

func TestIndentLevelTwoTodoToLevelThreeFlattensChildren(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{
				ID:    "p1",
				Title: "Parent",
				Subtodos: []Todo{
					{ID: "c1", Title: "Child A"},
					{
						ID:    "c2",
						Title: "Child B",
						Subtodos: []Todo{
							{ID: "g1", Title: "Grandchild"},
						},
					},
				},
			},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: 1, Sub2: -1},
	}

	if ok := m.indentSelectedTodo(); !ok {
		t.Fatalf("expected indent to level 3 to succeed")
	}

	parent := m.workspace.Todos[0]
	if len(parent.Subtodos) != 1 {
		t.Fatalf("expected one level-2 todo after indent, got %d", len(parent.Subtodos))
	}
	if len(parent.Subtodos[0].Subtodos) != 2 {
		t.Fatalf("expected moved todo and its child to be clamped at level 3, got %d", len(parent.Subtodos[0].Subtodos))
	}
	if parent.Subtodos[0].Subtodos[0].Title != "Child B" || parent.Subtodos[0].Subtodos[1].Title != "Grandchild" {
		t.Fatalf("unexpected level-3 todo order after clamp: %#v", parent.Subtodos[0].Subtodos)
	}
}

func TestIndentGateRequiresParentAtPreviousLevel(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{
				ID:    "p1",
				Title: "Parent",
				Subtodos: []Todo{
					{ID: "c1", Title: "Only Child"},
				},
			},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: 0, Sub2: -1},
	}

	if ok := m.indentSelectedTodo(); ok {
		t.Fatalf("expected indent to fail without a previous level-2 sibling")
	}
}

func TestIndentTopLevelClampsNestedDepthToThree(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{ID: "p1", Title: "Parent"},
			{
				ID:    "p2",
				Title: "Child Parent",
				Subtodos: []Todo{
					{
						ID:    "c1",
						Title: "Level 2",
						Subtodos: []Todo{
							{
								ID:    "g1",
								Title: "Level 3",
								Subtodos: []Todo{
									{ID: "g2", Title: "Level 4"},
								},
							},
						},
					},
				},
			},
		}},
		todoSelection: TodoSelectionState{Top: 1, Sub: -1, Sub2: -1},
	}

	if ok := m.indentSelectedTodo(); !ok {
		t.Fatalf("expected top-level indent to succeed")
	}

	got := m.workspace.Todos[0].Subtodos
	if len(got) != 1 {
		t.Fatalf("expected one level-2 child under new parent, got len=%d", len(got))
	}
	if got[0].Title != "Child Parent" {
		t.Fatalf("unexpected moved todo after top-level indent: %#v", got[0])
	}
	if len(got[0].Subtodos) != 3 {
		t.Fatalf("expected descendants to be clamped at level-3 list, got len=%d", len(got[0].Subtodos))
	}
	if got[0].Subtodos[0].Title != "Level 2" || got[0].Subtodos[1].Title != "Level 3" || got[0].Subtodos[2].Title != "Level 4" {
		t.Fatalf("unexpected descendant order after clamp: %#v", got[0].Subtodos)
	}
	for _, child := range got[0].Subtodos {
		if len(child.Subtodos) != 0 {
			t.Fatalf("expected clamped descendants to have no deeper subtodos: %#v", got[0].Subtodos)
		}
	}
}

// TestTodoDraftAcceptsJAndKWhenTyping verifies that "j" and "k" are appended to
// the todo draft when the user is already typing (todoDraft non-empty), instead
// of being silently swallowed as navigation keys.
func TestTodoDraftAcceptsJAndKWhenTyping(t *testing.T) {
	m := Model{
		workspace:     WorkspaceDataState{Todos: []Todo{}},
		day:           DayViewState{Selection: SelectionState{Pane: 1, DayTab: 0}},
		todoSelection: TodoSelectionState{Top: -1, Sub: -1, Sub2: -1},
	}

	// Seed the draft so that j/k must be treated as text, not navigation.
	m.todoEditor.Draft = "tas"

	for _, ch := range []string{"k", "j"} {
		m.appendTodoDraft(ch)
	}

	want := "taskj"
	if m.todoEditor.Draft != want {
		t.Fatalf("expected todoDraft %q, got %q", want, m.todoEditor.Draft)
	}
}

func TestTodoKeyTogglesPaneFocus(t *testing.T) {
	m := Model{
		context:    AppContextState{Config: model.DefaultAppConfig()},
		day:        DayViewState{Selection: SelectionState{DayTab: 0, Pane: 0}},
		todoEditor: TodoEditorState{Draft: "stale", InputMode: false},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	got := updated.(Model)
	if got.day.Selection.Pane != 1 {
		t.Fatalf("expected todo pane to be focused, got pane=%d", got.day.Selection.Pane)
	}

	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	got = updated.(Model)
	if got.day.Selection.Pane != 0 {
		t.Fatalf("expected todo pane to close and return to worklog pane, got pane=%d", got.day.Selection.Pane)
	}
	if got.todoEditor.InputMode {
		t.Fatalf("expected todo input mode to be disabled when closing todo pane")
	}
	if got.todoEditor.Draft != "" {
		t.Fatalf("expected todo draft to be cleared when closing todo pane, got %q", got.todoEditor.Draft)
	}
}

func TestTodoEnterEnablesInputThenSavesInlineInTodoPane(t *testing.T) {
	m := Model{
		context:       AppContextState{Config: model.DefaultAppConfig()},
		day:           DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
		todoSelection: TodoSelectionState{Top: -1, Sub: -1, Sub2: -1},
		workspace:     WorkspaceDataState{Todos: []Todo{}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if !got.todoEditor.InputMode {
		t.Fatalf("expected enter to enable todo input mode")
	}

	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	got = updated.(Model)
	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got = updated.(Model)
	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	got = updated.(Model)
	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got = updated.(Model)

	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(Model)

	if got.todoEditor.InputMode {
		t.Fatalf("expected todo input mode to be disabled after submit")
	}
	if got.day.Selection.Pane != 1 {
		t.Fatalf("expected to stay in todo pane after submit, got pane=%d", got.day.Selection.Pane)
	}
	if len(got.workspace.Todos) != 1 || got.workspace.Todos[0].Title != "task" {
		t.Fatalf("expected one saved todo titled task, got %#v", got.workspace.Todos)
	}
}

func TestTodoTypingStartsInlineInputMode(t *testing.T) {
	m := Model{
		context: AppContextState{Config: model.DefaultAppConfig()},
		day:     DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	got := updated.(Model)
	if !got.todoEditor.InputMode {
		t.Fatalf("expected typing in todo pane to start inline input mode")
	}
	if got.todoEditor.Draft != "z" {
		t.Fatalf("expected first typed rune to seed inline todo draft, got %q", got.todoEditor.Draft)
	}
}

func TestTodoAddKeyStartsInlineInputInsteadOfModal(t *testing.T) {
	m := Model{
		context: AppContextState{Config: model.DefaultAppConfig()},
		ui:      UIState{Current: stateDayView},
		day:     DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(Model)
	if got.ui.Current != stateDayView {
		t.Fatalf("expected to remain in day view (no create modal), got state=%v", got.ui.Current)
	}
	if !got.todoEditor.InputMode {
		t.Fatalf("expected add key to enter inline todo input mode")
	}
	if got.todoEditor.Draft != "" {
		t.Fatalf("expected add key to open empty inline draft, got %q", got.todoEditor.Draft)
	}
}

func TestTodoEditKeyStaysInDayViewWithoutSelection(t *testing.T) {
	m := Model{
		context:       AppContextState{Config: model.DefaultAppConfig()},
		ui:            UIState{Current: stateDayView},
		window:        WindowState{Width: 120},
		day:           DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
		todoSelection: TodoSelectionState{Top: -1, Sub: -1, Sub2: -1},
		workspace:     WorkspaceDataState{Todos: []Todo{}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(m.context.Config.Keybinds.Day.Edit)})
	got := updated.(Model)
	if got.ui.Current != stateDayView {
		t.Fatalf("expected to stay in day view when editing without selection, got state=%v", got.ui.Current)
	}
}

func TestTodoPaneToggleKeyDoesNotStartInlineTyping(t *testing.T) {
	m := Model{
		context: AppContextState{Config: model.DefaultAppConfig()},
		day:     DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}) // Toggle todo pane
	got := updated.(Model)
	if got.todoEditor.InputMode {
		t.Fatalf("expected command key to not start inline draft mode")
	}
	if got.todoEditor.Draft != "" {
		t.Fatalf("expected command key to not seed draft, got %q", got.todoEditor.Draft)
	}
}

func TestTodoNonPrintableDoesNotStartInlineTyping(t *testing.T) {
	m := Model{
		context: AppContextState{Config: model.DefaultAppConfig()},
		day:     DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(Model)
	if got.todoEditor.InputMode {
		t.Fatalf("expected non-printable key to not start inline draft mode")
	}
	if got.todoEditor.Draft != "" {
		t.Fatalf("expected non-printable key to not seed draft, got %q", got.todoEditor.Draft)
	}
}

func TestTodoCustomDayKeybindDoesNotStartInlineTyping(t *testing.T) {
	cfg := model.DefaultAppConfig()
	cfg.Keybinds.Day.TodoOverview = "z"
	m := Model{
		context: AppContextState{Config: cfg},
		day:     DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	got := updated.(Model)
	if got.todoEditor.InputMode {
		t.Fatalf("expected custom day keybind to be treated as command key")
	}
	if got.todoEditor.Draft != "" {
		t.Fatalf("expected custom command key to not seed draft, got %q", got.todoEditor.Draft)
	}
}

func TestTodoInputModeSwallowsMutationAndNavigationKeys(t *testing.T) {
	m := Model{
		context:       AppContextState{Config: model.DefaultAppConfig()},
		day:           DayViewState{Selection: SelectionState{DayTab: 0, Pane: 1}},
		todoEditor:    TodoEditorState{InputMode: true},
		todoSelection: TodoSelectionState{Top: 0, Sub: -1, Sub2: -1},
		workspace: WorkspaceDataState{Todos: []Todo{
			{
				ID:    "t1",
				Title: "Top 1",
			},
			{
				ID:    "t2",
				Title: "Top 2",
			},
		}},
	}

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyDelete},
		{Type: tea.KeyShiftTab},
		{Type: tea.KeyTab},
		{Type: tea.KeyDown},
		{Type: tea.KeyUp},
	} {
		updated, _ := m.handleDayViewKey(key)
		m = updated.(Model)
	}

	if len(m.workspace.Todos) != 2 {
		t.Fatalf("expected todo list to stay unchanged in input mode, got len=%d", len(m.workspace.Todos))
	}
	if m.todoSelection.Top != 0 || m.todoSelection.Sub != -1 || m.todoSelection.Sub2 != -1 {
		t.Fatalf("expected selection to stay unchanged in input mode, got top=%d sub=%d sub2=%d", m.todoSelection.Top, m.todoSelection.Sub, m.todoSelection.Sub2)
	}
}

func TestTodosPanelUsesSingleDraftHintAcrossEnterToggle(t *testing.T) {
	const (
		primaryHint = "type to add, enter to save"
		legacyHint  = "type to add a todo, enter to save"
	)

	m := Model{
		ui:         UIState{Current: stateDayView},
		day:        DayViewState{Selection: SelectionState{Pane: 1, DayTab: 0}},
		todoEditor: TodoEditorState{InputMode: false, Draft: ""},
		workspace:  WorkspaceDataState{Todos: []Todo{}},
		context:    AppContextState{Config: model.AppConfig{Modules: model.Modules{TodoEnabled: true}}},
	}

	panelBefore := m.renderTodosPanel(60)
	if !strings.Contains(panelBefore, primaryHint) {
		t.Fatalf("expected default draft hint before enter, got:\n%s", panelBefore)
	}
	if strings.Contains(panelBefore, legacyHint) {
		t.Fatalf("unexpected legacy hint before enter, got:\n%s", panelBefore)
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if !got.todoEditor.InputMode {
		t.Fatalf("expected enter to enable todo input mode")
	}
	panelAfter := got.renderTodosPanel(60)
	if !strings.Contains(panelAfter, primaryHint) {
		t.Fatalf("expected same draft hint after enter, got:\n%s", panelAfter)
	}
	if strings.Contains(panelAfter, legacyHint) {
		t.Fatalf("unexpected alternate hint after enter, got:\n%s", panelAfter)
	}
}

func TestTodosPanelShowsInlineDraftInInputMode(t *testing.T) {
	m := Model{
		day:        DayViewState{Selection: SelectionState{Pane: 1, DayTab: 0}},
		todoEditor: TodoEditorState{InputMode: true, Draft: "draft task"},
		workspace:  WorkspaceDataState{Todos: []Todo{}},
	}

	rendered := m.renderTodosPanel(60)
	if !strings.Contains(rendered, "+ draft task") {
		t.Fatalf("expected inline draft line while in input mode, got:\n%s", rendered)
	}
}

func TestTodosPanelShowsTodayDoneSection(t *testing.T) {
	m := Model{
		day: DayViewState{
			Selection: SelectionState{Pane: 1, DayTab: 0},
			Record: DayRecord{
				TodayDone: []Todo{
					{ID: "a", Title: "Done today", Completed: true},
				},
			},
		},
		workspace: WorkspaceDataState{Todos: []Todo{}},
	}

	rendered := m.renderTodosPanel(60)
	if !strings.Contains(rendered, "Today Done") {
		t.Fatalf("expected Today Done section, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "✓ Done today") {
		t.Fatalf("expected today-done todo line, got:\n%s", rendered)
	}
}

func TestTodosPanelShowsDashedContextParentsAndDoneNodes(t *testing.T) {
	m := Model{
		day: DayViewState{
			Selection: SelectionState{Pane: 1, DayTab: 0},
			Record: DayRecord{
				TodayDone: []Todo{
					{
						ID:        "p",
						Title:     "Parent context",
						Completed: false,
						Subtodos: []Todo{
							{ID: "c", Title: "Done child", Completed: true},
						},
					},
				},
			},
		},
		workspace: WorkspaceDataState{Todos: []Todo{}},
	}

	rendered := m.renderTodosPanel(60)
	if !strings.Contains(rendered, "- Parent context") {
		t.Fatalf("expected dashed context parent line, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "✓ Done child") {
		t.Fatalf("expected done child line, got:\n%s", rendered)
	}
}

func TestConfirmDeleteUsesTodoSubjectForTodoDeletion(t *testing.T) {
	m := Model{
		window:        WindowState{Width: 120, Height: 40},
		delete:        DeleteState{Day: false, Idx: deleteTodoIdx},
		todoSelection: TodoSelectionState{Top: 0, Sub: -1, Sub2: -1},
		workspace: WorkspaceDataState{Todos: []Todo{
			{ID: "t1", Title: "Ship release"},
		}},
	}

	view := m.viewConfirmDelete()
	if !strings.Contains(view, `Delete todo "Ship release"?`) {
		t.Fatalf("expected todo-specific delete prompt, got:\n%s", view)
	}
	if strings.Contains(view, `Delete entry "`) {
		t.Fatalf("expected no entry wording for todo deletion, got:\n%s", view)
	}
}

// ─── pruneCompletedTodos ──────────────────────────────────────────────────────

func TestPruneCompletedTodosRemovesFullyDone(t *testing.T) {
	todos := []Todo{
		{ID: "a", Title: "Done", Completed: true, Subtodos: []Todo{}},
		{ID: "b", Title: "Not done", Completed: false, Subtodos: []Todo{}},
	}
	result := pruneCompletedTodos(todos)
	if len(result) != 1 || result[0].ID != "b" {
		t.Fatalf("expected only 'Not done' to remain, got %+v", result)
	}
}

func TestPruneCompletedTodosKeepsParentWithIncompleteChild(t *testing.T) {
	todos := []Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []Todo{
				{ID: "a1", Title: "Child not done", Completed: false, Subtodos: []Todo{}},
			},
		},
	}
	result := pruneCompletedTodos(todos)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("expected parent to remain because child is incomplete, got %+v", result)
	}
}

func TestPruneCompletedTodosRemovesFullyDoneNested(t *testing.T) {
	todos := []Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []Todo{
				{ID: "a1", Title: "Child done", Completed: true, Subtodos: []Todo{}},
			},
		},
		{ID: "b", Title: "Keep me", Completed: false, Subtodos: []Todo{}},
	}
	result := pruneCompletedTodos(todos)
	if len(result) != 1 || result[0].ID != "b" {
		t.Fatalf("expected only 'Keep me' to remain, got %+v", result)
	}
}

func TestPruneCompletedTodosPrunesCompletedSubtodoFromIncompleteParent(t *testing.T) {
	todos := []Todo{
		{
			ID:        "a",
			Title:     "Parent incomplete",
			Completed: false,
			Subtodos: []Todo{
				{ID: "a1", Title: "Done child", Completed: true, Subtodos: []Todo{}},
				{ID: "a2", Title: "Incomplete child", Completed: false, Subtodos: []Todo{}},
			},
		},
	}
	result := pruneCompletedTodos(todos)
	if len(result) != 1 {
		t.Fatalf("expected parent to remain, got %+v", result)
	}
	if len(result[0].Subtodos) != 1 || result[0].Subtodos[0].ID != "a2" {
		t.Fatalf("expected only incomplete child to remain, got subtodos %+v", result[0].Subtodos)
	}
}

// ─── collectFullyCompleted ────────────────────────────────────────────────────

func TestCollectFullyCompletedReturnsFullyDoneOnly(t *testing.T) {
	todos := []Todo{
		{ID: "a", Title: "Done", Completed: true, Subtodos: []Todo{}},
		{ID: "b", Title: "Not done", Completed: false, Subtodos: []Todo{}},
	}
	result := collectFullyCompleted(todos)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("expected only fully-done todo, got %+v", result)
	}
}

func TestCollectFullyCompletedSkipsParentWithIncompleteChild(t *testing.T) {
	todos := []Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []Todo{
				{ID: "a1", Title: "Child not done", Completed: false, Subtodos: []Todo{}},
			},
		},
	}
	result := collectFullyCompleted(todos)
	if len(result) != 0 {
		t.Fatalf("expected no fully-done todo (child incomplete), got %+v", result)
	}
}

func TestCollectFullyCompletedIncludesFullyDoneTree(t *testing.T) {
	todos := []Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []Todo{
				{ID: "a1", Title: "Child done", Completed: true, Subtodos: []Todo{}},
			},
		},
		{ID: "b", Title: "Keep me", Completed: false, Subtodos: []Todo{}},
	}
	result := collectFullyCompleted(todos)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("expected fully-done tree to be collected, got %+v", result)
	}
	if len(result[0].Subtodos) != 1 || result[0].Subtodos[0].ID != "a1" {
		t.Fatalf("expected subtree to be preserved, got %+v", result[0].Subtodos)
	}
}

func TestCollectFullyCompletedEmptyList(t *testing.T) {
	result := collectFullyCompleted(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty result for nil input, got %+v", result)
	}
}

// ─── moveSelectedTodoDelta ────────────────────────────────────────────────────

func TestMoveSelectedTodoDeltaTopLevel(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{ID: "a", Title: "A"},
			{ID: "b", Title: "B"},
			{ID: "c", Title: "C"},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: -1, Sub2: -1},
	}

	ok := m.moveSelectedTodoDelta(1)
	if !ok {
		t.Fatal("expected move to succeed")
	}
	if m.workspace.Todos[0].ID != "b" || m.workspace.Todos[1].ID != "a" {
		t.Fatalf("expected A to move down, got %v %v", m.workspace.Todos[0].ID, m.workspace.Todos[1].ID)
	}
	if m.todoSelection.Top != 1 {
		t.Fatalf("expected selectedTodo=1, got %d", m.todoSelection.Top)
	}
}

func TestMoveSelectedTodoDeltaTopLevelBoundary(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{ID: "a", Title: "A"},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: -1, Sub2: -1},
	}

	ok := m.moveSelectedTodoDelta(-1)
	if ok {
		t.Fatal("expected move to fail at boundary")
	}
}

func TestMoveSelectedTodoDeltaLevelTwo(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{
				ID:    "p",
				Title: "Parent",
				Subtodos: []Todo{
					{ID: "c1", Title: "Child 1"},
					{ID: "c2", Title: "Child 2"},
				},
			},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: 0, Sub2: -1},
	}

	ok := m.moveSelectedTodoDelta(1)
	if !ok {
		t.Fatal("expected move to succeed")
	}
	if m.workspace.Todos[0].Subtodos[0].ID != "c2" || m.workspace.Todos[0].Subtodos[1].ID != "c1" {
		t.Fatalf("expected c1 to move down, got %v %v", m.workspace.Todos[0].Subtodos[0].ID, m.workspace.Todos[0].Subtodos[1].ID)
	}
	if m.todoSelection.Sub != 1 {
		t.Fatalf("expected selectedSub=1, got %d", m.todoSelection.Sub)
	}
}

func TestMoveSelectedTodoDeltaLevelThree(t *testing.T) {
	m := Model{
		workspace: WorkspaceDataState{Todos: []Todo{
			{
				ID:    "p",
				Title: "Parent",
				Subtodos: []Todo{
					{
						ID:    "c1",
						Title: "Child",
						Subtodos: []Todo{
							{ID: "g1", Title: "Grand 1"},
							{ID: "g2", Title: "Grand 2"},
						},
					},
				},
			},
		}},
		todoSelection: TodoSelectionState{Top: 0, Sub: 0, Sub2: 0},
	}

	ok := m.moveSelectedTodoDelta(1)
	if !ok {
		t.Fatal("expected move to succeed")
	}
	subs := m.workspace.Todos[0].Subtodos[0].Subtodos
	if subs[0].ID != "g2" || subs[1].ID != "g1" {
		t.Fatalf("expected g1 to move down, got %v %v", subs[0].ID, subs[1].ID)
	}
	if m.todoSelection.Sub2 != 1 {
		t.Fatalf("expected selectedSub2=1, got %d", m.todoSelection.Sub2)
	}
}
