package service

import (
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// ─── IsFullyCompleted ─────────────────────────────────────────────────────────

func TestTodoOperations_IsFullyCompleted_SingleTodoCompleted(t *testing.T) {
	ops := NewTodoOperations()
	todo := model.Todo{
		ID:        "a",
		Title:     "Complete task",
		Completed: true,
		Subtodos:  []model.Todo{},
	}

	if !ops.IsFullyCompleted(todo) {
		t.Error("expected single completed TODO to be fully completed")
	}
}

func TestTodoOperations_IsFullyCompleted_SingleTodoIncomplete(t *testing.T) {
	ops := NewTodoOperations()
	todo := model.Todo{
		ID:        "a",
		Title:     "Incomplete task",
		Completed: false,
		Subtodos:  []model.Todo{},
	}

	if ops.IsFullyCompleted(todo) {
		t.Error("expected incomplete TODO to not be fully completed")
	}
}

func TestTodoOperations_IsFullyCompleted_ParentCompletedChildIncomplete(t *testing.T) {
	ops := NewTodoOperations()
	todo := model.Todo{
		ID:        "a",
		Title:     "Parent",
		Completed: true,
		Subtodos: []model.Todo{
			{ID: "a1", Title: "Child", Completed: false, Subtodos: []model.Todo{}},
		},
	}

	if ops.IsFullyCompleted(todo) {
		t.Error("expected parent with incomplete child to not be fully completed")
	}
}

func TestTodoOperations_IsFullyCompleted_AllLevelsCompleted(t *testing.T) {
	ops := NewTodoOperations()
	todo := model.Todo{
		ID:        "a",
		Title:     "Parent",
		Completed: true,
		Subtodos: []model.Todo{
			{
				ID:        "a1",
				Title:     "Child",
				Completed: true,
				Subtodos: []model.Todo{
					{ID: "a1a", Title: "Grandchild", Completed: true, Subtodos: []model.Todo{}},
				},
			},
		},
	}

	if !ops.IsFullyCompleted(todo) {
		t.Error("expected fully completed 3-level tree to be fully completed")
	}
}

// ─── CollectFullyCompleted ────────────────────────────────────────────────────

func TestTodoOperations_CollectFullyCompleted_ReturnsFullyDoneOnly(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{ID: "a", Title: "Done", Completed: true, Subtodos: []model.Todo{}},
		{ID: "b", Title: "Not done", Completed: false, Subtodos: []model.Todo{}},
	}

	result := ops.CollectFullyCompleted(todos)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("expected only fully-done todo, got %+v", result)
	}
}

func TestTodoOperations_CollectFullyCompleted_SkipsParentWithIncompleteChild(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []model.Todo{
				{ID: "a1", Title: "Child not done", Completed: false, Subtodos: []model.Todo{}},
			},
		},
	}

	result := ops.CollectFullyCompleted(todos)
	if len(result) != 0 {
		t.Fatalf("expected no fully-done todo (child incomplete), got %+v", result)
	}
}

func TestTodoOperations_CollectFullyCompleted_IncludesFullyDoneTree(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []model.Todo{
				{ID: "a1", Title: "Child done", Completed: true, Subtodos: []model.Todo{}},
			},
		},
		{ID: "b", Title: "Keep me", Completed: false, Subtodos: []model.Todo{}},
	}

	result := ops.CollectFullyCompleted(todos)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("expected fully-done tree to be collected, got %+v", result)
	}
	if len(result[0].Subtodos) != 1 || result[0].Subtodos[0].ID != "a1" {
		t.Fatalf("expected subtree to be preserved, got %+v", result[0].Subtodos)
	}
}

func TestTodoOperations_CollectFullyCompleted_EmptyList(t *testing.T) {
	ops := NewTodoOperations()
	result := ops.CollectFullyCompleted(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty result for nil input, got %+v", result)
	}
}

// ─── PruneCompleted ───────────────────────────────────────────────────────────

