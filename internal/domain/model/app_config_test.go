package model

import (
	"testing"
	"time"
)

func TestDefaultAppConfigModulesEnabled(t *testing.T) {
	cfg := DefaultAppConfig()
	if !cfg.Modules.ClockEnabled {
		t.Error("DefaultAppConfig().Modules.ClockEnabled = false, want true")
	}
	if !cfg.Modules.TodoEnabled {
		t.Error("DefaultAppConfig().Modules.TodoEnabled = false, want true")
	}
}

func TestValidateAndNormalizePreservesModuleSettings(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Modules.ClockEnabled = false
	cfg.Modules.TodoEnabled = false

	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}
	if cfg.Modules.ClockEnabled {
		t.Error("ValidateAndNormalize() reset ClockEnabled to true, want false preserved")
	}
	if cfg.Modules.TodoEnabled {
		t.Error("ValidateAndNormalize() reset TodoEnabled to true, want false preserved")
	}
}


func TestDefaultAppConfigWeeklyHoursGoal(t *testing.T) {
	cfg := DefaultAppConfig()
	if cfg.WeeklyHoursGoal != 40 {
		t.Errorf("WeeklyHoursGoal = %g, want 40", cfg.WeeklyHoursGoal)
	}
}

func TestDefaultAppConfigStoragePath(t *testing.T) {
	cfg := DefaultAppConfig()
	if cfg.StoragePath != "~/.journal" {
		t.Errorf("StoragePath = %q, want %q", cfg.StoragePath, "~/.journal")
	}
}

func TestDefaultAppConfigKeybindsNotEmpty(t *testing.T) {
	cfg := DefaultAppConfig()

	checkNotEmpty := func(field, value string) {
		t.Helper()
		if value == "" {
			t.Errorf("default keybind for %s is empty", field)
		}
	}

	lk := cfg.Keybinds.List
	checkNotEmpty("list.quit", lk.Quit)
	checkNotEmpty("list.open_today", lk.OpenToday)
	checkNotEmpty("list.open_date", lk.OpenDate)
	checkNotEmpty("list.delete", lk.Delete)
	checkNotEmpty("list.week_view", lk.WeekView)
	checkNotEmpty("list.stats_view", lk.StatsView)
	checkNotEmpty("list.switch_workspace", lk.SwitchWorkspace)

	dk := cfg.Keybinds.Day
	checkNotEmpty("day.add_work", dk.AddWork)
	checkNotEmpty("day.add_break", dk.AddBreak)
	checkNotEmpty("day.edit", dk.Edit)
	checkNotEmpty("day.delete", dk.Delete)
	checkNotEmpty("day.set_start_now", dk.SetStartNow)
	checkNotEmpty("day.set_start_manual", dk.SetStartManual)
	checkNotEmpty("day.set_end_now", dk.SetEndNow)
	checkNotEmpty("day.set_end_manual", dk.SetEndManual)
	checkNotEmpty("day.notes", dk.Notes)
	checkNotEmpty("day.todo_overview", dk.TodoOverview)
	checkNotEmpty("day.clock_start", dk.ClockStart)
	checkNotEmpty("day.clock_stop", dk.ClockStop)
}

func TestValidateAndNormalizeFillsEmptyKeybinds(t *testing.T) {
	cfg := AppConfig{WeeklyHoursGoal: 40}
	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}

	def := DefaultAppConfig()
	if cfg.Keybinds.List.Quit != def.Keybinds.List.Quit {
		t.Errorf("List.Quit = %q, want %q", cfg.Keybinds.List.Quit, def.Keybinds.List.Quit)
	}
	if cfg.Keybinds.Day.AddWork != def.Keybinds.Day.AddWork {
		t.Errorf("Day.AddWork = %q, want %q", cfg.Keybinds.Day.AddWork, def.Keybinds.Day.AddWork)
	}
}

func TestValidateAndNormalizeZeroWeeklyHoursGoalReset(t *testing.T) {
	cfg := AppConfig{WeeklyHoursGoal: 0}
	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}
	if cfg.WeeklyHoursGoal <= 0 {
		t.Errorf("WeeklyHoursGoal = %g after ValidateAndNormalize, want > 0", cfg.WeeklyHoursGoal)
	}
}

func TestValidateAndNormalizeDuplicateListKeybindReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Keybinds.List.Quit = "q"
	cfg.Keybinds.List.OpenToday = "q"

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for duplicate keybind, got nil")
	}
}

func TestValidateAndNormalizeDuplicateDayKeybindReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Keybinds.Day.AddWork = "x"
	cfg.Keybinds.Day.AddBreak = "x"

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for duplicate day keybind, got nil")
	}
}

func TestValidateAndNormalizeDefaultHasNoDuplicates(t *testing.T) {
	cfg := DefaultAppConfig()
	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Errorf("ValidateAndNormalize() on DefaultAppConfig returned error: %v", err)
	}
}

