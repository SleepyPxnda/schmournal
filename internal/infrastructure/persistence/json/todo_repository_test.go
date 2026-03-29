package json

import (
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestFileSystemTodoRepository_LoadMissingReturnsEmptySlices(t *testing.T) {
	storage, err := NewStorageManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}
	repo := NewFileSystemTodoRepository(storage)

	got, err := repo.Load("default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Todos == nil || got.Archived == nil {
		t.Fatalf("expected non-nil slices, got %+v", got)
	}
	if len(got.Todos) != 0 || len(got.Archived) != 0 {
		t.Fatalf("expected empty slices, got %+v", got)
	}
}

func TestFileSystemTodoRepository_SaveLoadDeleteRoundTrip(t *testing.T) {
	storage, err := NewStorageManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}
	repo := NewFileSystemTodoRepository(storage)
	workspace := "client-a"

	want := model.WorkspaceTodos{
		Todos: []model.Todo{
			{
				ID:        "t1",
				Title:     "Top",
				Completed: false,
				Subtodos: []model.Todo{
					{ID: "t1-1", Title: "Sub", Completed: true},
				},
			},
		},
		Archived: []model.Todo{
			{ID: "a1", Title: "Archived", Completed: true},
		},
	}
	if err := repo.Save(workspace, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.Load(workspace)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Todos) != 1 || len(got.Todos[0].Subtodos) != 1 {
		t.Fatalf("loaded todos mismatch: %+v", got.Todos)
	}
	if got.Todos[0].Subtodos[0].Title != "Sub" || got.Archived[0].Title != "Archived" {
		t.Fatalf("loaded data mismatch: %+v", got)
	}

	if err := repo.Delete(workspace); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	gotAfterDelete, err := repo.Load(workspace)
	if err != nil {
		t.Fatalf("Load() after delete error = %v", err)
	}
	if len(gotAfterDelete.Todos) != 0 || len(gotAfterDelete.Archived) != 0 {
		t.Fatalf("expected empty after delete, got %+v", gotAfterDelete)
	}
}

func TestFileSystemTodoRepository_NormalizesNilSubtodos(t *testing.T) {
	storage, err := NewStorageManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}
	repo := NewFileSystemTodoRepository(storage)

	in := model.WorkspaceTodos{
		Todos: []model.Todo{
			{ID: "t1", Title: "Top", Completed: false, Subtodos: nil},
		},
		Archived: nil,
	}
	if err := repo.Save("default", in); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.Load("default")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Archived == nil || got.Todos == nil || got.Todos[0].Subtodos == nil {
		t.Fatalf("expected normalized non-nil slices, got %+v", got)
	}
}
