package ui

import (
	"testing"

	"github.com/sleepypxnda/schmournal/journal"
)

func TestTodoCursorsIncludeThirdLevel(t *testing.T) {
	m := Model{
		dayRecord: journal.DayRecord{
			Todos: []journal.Todo{
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
		dayRecord: journal.DayRecord{
			Todos: []journal.Todo{
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
		},
		selectedTodo: 0,
		selectedSub:  1,
		selectedSub2: -1,
	}

	if ok := m.indentSelectedTodo(); !ok {
		t.Fatalf("expected indent to level 3 to succeed")
	}

	parent := m.dayRecord.Todos[0]
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
		dayRecord: journal.DayRecord{
			Todos: []journal.Todo{
				{
					ID:    "p1",
					Title: "Parent",
					Subtodos: []journal.Todo{
						{ID: "c1", Title: "Only Child"},
					},
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
		dayRecord: journal.DayRecord{
			Todos: []journal.Todo{
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
		},
		selectedTodo: 1,
		selectedSub:  -1,
		selectedSub2: -1,
	}

	if ok := m.indentSelectedTodo(); !ok {
		t.Fatalf("expected top-level indent to succeed")
	}

	got := m.dayRecord.Todos[0].Subtodos
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
		dayRecord:    journal.DayRecord{},
		selectedPane: 1,
		dayViewTab:   0,
		selectedTodo: -1,
		selectedSub:  -1,
		selectedSub2: -1,
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
