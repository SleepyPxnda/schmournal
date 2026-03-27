package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/config"
	"github.com/sleepypxnda/schmournal/journal"
)

func TestTodoCursorsIncludeThirdLevel(t *testing.T) {
	m := Model{
		workspaceTodos: []journal.Todo{
			{
				ID:    "p1",
				Title: "Parent",
				Subtodos: []journal.Todo{
					{
						ID:    "c1",
						Title: "Child",
						Subtodos: []journal.Todo{
							{ID: "g1", Title: "Grandchild"},
						},
					},
				},
			},
		},
		selectedTodo: 0,
		selectedSub:  -1,
		selectedSub2: -1,
	}

	m.todoMove(2)

	if m.selectedTodo != 0 || m.selectedSub != 0 || m.selectedSub2 != 0 {
		t.Fatalf("expected to navigate to third-level todo, got top=%d sub=%d sub2=%d", m.selectedTodo, m.selectedSub, m.selectedSub2)
	}
}

func TestIndentLevelTwoTodoToLevelThreeFlattensChildren(t *testing.T) {
	m := Model{
		workspaceTodos: []journal.Todo{
			{
				ID:    "p1",
				Title: "Parent",
				Subtodos: []journal.Todo{
					{ID: "c1", Title: "Child A"},
					{
						ID:    "c2",
						Title: "Child B",
						Subtodos: []journal.Todo{
							{ID: "g1", Title: "Grandchild"},
						},
					},
				},
			},
		},
		selectedTodo: 0,
		selectedSub:  1,
		selectedSub2: -1,
	}

	if ok := m.indentSelectedTodo(); !ok {
		t.Fatalf("expected indent to level 3 to succeed")
	}

	parent := m.workspaceTodos[0]
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
		workspaceTodos: []journal.Todo{
			{
				ID:    "p1",
				Title: "Parent",
				Subtodos: []journal.Todo{
					{ID: "c1", Title: "Only Child"},
				},
			},
		},
		selectedTodo: 0,
		selectedSub:  0,
		selectedSub2: -1,
	}

	if ok := m.indentSelectedTodo(); ok {
		t.Fatalf("expected indent to fail without a previous level-2 sibling")
	}
}

func TestIndentTopLevelClampsNestedDepthToThree(t *testing.T) {
	m := Model{
		workspaceTodos: []journal.Todo{
			{ID: "p1", Title: "Parent"},
			{
				ID:    "p2",
				Title: "Child Parent",
				Subtodos: []journal.Todo{
					{
						ID:    "c1",
						Title: "Level 2",
						Subtodos: []journal.Todo{
							{
								ID:    "g1",
								Title: "Level 3",
								Subtodos: []journal.Todo{
									{ID: "g2", Title: "Level 4"},
								},
							},
						},
					},
				},
			},
		},
		selectedTodo: 1,
		selectedSub:  -1,
		selectedSub2: -1,
	}

	if ok := m.indentSelectedTodo(); !ok {
		t.Fatalf("expected top-level indent to succeed")
	}

	got := m.workspaceTodos[0].Subtodos
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
		workspaceTodos: []journal.Todo{},
		selectedPane:   1,
		dayViewTab:     0,
		selectedTodo:   -1,
		selectedSub:    -1,
		selectedSub2:   -1,
	}

	// Seed the draft so that j/k must be treated as text, not navigation.
	m.todoDraft = "tas"

	for _, ch := range []string{"k", "j"} {
		m.appendTodoDraft(ch)
	}

	want := "taskj"
	if m.todoDraft != want {
		t.Fatalf("expected todoDraft %q, got %q", want, m.todoDraft)
	}
}