func TestValidateAndNormalizeWorkspaceEmptyNameReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{{Name: "", StoragePath: "~/.journal/a"}}

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for empty workspace name, got nil")
	}
}

func TestValidateAndNormalizeWorkspaceWhitespaceNameReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{{Name: " Work", StoragePath: "~/.journal/work"}}

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for workspace name with leading whitespace, got nil")
	}
}

func TestValidateAndNormalizeWorkspaceDuplicateNameReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work"},
		{Name: "Work", StoragePath: "~/.journal/work2"},
	}

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for duplicate workspace name, got nil")
	}
}

func TestValidateAndNormalizeWorkspaceNegativeGoalReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{{Name: "Work", StoragePath: "~/.journal/work", WeeklyHoursGoal: -1}}

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for negative workspace weekly_hours_goal, got nil")
	}
}

func TestValidateAndNormalizeWorkspaceValidConfigOK(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Personal", StoragePath: "~/.journal/personal"},
		{Name: "Work", StoragePath: "~/.journal/work", WeeklyHoursGoal: 37.5},
	}

	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Errorf("ValidateAndNormalize() unexpected error for valid workspaces: %v", err)
	}
}

func TestValidateAndNormalizeWorkspaceInvalidWorkDayReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work", WorkDays: []string{"monday", "funday"}},
	}

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for invalid workspace work_day, got nil")
	}
}

func TestValidateAndNormalizeWorkspaceWorkDaysNormalisedToLowercase(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work", WorkDays: []string{"Monday", "TUESDAY"}},
	}

	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}

	for _, d := range cfg.Workspaces[0].WorkDays {
		for _, r := range d {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("workspace WorkDays entry %q still has uppercase after ValidateAndNormalize()", d)
			}
		}
	}
}

func TestValidateAndNormalizeWorkspaceEmptyWorkDaysFallsBackToTopLevel(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work"},
	}

	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}
	if len(cfg.Workspaces[0].WorkDays) != 0 {
		t.Errorf("ValidateAndNormalize() should leave empty workspace WorkDays untouched, got %v", cfg.Workspaces[0].WorkDays)
	}
}

func TestDefaultWorkDaysIsMonToFri(t *testing.T) {
	cfg := DefaultAppConfig()
	want := []string{"monday", "tuesday", "wednesday", "thursday", "friday"}

	if len(cfg.WorkDays) != len(want) {
		t.Fatalf("WorkDays len = %d, want %d", len(cfg.WorkDays), len(want))
	}
	for i, d := range want {
		if cfg.WorkDays[i] != d {
			t.Errorf("WorkDays[%d] = %q, want %q", i, cfg.WorkDays[i], d)
		}
	}
}

func TestIsWorkDayDefaultConfig(t *testing.T) {
	cfg := DefaultAppConfig()

	monday := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	if !cfg.IsWorkDay(monday) {
		t.Error("IsWorkDay(monday) = false, want true")
	}

	saturday := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)
	if cfg.IsWorkDay(saturday) {
		t.Error("IsWorkDay(saturday) = true, want false")
	}

	sunday := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	if cfg.IsWorkDay(sunday) {
		t.Error("IsWorkDay(sunday) = true, want false")
	}
}

func TestIsWorkDayCustomConfig(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.WorkDays = []string{"saturday", "sunday"}

	saturday := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)
	if !cfg.IsWorkDay(saturday) {
		t.Error("IsWorkDay(saturday) = false with custom work_days, want true")
	}

	monday := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	if cfg.IsWorkDay(monday) {
		t.Error("IsWorkDay(monday) = true with custom work_days, want false")
	}
}

func TestValidateAndNormalizeEmptyWorkDaysFillsDefault(t *testing.T) {
	cfg := AppConfig{WeeklyHoursGoal: 40}
	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}
	if len(cfg.WorkDays) == 0 {
		t.Error("ValidateAndNormalize() left WorkDays empty, want defaults filled in")
	}
}

func TestValidateAndNormalizeInvalidWorkDayReturnsError(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.WorkDays = []string{"monday", "funday"}

	if err := cfg.ValidateAndNormalize(); err == nil {
		t.Error("ValidateAndNormalize() expected error for invalid work_day, got nil")
	}
}

func TestValidateAndNormalizeWorkDaysNormalisedToLowercase(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.WorkDays = []string{"Monday", "TUESDAY", "Wednesday"}

	if err := cfg.ValidateAndNormalize(); err != nil {
		t.Fatalf("ValidateAndNormalize() error: %v", err)
	}

	for _, d := range cfg.WorkDays {
		for _, r := range d {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("WorkDays entry %q still has uppercase after ValidateAndNormalize()", d)
			}
		}
	}
}
