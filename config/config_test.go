package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ── Default ───────────────────────────────────────────────────────────────────

func TestDefaultWeeklyHoursGoal(t *testing.T) {
	cfg := Default()
	if cfg.WeeklyHoursGoal != 40 {
		t.Errorf("WeeklyHoursGoal = %g, want 40", cfg.WeeklyHoursGoal)
	}
}

func TestDefaultStoragePath(t *testing.T) {
	cfg := Default()
	if cfg.StoragePath != "~/.journal" {
		t.Errorf("StoragePath = %q, want %q", cfg.StoragePath, "~/.journal")
	}
}

func TestDefaultKeybindsNotEmpty(t *testing.T) {
	cfg := Default()

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
	checkNotEmpty("list.add_work", lk.AddWork)
	checkNotEmpty("list.add_break", lk.AddBreak)
	checkNotEmpty("list.export", lk.Export)
	checkNotEmpty("list.week_view", lk.WeekView)
	checkNotEmpty("list.stats_view", lk.StatsView)

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
	checkNotEmpty("day.export", dk.Export)

	wk := cfg.Keybinds.Week
	checkNotEmpty("week.prev_week", wk.PrevWeek)
	checkNotEmpty("week.next_week", wk.NextWeek)
	checkNotEmpty("week.set_weekly_hours", wk.SetWeeklyHours)
}

// ── validate ──────────────────────────────────────────────────────────────────

func TestValidateFillsEmptyKeybinds(t *testing.T) {
	cfg := Config{WeeklyHoursGoal: 40}
	// All keybind fields start empty; validate should fill them with defaults.
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	def := Default()
	if cfg.Keybinds.List.Quit != def.Keybinds.List.Quit {
		t.Errorf("List.Quit = %q, want %q", cfg.Keybinds.List.Quit, def.Keybinds.List.Quit)
	}
	if cfg.Keybinds.Day.AddWork != def.Keybinds.Day.AddWork {
		t.Errorf("Day.AddWork = %q, want %q", cfg.Keybinds.Day.AddWork, def.Keybinds.Day.AddWork)
	}
	if cfg.Keybinds.Week.PrevWeek != def.Keybinds.Week.PrevWeek {
		t.Errorf("Week.PrevWeek = %q, want %q", cfg.Keybinds.Week.PrevWeek, def.Keybinds.Week.PrevWeek)
	}
}

func TestValidateZeroWeeklyHoursGoalReset(t *testing.T) {
	cfg := Config{WeeklyHoursGoal: 0}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	if cfg.WeeklyHoursGoal <= 0 {
		t.Errorf("WeeklyHoursGoal = %g after validate, want > 0", cfg.WeeklyHoursGoal)
	}
}

func TestValidateDuplicateListKeybindReturnsError(t *testing.T) {
	cfg := Default()
	// Set two list keys to the same value to trigger duplicate detection.
	cfg.Keybinds.List.Quit = "q"
	cfg.Keybinds.List.OpenToday = "q"

	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for duplicate keybind, got nil")
	}
}

func TestValidateDuplicateDayKeybindReturnsError(t *testing.T) {
	cfg := Default()
	cfg.Keybinds.Day.AddWork = "x"
	cfg.Keybinds.Day.Export = "x"

	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for duplicate day keybind, got nil")
	}
}

func TestValidateDefaultHasNoDuplicates(t *testing.T) {
	cfg := Default()
	if err := cfg.validate(); err != nil {
		t.Errorf("validate() on Default config returned error: %v", err)
	}
}

// ── collectTOMLPaths ──────────────────────────────────────────────────────────

func TestCollectTOMLPathsCoversAllLeafs(t *testing.T) {
	paths := collectTOMLPaths(reflect.TypeOf(Config{}), nil)
	if len(paths) == 0 {
		t.Fatal("collectTOMLPaths returned no paths")
	}
	// Every path must be non-empty.
	for _, p := range paths {
		if len(p) == 0 {
			t.Error("collectTOMLPaths returned an empty path slice")
		}
	}
}