func TestTodoKeyTogglesPaneFocus(t *testing.T) {
	m := Model{
		cfg:           config.Default(),
		dayViewTab:    0,
		selectedPane:  0,
		todoDraft:     "stale",
		todoInputMode: false,
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	got := updated.(Model)
	if got.selectedPane != 1 {
		t.Fatalf("expected todo pane to be focused, got pane=%d", got.selectedPane)
	}

	updated, _ = got.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	got = updated.(Model)
	if got.selectedPane != 0 {
		t.Fatalf("expected todo pane to close and return to worklog pane, got pane=%d", got.selectedPane)
	}
	if got.todoInputMode {
		t.Fatalf("expected todo input mode to be disabled when closing todo pane")
	}
	if got.todoDraft != "" {
		t.Fatalf("expected todo draft to be cleared when closing todo pane, got %q", got.todoDraft)
	}
}

func TestTodoEnterEnablesInputThenSavesInlineInTodoPane(t *testing.T) {
	m := Model{
		cfg:            config.Default(),
		dayViewTab:     0,
		selectedPane:   1,
		selectedTodo:   -1,
		selectedSub:    -1,
		selectedSub2:   -1,
		workspaceTodos: []journal.Todo{},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if !got.todoInputMode {
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

	if got.todoInputMode {
		t.Fatalf("expected todo input mode to be disabled after submit")
	}
	if got.selectedPane != 1 {
		t.Fatalf("expected to stay in todo pane after submit, got pane=%d", got.selectedPane)
	}
	if len(got.workspaceTodos) != 1 || got.workspaceTodos[0].Title != "task" {
		t.Fatalf("expected one saved todo titled task, got %#v", got.workspaceTodos)
	}
}

func TestTodoTypingStartsInlineInputMode(t *testing.T) {
	m := Model{
		cfg:          config.Default(),
		dayViewTab:   0,
		selectedPane: 1,
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	got := updated.(Model)
	if !got.todoInputMode {
		t.Fatalf("expected typing in todo pane to start inline input mode")
	}
	if got.todoDraft != "z" {
		t.Fatalf("expected first typed rune to seed inline todo draft, got %q", got.todoDraft)
	}
}

func TestTodoAddKeyStartsInlineInputInsteadOfModal(t *testing.T) {
	m := Model{
		cfg:          config.Default(),
		state:        stateDayView,
		dayViewTab:   0,
		selectedPane: 1,
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(Model)
	if got.state != stateDayView {
		t.Fatalf("expected to remain in day view (no create modal), got state=%v", got.state)
	}
	if !got.todoInputMode {
		t.Fatalf("expected add key to enter inline todo input mode")
	}
	if got.todoDraft != "" {
		t.Fatalf("expected add key to open empty inline draft, got %q", got.todoDraft)
	}
}

func TestTodoEditKeyStaysInDayViewWithoutSelection(t *testing.T) {
	m := Model{
		cfg:            config.Default(),
		state:          stateDayView,
		width:          120,
		dayViewTab:     0,
		selectedPane:   1,
		selectedTodo:   -1,
		selectedSub:    -1,
		selectedSub2:   -1,
		workspaceTodos: []journal.Todo{},
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(m.cfg.Keybinds.Day.Edit)})
	got := updated.(Model)
	if got.state != stateDayView {
		t.Fatalf("expected to stay in day view when editing without selection, got state=%v", got.state)
	}
}

func TestTodoCommandKeyDoesNotStartInlineTyping(t *testing.T) {
	m := Model{
		cfg:          config.Default(),
		dayViewTab:   0,
		selectedPane: 1,
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}) // TodoOverview
	got := updated.(Model)
	if got.todoInputMode {
		t.Fatalf("expected command key to not start inline draft mode")
	}
	if got.todoDraft != "" {
		t.Fatalf("expected command key to not seed draft, got %q", got.todoDraft)
	}
}

func TestTodoNonPrintableDoesNotStartInlineTyping(t *testing.T) {
	m := Model{
		cfg:          config.Default(),
		dayViewTab:   0,
		selectedPane: 1,
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(Model)
	if got.todoInputMode {
		t.Fatalf("expected non-printable key to not start inline draft mode")
	}
	if got.todoDraft != "" {
		t.Fatalf("expected non-printable key to not seed draft, got %q", got.todoDraft)
	}
}

func TestTodoCustomDayKeybindDoesNotStartInlineTyping(t *testing.T) {
	cfg := config.Default()
	cfg.Keybinds.Day.TodoOverview = "z"
	m := Model{
		cfg:          cfg,
		dayViewTab:   0,
		selectedPane: 1,
	}

	updated, _ := m.handleDayViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	got := updated.(Model)
	if got.todoInputMode {
		t.Fatalf("expected custom day keybind to be treated as command key")
	}
	if got.todoDraft != "" {
		t.Fatalf("expected custom command key to not seed draft, got %q", got.todoDraft)
	}
}

func TestTodoInputModeSwallowsMutationAndNavigationKeys(t *testing.T) {
	m := Model{
		cfg:           config.Default(),
		dayViewTab:    0,
		selectedPane:  1,
		todoInputMode: true,
		selectedTodo:  0,
		selectedSub:   -1,
		selectedSub2:  -1,
		workspaceTodos: []journal.Todo{
			{
				ID:    "t1",
				Title: "Top 1",
			},
			{
				ID:    "t2",
				Title: "Top 2",
			},
		},
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

	if len(m.workspaceTodos) != 2 {
		t.Fatalf("expected todo list to stay unchanged in input mode, got len=%d", len(m.workspaceTodos))
	}
	if m.selectedTodo != 0 || m.selectedSub != -1 || m.selectedSub2 != -1 {
		t.Fatalf("expected selection to stay unchanged in input mode, got top=%d sub=%d sub2=%d", m.selectedTodo, m.selectedSub, m.selectedSub2)
	}
}

func TestTodosPanelUsesSingleDraftHintAcrossEnterToggle(t *testing.T) {
	const (
		primaryHint = "type to add, enter to save"
		legacyHint  = "type to add a todo, enter to save"
	)

	m := Model{
		state:          stateDayView,
		selectedPane:   1,
		dayViewTab:     0,
		todoInputMode:  false,
		todoDraft:      "",
		workspaceTodos: []journal.Todo{},
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
	if !got.todoInputMode {
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
		selectedPane:   1,
		dayViewTab:     0,
		todoInputMode:  true,
		todoDraft:      "draft task",
		workspaceTodos: []journal.Todo{},
	}

	rendered := m.renderTodosPanel(60)
	if !strings.Contains(rendered, "+ draft task") {
		t.Fatalf("expected inline draft line while in input mode, got:\n%s", rendered)
	}
}

func TestConfirmDeleteUsesTodoSubjectForTodoDeletion(t *testing.T) {
	m := Model{
		width:        120,
		height:       40,
		deleteDay:    false,
		deleteIdx:    deleteTodoIdx,
		selectedTodo: 0,
		selectedSub:  -1,
		selectedSub2: -1,
		workspaceTodos: []journal.Todo{
			{ID: "t1", Title: "Ship release"},
		},
	}

	view := m.viewConfirmDelete()
	if !strings.Contains(view, `Delete todo "Ship release"?`) {
		t.Fatalf("expected todo-specific delete prompt, got:\n%s", view)
	}
	if strings.Contains(view, `Delete entry "`) {
		t.Fatalf("expected no entry wording for todo deletion, got:\n%s", view)
	}
}
