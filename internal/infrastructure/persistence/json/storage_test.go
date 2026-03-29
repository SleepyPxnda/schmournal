package json

import (
	"path/filepath"
	"testing"
)

func TestStorageManager_TodosPath_UsesBaseDir(t *testing.T) {
	base := t.TempDir()
	storage, err := NewStorageManager(base)
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}

	got, err := storage.TodosPath()
	if err != nil {
		t.Fatalf("TodosPath() error = %v", err)
	}
	want := filepath.Join(base, "todos.json")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}
