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
