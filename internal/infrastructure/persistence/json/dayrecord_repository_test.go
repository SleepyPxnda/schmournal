package json

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestFileSystemDayRecordRepository_FindByDateMissingReturnsEmptyRecord(t *testing.T) {
	storage, err := NewStorageManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}
	repo := NewFileSystemDayRecordRepository(storage)

	rec, err := repo.FindByDate("2026-03-29")
	if err != nil {
		t.Fatalf("FindByDate() error = %v", err)
	}
	if rec.Date != "2026-03-29" {
		t.Fatalf("Date = %q, want %q", rec.Date, "2026-03-29")
	}
	if len(rec.Entries) != 0 {
		t.Fatalf("Entries length = %d, want 0", len(rec.Entries))
	}
}

func TestFileSystemDayRecordRepository_SaveFindExistsDeleteFlow(t *testing.T) {
	storage, err := NewStorageManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}
	repo := NewFileSystemDayRecordRepository(storage)

	record := model.DayRecord{
		Date:      "2026-03-28",
		StartTime: "09:00",
		EndTime:   "17:00",
		Entries: []model.WorkEntry{
			{ID: "1", Task: "Dev", Project: "Core", DurationMin: 120},
		},
		Notes: "done",
		TodayDone: []model.Todo{
			{
				ID:        "t1",
				Title:     "Top done",
				Completed: true,
				Subtodos: []model.Todo{
					{ID: "t1-1", Title: "Nested done", Completed: true},
				},
			},
		},
	}
	if err := repo.Save(record); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := repo.FindByDate(record.Date)
	if err != nil {
		t.Fatalf("FindByDate() error = %v", err)
	}
	if loaded.Date != record.Date || loaded.StartTime != record.StartTime || loaded.EndTime != record.EndTime || loaded.Notes != record.Notes {
		t.Fatalf("loaded record mismatch: %#v", loaded)
	}
	if len(loaded.Entries) != 1 || loaded.Entries[0].Task != "Dev" {
		t.Fatalf("loaded entries mismatch: %#v", loaded.Entries)
	}
	if len(loaded.TodayDone) != 1 || loaded.TodayDone[0].ID != "t1" {
		t.Fatalf("loaded today_done mismatch: %#v", loaded.TodayDone)
	}
	if len(loaded.TodayDone[0].Subtodos) != 1 || loaded.TodayDone[0].Subtodos[0].ID != "t1-1" {
		t.Fatalf("loaded nested today_done mismatch: %#v", loaded.TodayDone[0].Subtodos)
	}

	exists, err := repo.Exists(record.Date)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Fatal("Exists() = false, want true")
	}

	if err := repo.Delete(record.Date); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	exists, err = repo.Exists(record.Date)
	if err != nil {
		t.Fatalf("Exists() after delete error = %v", err)
	}
	if exists {
		t.Fatal("Exists() after delete = true, want false")
	}
}

func TestFileSystemDayRecordRepository_FindAllSortedAndSkipsInvalid(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewStorageManager(dir)
	if err != nil {
		t.Fatalf("NewStorageManager() error = %v", err)
	}
	repo := NewFileSystemDayRecordRepository(storage)

	for _, d := range []string{"2026-03-27", "2026-03-29", "2026-03-28"} {
		if err := repo.Save(model.DayRecord{Date: d, Entries: []model.WorkEntry{}}); err != nil {
			t.Fatalf("Save(%s) error = %v", d, err)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "2026-03-26.json"), []byte("{invalid-json"), 0o644); err != nil {
		t.Fatalf("writing invalid file: %v", err)
	}

	recs, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("FindAll() length = %d, want 3", len(recs))
	}
	want := []string{"2026-03-29", "2026-03-28", "2026-03-27"}
	for i, date := range want {
		if recs[i].Date != date {
			t.Fatalf("recs[%d].Date = %q, want %q", i, recs[i].Date, date)
		}
	}
}
