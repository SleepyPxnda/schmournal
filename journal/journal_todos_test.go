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

func TestSaveAlwaysWritesTodosField(t *testing.T) {
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
	if !ok {
		t.Fatal("saved JSON missing \"todos\" field")
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("\"todos\" has wrong type %T, want array", v)
	}
	if len(arr) != 0 {
		t.Fatalf("\"todos\" len=%d, want 0", len(arr))
	}
}
