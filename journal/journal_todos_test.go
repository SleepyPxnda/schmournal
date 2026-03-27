package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingTodosDefaultsToEmptySlice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-03-25.json")
	raw := `{
  "date": "2026-03-25",
  "start_time": "09:00",
  "end_time": "17:00",
  "entries": [],
  "notes": "legacy file"
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	rec, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if rec.Todos == nil {
		t.Fatal("Load() left Todos nil, want empty slice")
	}
	if len(rec.Todos) != 0 {
		t.Fatalf("Load() Todos len=%d, want 0", len(rec.Todos))
	}
}

func TestSaveDoesNotWriteTodosField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-03-26.json")
	rec := DayRecord{
		Date:      "2026-03-26",
		StartTime: "09:00",
		EndTime:   "17:00",
		Entries:   []WorkEntry{},
		Notes:     "new file",
		Path:      path,
	}
	if err := Save(rec); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	v, ok := decoded["todos"]
	if ok {
		t.Fatalf("saved JSON unexpectedly contains legacy \"todos\" field: %#v", v)
	}
}

func TestWorkspaceTodosRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := SetStoragePath(dir); err != nil {
		t.Fatalf("SetStoragePath() error: %v", err)
	}
	t.Cleanup(func() {
		_ = SetStoragePath("")
	})

	in := WorkspaceTodos{
		Todos: []Todo{
			{
				ID:        "1",
				Title:     "Top",
				Completed: false,
				Subtodos: []Todo{
					{ID: "2", Title: "Sub", Completed: true},
				},
			},
		},
	}
	if err := SaveWorkspaceTodos(in); err != nil {
		t.Fatalf("SaveWorkspaceTodos() error: %v", err)
	}
	out, err := LoadWorkspaceTodos()
	if err != nil {
		t.Fatalf("LoadWorkspaceTodos() error: %v", err)
	}
	if len(out.Todos) != 1 || out.Todos[0].Title != "Top" {
		t.Fatalf("unexpected workspace todos: %#v", out.Todos)
	}
	if len(out.Todos[0].Subtodos) != 1 || out.Todos[0].Subtodos[0].Title != "Sub" {
		t.Fatalf("unexpected nested workspace todos: %#v", out.Todos[0].Subtodos)
	}
}