func TestCollectTOMLPathsIncludesKnownPaths(t *testing.T) {
	paths := collectTOMLPaths(reflect.TypeOf(Config{}), nil)

	want := [][]string{
		{"storage_path"},
		{"weekly_hours_goal"},
		{"keybinds", "list", "quit"},
		{"keybinds", "day", "add_work"},
		{"keybinds", "week", "prev_week"},
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[joinPath(p)] = true
	}

	for _, w := range want {
		if !pathSet[joinPath(w)] {
			t.Errorf("collectTOMLPaths missing expected path %v", w)
		}
	}
}

// ── ExpandPath ────────────────────────────────────────────────────────────────

func TestExpandPathNoTilde(t *testing.T) {
	got, err := ExpandPath("/absolute/path")
	if err != nil {
		t.Fatalf("ExpandPath error: %v", err)
	}
	if got != "/absolute/path" {
		t.Errorf("ExpandPath = %q, want %q", got, "/absolute/path")
	}
}

func TestExpandPathTilde(t *testing.T) {
	got, err := ExpandPath("~/docs")
	if err != nil {
		t.Fatalf("ExpandPath error: %v", err)
	}
	if got == "~/docs" {
		t.Error("ExpandPath did not expand tilde")
	}
	if len(got) == 0 {
		t.Error("ExpandPath returned empty string")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// joinPath joins a TOML path slice into a dot-separated string for set lookup.
func joinPath(p []string) string {
	s := ""
	for i, part := range p {
		if i > 0 {
			s += "."
		}
		s += part
	}
	return s
}

// ── validate workspaces ───────────────────────────────────────────────────────

func TestValidateWorkspaceEmptyNameReturnsError(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{{Name: "", StoragePath: "~/.journal/a"}}
	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for empty workspace name, got nil")
	}
}

func TestValidateWorkspaceWhitespaceNameReturnsError(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{{Name: " Work", StoragePath: "~/.journal/work"}}
	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for workspace name with leading whitespace, got nil")
	}
}

func TestValidateWorkspaceDuplicateNameReturnsError(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work"},
		{Name: "Work", StoragePath: "~/.journal/work2"},
	}
	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for duplicate workspace name, got nil")
	}
}

func TestValidateWorkspaceNegativeGoalReturnsError(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{{Name: "Work", StoragePath: "~/.journal/work", WeeklyHoursGoal: -1}}
	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for negative workspace weekly_hours_goal, got nil")
	}
}

func TestValidateWorkspaceValidConfigOK(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Personal", StoragePath: "~/.journal/personal"},
		{Name: "Work", StoragePath: "~/.journal/work", WeeklyHoursGoal: 37.5},
	}
	if err := cfg.validate(); err != nil {
		t.Errorf("validate() unexpected error for valid workspaces: %v", err)
	}
}

// ── LoadState / SaveState ─────────────────────────────────────────────────────

func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	return dir
}

func TestLoadStateMissingFileReturnsEmpty(t *testing.T) {
	withTempHome(t)
	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() unexpected error: %v", err)
	}
	if s.ActiveWorkspace != "" {
		t.Errorf("LoadState() ActiveWorkspace = %q, want empty", s.ActiveWorkspace)
	}
}

func TestSaveStateCreatesFile(t *testing.T) {
	home := withTempHome(t)
	s := AppState{ActiveWorkspace: "Work"}
	if err := SaveState(s); err != nil {
		t.Fatalf("SaveState() error: %v", err)
	}
	stateFile := filepath.Join(home, ".config", stateFileName)
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("SaveState() did not create file: %v", err)
	}
}

func TestLoadStateRoundTrip(t *testing.T) {
	withTempHome(t)
	want := AppState{ActiveWorkspace: "Personal"}
	if err := SaveState(want); err != nil {
		t.Fatalf("SaveState() error: %v", err)
	}
	got, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error: %v", err)
	}
	if got != want {
		t.Errorf("LoadState() = %+v, want %+v", got, want)
	}
}