func TestTodoOperations_PruneCompleted_RemovesFullyDone(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{ID: "a", Title: "Done", Completed: true, Subtodos: []model.Todo{}},
		{ID: "b", Title: "Not done", Completed: false, Subtodos: []model.Todo{}},
	}

	result := ops.PruneCompleted(todos)
	if len(result) != 1 || result[0].ID != "b" {
		t.Fatalf("expected only 'Not done' to remain, got %+v", result)
	}
}

func TestTodoOperations_PruneCompleted_KeepsParentWithIncompleteChild(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []model.Todo{
				{ID: "a1", Title: "Child not done", Completed: false, Subtodos: []model.Todo{}},
			},
		},
	}

	result := ops.PruneCompleted(todos)
	if len(result) != 1 || result[0].ID != "a" {
		t.Fatalf("expected parent to remain because child is incomplete, got %+v", result)
	}
}

func TestTodoOperations_PruneCompleted_RemovesFullyDoneNested(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Parent done",
			Completed: true,
			Subtodos: []model.Todo{
				{ID: "a1", Title: "Child done", Completed: true, Subtodos: []model.Todo{}},
			},
		},
		{ID: "b", Title: "Keep me", Completed: false, Subtodos: []model.Todo{}},
	}

	result := ops.PruneCompleted(todos)
	if len(result) != 1 || result[0].ID != "b" {
		t.Fatalf("expected only 'Keep me' to remain, got %+v", result)
	}
}

func TestTodoOperations_PruneCompleted_PrunesCompletedSubtodoFromIncompleteParent(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Parent incomplete",
			Completed: false,
			Subtodos: []model.Todo{
				{ID: "a1", Title: "Done child", Completed: true, Subtodos: []model.Todo{}},
				{ID: "a2", Title: "Incomplete child", Completed: false, Subtodos: []model.Todo{}},
			},
		},
	}

	result := ops.PruneCompleted(todos)
	if len(result) != 1 {
		t.Fatalf("expected parent to remain, got %+v", result)
	}
	if len(result[0].Subtodos) != 1 || result[0].Subtodos[0].ID != "a2" {
		t.Fatalf("expected only incomplete child to remain, got subtodos %+v", result[0].Subtodos)
	}
}

// ─── ValidateTodoTree ─────────────────────────────────────────────────────────

func TestTodoOperations_ValidateTodoTree_ValidTree(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Level 0",
			Completed: false,
			Subtodos: []model.Todo{
				{
					ID:        "a1",
					Title:     "Level 1",
					Completed: false,
					Subtodos: []model.Todo{
						{ID: "a1a", Title: "Level 2", Completed: false, Subtodos: []model.Todo{}},
					},
				},
			},
		},
	}

	err := ops.ValidateTodoTree(todos, 0)
	if err != nil {
		t.Errorf("expected valid tree to pass validation, got error: %v", err)
	}
}

func TestTodoOperations_ValidateTodoTree_EmptyTitle(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{ID: "a", Title: "", Completed: false, Subtodos: []model.Todo{}},
	}

	err := ops.ValidateTodoTree(todos, 0)
	if err == nil {
		t.Error("expected validation error for empty title")
	}
	if err.Error() != "TODO title cannot be empty" {
		t.Errorf("expected 'TODO title cannot be empty', got %v", err)
	}
}

func TestTodoOperations_ValidateTodoTree_TooDeep(t *testing.T) {
	ops := NewTodoOperations()
	todos := []model.Todo{
		{
			ID:        "a",
			Title:     "Level 0",
			Completed: false,
			Subtodos: []model.Todo{
				{
					ID:        "a1",
					Title:     "Level 1",
					Completed: false,
					Subtodos: []model.Todo{
						{
							ID:        "a1a",
							Title:     "Level 2",
							Completed: false,
							Subtodos: []model.Todo{
								{ID: "a1a1", Title: "Level 3 (too deep)", Completed: false, Subtodos: []model.Todo{}},
							},
						},
					},
				},
			},
		},
	}

	err := ops.ValidateTodoTree(todos, 0)
	if err == nil {
		t.Error("expected validation error for excessive nesting")
	}
	if err.Error() != "TODO nesting exceeds maximum depth of 3 levels" {
		t.Errorf("expected depth error, got %v", err)
	}
}
