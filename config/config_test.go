package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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
	checkNotEmpty("list.export", lk.Export)
	checkNotEmpty("list.week_view", lk.WeekView)
	checkNotEmpty("list.stats_view", lk.StatsView)
	checkNotEmpty("list.todo_overview", lk.TodoOverview)

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
	checkNotEmpty("day.export", dk.Export)

	wk := cfg.Keybinds.Week
	checkNotEmpty("week.prev_week", wk.PrevWeek)
	checkNotEmpty("week.next_week", wk.NextWeek)
	checkNotEmpty("week.set_weekly_hours", wk.SetWeeklyHours)
	checkNotEmpty("week.todo_overview", wk.TodoOverview)
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
	if cfg.Keybinds.Week.TodoOverview != def.Keybinds.Week.TodoOverview {
		t.Errorf("Week.TodoOverview = %q, want %q", cfg.Keybinds.Week.TodoOverview, def.Keybinds.Week.TodoOverview)
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
		{"keybinds", "list", "todo_overview"},
		{"keybinds", "day", "add_work"},
		{"keybinds", "day", "todo_overview"},
		{"keybinds", "week", "prev_week"},
		{"keybinds", "week", "todo_overview"},
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

func TestValidateWorkspaceInvalidWorkDayReturnsError(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work", WorkDays: []string{"monday", "funday"}},
	}
	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for invalid workspace work_day, got nil")
	}
}

func TestValidateWorkspaceWorkDaysNormalisedToLowercase(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work", WorkDays: []string{"Monday", "TUESDAY"}},
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	for _, d := range cfg.Workspaces[0].WorkDays {
		for _, r := range d {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("workspace WorkDays entry %q still has uppercase after validate()", d)
			}
		}
	}
}

func TestValidateWorkspaceEmptyWorkDaysFallsBackToTopLevel(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = []WorkspaceConfig{
		{Name: "Work", StoragePath: "~/.journal/work"},
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	// Empty workspace WorkDays is valid — it means "inherit from top-level config".
	if len(cfg.Workspaces[0].WorkDays) != 0 {
		t.Errorf("validate() should leave empty workspace WorkDays untouched, got %v", cfg.Workspaces[0].WorkDays)
	}
}

func TestMigrateConfigFillsWorkspaceWorkDays(t *testing.T) {
	home := withTempHome(t)
	cfgPath := filepath.Join(home, ".config", "schmournal.config")

	// Write a config that has a workspace but no work_days for it.
	raw := `storage_path = "~/.journal"
weekly_hours_goal = 40.0
work_days = ["monday", "tuesday", "wednesday", "thursday", "friday"]

[[workspaces]]
name         = "Work"
storage_path = "~/.journal/work"
`
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// After migration the workspace should have work_days = all 7 days.
	if len(cfg.Workspaces) == 0 {
		t.Fatal("Load() lost workspace definitions during migration")
	}
	ws := cfg.Workspaces[0]
	if len(ws.WorkDays) == 0 {
		t.Error("migrated workspace has empty WorkDays, want all 7 days")
	}
	allDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	if len(ws.WorkDays) != len(allDays) {
		t.Errorf("migrated workspace WorkDays = %v, want %v", ws.WorkDays, allDays)
	}
}

func TestGenerateWorkspacesTOMLEmpty(t *testing.T) {
	if got := generateWorkspacesTOML(nil); got != "" {
		t.Errorf("generateWorkspacesTOML(nil) = %q, want empty string", got)
	}
}

func TestGenerateWorkspacesTOMLWithWorkDays(t *testing.T) {
	ws := []WorkspaceConfig{
		{
			Name:        "Work",
			StoragePath: "~/.journal/work",
			WorkDays:    []string{"monday", "tuesday"},
		},
	}
	got := generateWorkspacesTOML(ws)
	if !strings.Contains(got, `[[workspaces]]`) {
		t.Error("generateWorkspacesTOML missing [[workspaces]] header")
	}
	if !strings.Contains(got, `"monday"`) {
		t.Error("generateWorkspacesTOML missing work_days monday")
	}
}

// ── WorkDays / IsWorkDay ──────────────────────────────────────────────────────

func TestDefaultWorkDaysIsMonToFri(t *testing.T) {
	cfg := Default()
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
	cfg := Default()
	// Monday (weekday) should be a work day.
	monday := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC) // Monday
	if !cfg.IsWorkDay(monday) {
		t.Error("IsWorkDay(monday) = false, want true")
	}
	// Saturday should not be a work day with default config.
	saturday := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC) // Saturday
	if cfg.IsWorkDay(saturday) {
		t.Error("IsWorkDay(saturday) = true, want false")
	}
	// Sunday should not be a work day with default config.
	sunday := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC) // Sunday
	if cfg.IsWorkDay(sunday) {
		t.Error("IsWorkDay(sunday) = true, want false")
	}
}

func TestIsWorkDayCustomConfig(t *testing.T) {
	cfg := Default()
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

func TestValidateEmptyWorkDaysFillsDefault(t *testing.T) {
	cfg := Config{WeeklyHoursGoal: 40}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	if len(cfg.WorkDays) == 0 {
		t.Error("validate() left WorkDays empty, want defaults filled in")
	}
}

func TestValidateInvalidWorkDayReturnsError(t *testing.T) {
	cfg := Default()
	cfg.WorkDays = []string{"monday", "funday"}
	if err := cfg.validate(); err == nil {
		t.Error("validate() expected error for invalid work_day, got nil")
	}
}

func TestValidateWorkDaysNormalisedToLowercase(t *testing.T) {
	cfg := Default()
	cfg.WorkDays = []string{"Monday", "TUESDAY", "Wednesday"}
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	for _, d := range cfg.WorkDays {
		for _, r := range d {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("WorkDays entry %q still has uppercase after validate()", d)
			}
		}
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
