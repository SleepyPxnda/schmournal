package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestFileSystemConfigRepository_LoadDefaultsModulesWhenAbsentInOldConfig(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFileSystemConfigRepository(dir)
	if err != nil {
		t.Fatalf("NewFileSystemConfigRepository() error = %v", err)
	}

	// Old config without a [modules] section at all (simulating pre-module config).
	cfgPath := filepath.Join(dir, "schmournal.config")
	oldCfg := `storage_path = "~/.journal"
weekly_hours_goal = 40.0
work_days = ["monday", "tuesday", "wednesday", "thursday", "friday"]

[keybinds.list]
quit = "q"
open_today = "n"
open_date = "c"
delete = "d"
export = "x"
week_view = "v"
stats_view = "s"
switch_workspace = "p"

[keybinds.day]
add_work = "w"
add_break = "b"
edit = "e"
delete = "d"
set_start_now = "s"
set_start_manual = "S"
set_end_now = "f"
set_end_manual = "F"
notes = "n"
todo_overview = "t"
export = "x"
clock_start = "c"
clock_stop = "c"
`
	if err := os.WriteFile(cfgPath, []byte(oldCfg), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// [modules] section was absent; both should default to enabled.
	if !loaded.Modules.ClockEnabled {
		t.Error("Load() ClockEnabled = false for old config without [modules], want true (default enabled)")
	}
	if !loaded.Modules.TodoEnabled {
		t.Error("Load() TodoEnabled = false for old config without [modules], want true (default enabled)")
	}

	// Migration should have been triggered; migrated file should contain [modules].
	newRaw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(newRaw), "clock_enabled") {
		t.Error("migrated config should contain clock_enabled")
	}
	if !strings.Contains(string(newRaw), "todo_enabled") {
		t.Error("migrated config should contain todo_enabled")
	}
}

func TestFileSystemConfigRepository_LoadPreservesExplicitFalseModules(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFileSystemConfigRepository(dir)
	if err != nil {
		t.Fatalf("NewFileSystemConfigRepository() error = %v", err)
	}

	// Config that explicitly disables both modules.
	cfgPath := filepath.Join(dir, "schmournal.config")
	oldCfg := `storage_path = "~/.journal"
weekly_hours_goal = 40.0
work_days = ["monday", "tuesday", "wednesday", "thursday", "friday"]

[modules]
clock_enabled = false
todo_enabled = false

[keybinds.list]
quit = "q"
open_today = "n"
open_date = "c"
delete = "d"
export = "x"
week_view = "v"
stats_view = "s"
switch_workspace = "p"

[keybinds.day]
add_work = "w"
add_break = "b"
edit = "e"
delete = "d"
set_start_now = "s"
set_start_manual = "S"
set_end_now = "f"
set_end_manual = "F"
notes = "n"
todo_overview = "t"
export = "x"
clock_start = "c"
clock_stop = "c"
`
	if err := os.WriteFile(cfgPath, []byte(oldCfg), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Modules.ClockEnabled {
		t.Error("Load() ClockEnabled = true, want false (explicitly set in config)")
	}
	if loaded.Modules.TodoEnabled {
		t.Error("Load() TodoEnabled = true, want false (explicitly set in config)")
	}
}


func TestFileSystemConfigRepository_LoadSaveRoundTrip(t *testing.T) {
	repo, err := NewFileSystemConfigRepository(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileSystemConfigRepository() error = %v", err)
	}

	cfg := model.DefaultAppConfig()
	cfg.WeeklyHoursGoal = 37.5
	cfg.WorkDays = []string{"monday", "wednesday", "friday"}

	if err := repo.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.WeeklyHoursGoal != cfg.WeeklyHoursGoal {
		t.Fatalf("WeeklyHoursGoal = %v, want %v", loaded.WeeklyHoursGoal, cfg.WeeklyHoursGoal)
	}
	if len(loaded.WorkDays) != len(cfg.WorkDays) {
		t.Fatalf("WorkDays length = %d, want %d", len(loaded.WorkDays), len(cfg.WorkDays))
	}
}

func TestFileSystemConfigRepository_LoadCreatesDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFileSystemConfigRepository(dir)
	if err != nil {
		t.Fatalf("NewFileSystemConfigRepository() error = %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	def := model.DefaultAppConfig()
	if loaded.WeeklyHoursGoal != def.WeeklyHoursGoal {
		t.Fatalf("WeeklyHoursGoal = %v, want %v", loaded.WeeklyHoursGoal, def.WeeklyHoursGoal)
	}

	cfgPath := filepath.Join(dir, "schmournal.config")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file at %s: %v", cfgPath, err)
	}
}

func TestFileSystemConfigRepository_LoadMigratesMissingKeysAndBacksUpOldConfig(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFileSystemConfigRepository(dir)
	if err != nil {
		t.Fatalf("NewFileSystemConfigRepository() error = %v", err)
	}

	cfgPath := filepath.Join(dir, "schmournal.config")
	oldCfg := `storage_path = "~/.journal"
weekly_hours_goal = 40.0
work_days = ["monday", "tuesday", "wednesday", "thursday", "friday"]

[[workspaces]]
name = "Work"
storage_path = "~/.journal/work"

[keybinds.list]
quit = "q"
open_today = "n"
open_date = "c"
delete = "d"
export = "x"
# week_view intentionally missing to simulate legacy config
stats_view = "s"
switch_workspace = "p"

[keybinds.day]
add_work = "w"
add_break = "b"
edit = "e"
delete = "d"
set_start_now = "s"
set_start_manual = "S"
set_end_now = "f"
set_end_manual = "F"
notes = "n"
todo_overview = "t"
export = "x"
clock_start = "c"
clock_stop = "c"
`
	if err := os.WriteFile(cfgPath, []byte(oldCfg), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Missing key should have been defaulted.
	if loaded.Keybinds.List.WeekView == "" {
		t.Fatal("WeekView keybind should be filled during load")
	}

	// Workspace work_days should be expanded during migration to preserve legacy behavior.
	if len(loaded.Workspaces) != 1 {
		t.Fatalf("Workspaces length = %d, want 1", len(loaded.Workspaces))
	}
	if len(loaded.Workspaces[0].WorkDays) != 7 {
		t.Fatalf("workspace WorkDays length = %d, want 7", len(loaded.Workspaces[0].WorkDays))
	}

	backupPath := filepath.Join(dir, "schmournal.old.config")
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup config at %s: %v", backupPath, err)
	}

	newRaw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(newRaw), "week_view") {
		t.Fatal("migrated config should contain week_view key")
	}
}
