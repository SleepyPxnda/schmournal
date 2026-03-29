package json

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStorageManager_TodosPathForWorkspace_DefaultUsesBaseDir(t *testing.T) {
	base := t.TempDir()
	storage, err := NewStorageManager(base)
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}

	got, err := storage.TodosPathForWorkspace("default")
	if err != nil {
		t.Fatalf("TodosPathForWorkspace() error = %v", err)
	}
	want := filepath.Join(base, "todos.json")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestStorageManager_TodosPathForWorkspace_NonDefaultUsesSiblingDir(t *testing.T) {
	base := t.TempDir()
	storage, err := NewStorageManager(filepath.Join(base, ".journal"))
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}

	got, err := storage.TodosPathForWorkspace("client")
	if err != nil {
		t.Fatalf("TodosPathForWorkspace() error = %v", err)
	}

	if !strings.Contains(got, ".journal-client") || !strings.HasSuffix(got, "todos.json") {
		t.Fatalf("unexpected workspace todos path: %q", got)
	}
}
