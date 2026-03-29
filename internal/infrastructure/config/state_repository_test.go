package config

import (
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestFileSystemStateRepository_LoadMissingReturnsEmptyState(t *testing.T) {
	repo := NewFileSystemStateRepository(t.TempDir())

	state, err := repo.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.ActiveWorkspace != "" {
		t.Fatalf("ActiveWorkspace = %q, want empty", state.ActiveWorkspace)
	}
}

func TestFileSystemStateRepository_SaveLoadRoundTrip(t *testing.T) {
	repo := NewFileSystemStateRepository(t.TempDir())
	want := model.AppState{ActiveWorkspace: "work"}

	if err := repo.SaveState(want); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	got, err := repo.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if got.ActiveWorkspace != want.ActiveWorkspace {
		t.Fatalf("ActiveWorkspace = %q, want %q", got.ActiveWorkspace, want.ActiveWorkspace)
	}
}
