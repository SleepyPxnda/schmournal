package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ── Workspace config ──────────────────────────────────────────────────────────

// WorkspaceConfig holds per-workspace settings. Each workspace is an
// independent journal directory with its own storage path and working-hours
// goal. Individual fields fall back to the top-level config defaults when left
// empty / zero.
type WorkspaceConfig struct {
	Name            string   `toml:"name"`
	StoragePath     string   `toml:"storage_path"`
	WeeklyHoursGoal float64  `toml:"weekly_hours_goal"`
	WorkDays        []string `toml:"work_days"`
}

// ── Keybind structs ───────────────────────────────────────────────────────────

// ListKeybinds holds configurable keys for the main list view.
type ListKeybinds struct {
	Quit            string `toml:"quit"`
	OpenToday       string `toml:"open_today"`
	OpenDate        string `toml:"open_date"`
	Delete          string `toml:"delete"`
	Export          string `toml:"export"`
	StatsView       string `toml:"stats_view"`
	SwitchWorkspace string `toml:"switch_workspace"`
}

// DayKeybinds holds configurable keys for the day detail view.
type DayKeybinds struct {
	AddWork        string `toml:"add_work"`
	AddBreak       string `toml:"add_break"`
	Edit           string `toml:"edit"`
	Delete         string `toml:"delete"`
	SetStartNow    string `toml:"set_start_now"`
	SetStartManual string `toml:"set_start_manual"`
	SetEndNow      string `toml:"set_end_now"`
	SetEndManual   string `toml:"set_end_manual"`
	Notes          string `toml:"notes"`
	TodoOverview   string `toml:"todo_overview"`
	Export         string `toml:"export"`
	ClockStart     string `toml:"clock_start"`
	ClockStop      string `toml:"clock_stop"`
}

// Keybinds groups all view-specific keybind configurations.
type Keybinds struct {
	List ListKeybinds `toml:"list"`
	Day  DayKeybinds  `toml:"day"`
}

// ── Config ────────────────────────────────────────────────────────────────────

// Config is the top-level configuration structure.
type Config struct {
	StoragePath     string            `toml:"storage_path"`
	WeeklyHoursGoal float64           `toml:"weekly_hours_goal"`
	WorkDays        []string          `toml:"work_days"`
	Keybinds        Keybinds          `toml:"keybinds"`
	Workspaces      []WorkspaceConfig `toml:"workspaces"`
}

// IsWorkDay reports whether t falls on a configured working day.
// Day matching is case-insensitive (config values are normalised to lowercase
// during validation, so this comparison is always exact after Load).
func (cfg Config) IsWorkDay(t time.Time) bool {
	wd := strings.ToLower(t.Weekday().String())
	for _, d := range cfg.WorkDays {
		if d == wd {
			return true
		}
	}
	return false
}

// Default returns a Config populated with the application defaults.
func Default() Config {
	return Config{
		StoragePath:     "~/.journal",
		WeeklyHoursGoal: 40,
		WorkDays:        []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
		Keybinds: Keybinds{
			List: ListKeybinds{
				Quit:            "q",
				OpenToday:       "n",
				OpenDate:        "c",
				Delete:          "d",
				Export:          "x",
				StatsView:       "s",
				SwitchWorkspace: "p",
			},
			Day: DayKeybinds{
				AddWork:        "w",
				AddBreak:       "b",
				Edit:           "e",
				Delete:         "d",
				SetStartNow:    "s",
				SetStartManual: "S",
				SetEndNow:      "f",
				SetEndManual:   "F",
				Notes:          "n",
				TodoOverview:   "t",
				Export:         "x",
				ClockStart:     "c",
				ClockStop:      "c",
			},
		},
	}
}

// FilePath returns the absolute path of the config file.
func FilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "schmournal.config"), nil
}

// Load reads the config file and returns a Config. If the file does not exist a
// default config is written to disk and the defaults are returned. On parse
// errors the defaults are returned together with the error.
//
// If the file exists but is missing keys introduced in a newer version of the
// application, a one-time migration is performed: the old file is renamed to
// schmournal.old.config and a fresh schmournal.config is written using the full
// default template with the user's existing values preserved.
func Load() (Config, error) {
	cfg := Default()

	path, err := FilePath()
	if err != nil {
		return cfg, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Best-effort: write the default config for the user to edit later.
		_ = WriteDefault(path)
		return cfg, nil
	}

	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Default(), err
	}

	if err := cfg.validate(); err != nil {
		return Default(), err
	}

	// If any keys are absent, migrate to a fresh config that includes them all.
	if needsMigration(md) {
		_ = migrateConfig(path, cfg)
	}

	return cfg, nil
}

// needsMigration reports whether the decoded metadata is missing any keys that
// are present in the current default config. It derives the expected key set
// automatically from the Config struct's toml tags so that adding a new field
// never requires a manual update here.
func needsMigration(md toml.MetaData) bool {
	for _, path := range collectTOMLPaths(reflect.TypeOf(Config{}), nil) {
		if !md.IsDefined(path...) {
			return true
		}
	}
	return false
}

// collectTOMLPaths recursively walks a struct type and returns the TOML key
// path (as a []string) for every leaf field, using the "toml" struct tag as
// the path component name. Nested structs are descended into with their tag
// name prepended to the path. Slice fields are skipped because they are
// optional (TOML arrays of tables) and do not need to be present for the
// config to be considered complete.
func collectTOMLPaths(t reflect.Type, prefix []string) [][]string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var paths [][]string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("toml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		// Skip slice fields – they are optional arrays of tables.
		if f.Type.Kind() == reflect.Slice {
			continue
		}
		path := make([]string, len(prefix), len(prefix)+1)
		copy(path, prefix)
		path = append(path, tag)
		if f.Type.Kind() == reflect.Struct {
			paths = append(paths, collectTOMLPaths(f.Type, path)...)
		} else {
			paths = append(paths, path)
		}
	}
	return paths
}

// migrateConfig renames path to schmournal.old.config and writes a fresh
// schmournal.config containing the full default template with the user's values
// substituted in so that no customisation is lost.
//
// Workspaces that do not yet have a work_days override receive all seven days
// as an explicit default, preserving the pre-work_days behaviour where every
// day counted as a work day.
func migrateConfig(path string, cfg Config) error {
	// Fill empty workspace work_days with the full week so that existing
	// workspaces keep their pre-feature behaviour (all days are work days).
	allDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	for i := range cfg.Workspaces {
		if len(cfg.Workspaces[i].WorkDays) == 0 {
			cfg.Workspaces[i].WorkDays = allDays
		}
	}
	oldPath := strings.TrimSuffix(path, ".config") + ".old.config"
	if err := os.Rename(path, oldPath); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(generateConfigContent(cfg)), 0o644)
}

// WriteDefault writes a commented default config file to path, creating
// intermediate directories as needed.
func WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(generateConfigContent(Default())), 0o644)
}

// generateConfigContent returns the full commented config file content with
// values taken from cfg. This is used both when writing a brand-new default
// config and when migrating an existing config that is missing newer keys.
func generateConfigContent(cfg Config) string {
	workDayParts := make([]string, len(cfg.WorkDays))
	for i, d := range cfg.WorkDays {
		workDayParts[i] = fmt.Sprintf("%q", d)
	}
	workDaysStr := "[" + strings.Join(workDayParts, ", ") + "]"

	return fmt.Sprintf(`# Schmournal Configuration
# Location: ~/.config/schmournal.config

# Directory where journal JSON files are stored.
# The ~ is expanded to your home directory.
storage_path = %q

# Default weekly working hours goal used in the stats bar progress meter.
weekly_hours_goal = %g

# Days of the week that count as working days.
# Non-working days are skipped when calculating your streak (so a weekend
# never breaks it) and are highlighted in the week bar.
# Logging on a non-working day is always allowed.
# Valid values (case-insensitive): monday, tuesday, wednesday, thursday,
#   friday, saturday, sunday
work_days = %s

# ── Workspaces ────────────────────────────────────────────────────────────────
# Workspaces let you maintain separate journal directories with independent
# settings. When defined you can switch between them from the list view.
# Press the switch_workspace key (default: p) to open the picker.
#
# Each [[workspaces]] entry may override storage_path, weekly_hours_goal, and
# work_days. Omitted fields fall back to the top-level defaults above.
#
# Example (uncomment and edit to enable):
#
# [[workspaces]]
# name             = "Personal"
# storage_path     = "~/.journal/personal"
# weekly_hours_goal = 40
# work_days        = ["monday", "tuesday", "wednesday", "thursday", "friday"]
#
# [[workspaces]]
# name             = "Work"
# storage_path     = "~/.journal/work"
# weekly_hours_goal = 37.5
# work_days        = ["monday", "tuesday", "wednesday", "thursday", "friday"]
%s
# ── Keybinds ──────────────────────────────────────────────────────────────────
# Each value is a single key string as understood by the terminal
# (e.g. "q", "x", "ctrl+s").  Arrow keys, Enter, Esc and Tab are not
# configurable here — they always keep their default role.

[keybinds.list]
quit             = %q   # Quit the application
open_today       = %q   # Open / create today's entry
open_date        = %q   # Open / create an entry for a specific date
delete           = %q   # Delete the selected day record
export           = %q   # Export the selected day to Markdown
stats_view       = %q   # Open the stats overview
switch_workspace = %q   # Open the workspace picker

[keybinds.day]
add_work        = %q   # Add a new work entry
add_break       = %q   # Add a new break entry
edit            = %q   # Edit selected entry (or open notes when none selected)
delete          = %q   # Delete selected entry (or the whole day when none selected)
set_start_now   = %q   # Set start time to now
set_start_manual = %q  # Set start time manually
set_end_now     = %q   # Set end time to now
set_end_manual  = %q   # Set end time manually
notes           = %q   # Open the notes editor
todo_overview   = %q   # Open the cross-day TODO overview
export          = %q   # Export day to Markdown
clock_start     = %q   # Start the clock timer (Clocking tab)
clock_stop      = %q   # Stop the clock and log the entry (Clocking tab)

`,
		cfg.StoragePath,
		cfg.WeeklyHoursGoal,
		workDaysStr,
		generateWorkspacesTOML(cfg.Workspaces),
		cfg.Keybinds.List.Quit,
		cfg.Keybinds.List.OpenToday,
		cfg.Keybinds.List.OpenDate,
		cfg.Keybinds.List.Delete,
		cfg.Keybinds.List.Export,
		cfg.Keybinds.List.StatsView,
		cfg.Keybinds.List.SwitchWorkspace,
		cfg.Keybinds.Day.AddWork,
		cfg.Keybinds.Day.AddBreak,
		cfg.Keybinds.Day.Edit,
		cfg.Keybinds.Day.Delete,
		cfg.Keybinds.Day.SetStartNow,
		cfg.Keybinds.Day.SetStartManual,
		cfg.Keybinds.Day.SetEndNow,
		cfg.Keybinds.Day.SetEndManual,
		cfg.Keybinds.Day.Notes,
		cfg.Keybinds.Day.TodoOverview,
		cfg.Keybinds.Day.Export,
		cfg.Keybinds.Day.ClockStart,
		cfg.Keybinds.Day.ClockStop,
	)
}

// generateWorkspacesTOML serialises cfg.Workspaces as TOML array-of-tables.
// Returns an empty string when there are no workspaces so the placeholder in
// the config template produces no output.
func generateWorkspacesTOML(workspaces []WorkspaceConfig) string {
	if len(workspaces) == 0 {
		return ""
	}
	var b strings.Builder
	for _, ws := range workspaces {
		b.WriteString("\n[[workspaces]]\n")
		b.WriteString(fmt.Sprintf("name             = %q\n", ws.Name))
		if ws.StoragePath != "" {
			b.WriteString(fmt.Sprintf("storage_path     = %q\n", ws.StoragePath))
		}
		if ws.WeeklyHoursGoal > 0 {
			b.WriteString(fmt.Sprintf("weekly_hours_goal = %g\n", ws.WeeklyHoursGoal))
		}
		if len(ws.WorkDays) > 0 {
			parts := make([]string, len(ws.WorkDays))
			for i, d := range ws.WorkDays {
				parts[i] = fmt.Sprintf("%q", d)
			}
			b.WriteString(fmt.Sprintf("work_days        = [%s]\n", strings.Join(parts, ", ")))
		}
	}
	return b.String()
}

// ExpandPath expands a leading ~ to the user's home directory.
// It trims any leading slash after the ~ so that filepath.Join does not
// interpret the remainder as an absolute path (e.g. "~/.journal" → "<home>/.journal").
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	rest := strings.TrimPrefix(path[1:], "/")
	if rest == "" {
		return home, nil
	}
	return filepath.Join(home, rest), nil
}

// validate fills empty keybind fields with their defaults and returns an error
// if any view contains duplicate keybind values (which would make some actions
// unreachable).
func (cfg *Config) validate() error {
	def := Default()

	if cfg.WeeklyHoursGoal <= 0 {
		cfg.WeeklyHoursGoal = def.WeeklyHoursGoal
	}

	// Validate and normalise work_days.
	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true,
		"thursday": true, "friday": true, "saturday": true, "sunday": true,
	}
	if len(cfg.WorkDays) == 0 {
		cfg.WorkDays = def.WorkDays
	} else {
		for i, d := range cfg.WorkDays {
			lower := strings.ToLower(d)
			if !validDays[lower] {
				return fmt.Errorf("config: invalid work_day %q (must be a day-of-the-week name, e.g. \"monday\")", d)
			}
			cfg.WorkDays[i] = lower
		}
	}

	fill := func(s *string, d string) {
		if *s == "" {
			*s = d
		}
	}

	lk := &cfg.Keybinds.List
	dl := def.Keybinds.List
	fill(&lk.Quit, dl.Quit)
	fill(&lk.OpenToday, dl.OpenToday)
	fill(&lk.OpenDate, dl.OpenDate)
	fill(&lk.Delete, dl.Delete)
	fill(&lk.Export, dl.Export)
	fill(&lk.StatsView, dl.StatsView)
	fill(&lk.SwitchWorkspace, dl.SwitchWorkspace)

	dk := &cfg.Keybinds.Day
	dd := def.Keybinds.Day
	fill(&dk.AddWork, dd.AddWork)
	fill(&dk.AddBreak, dd.AddBreak)
	fill(&dk.Edit, dd.Edit)
	fill(&dk.Delete, dd.Delete)
	fill(&dk.SetStartNow, dd.SetStartNow)
	fill(&dk.SetStartManual, dd.SetStartManual)
	fill(&dk.SetEndNow, dd.SetEndNow)
	fill(&dk.SetEndManual, dd.SetEndManual)
	fill(&dk.Notes, dd.Notes)
	fill(&dk.TodoOverview, dd.TodoOverview)
	fill(&dk.Export, dd.Export)
	fill(&dk.ClockStart, dd.ClockStart)
	fill(&dk.ClockStop, dd.ClockStop)

	if err := checkDuplicates("list", lk.Quit, lk.OpenToday, lk.OpenDate, lk.Delete, lk.Export, lk.StatsView, lk.SwitchWorkspace); err != nil {
		return err
	}
	dayKeys := []string{
		dk.AddWork, dk.AddBreak, dk.Edit, dk.Delete,
		dk.SetStartNow, dk.SetStartManual, dk.SetEndNow, dk.SetEndManual,
		dk.Notes, dk.TodoOverview, dk.Export, dk.ClockStart,
	}
	// Allow clock start/stop to intentionally share one key (toggle behavior).
	if dk.ClockStop != dk.ClockStart {
		dayKeys = append(dayKeys, dk.ClockStop)
	}
	if err := checkDuplicates("day", dayKeys...); err != nil {
		return err
	}
	// Validate workspace names: non-empty, no surrounding whitespace, unique.
	seen := make(map[string]struct{}, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		name := strings.TrimSpace(ws.Name)
		if name == "" {
			return fmt.Errorf("config: workspace at index %d has an empty name", i)
		}
		if name != ws.Name {
			return fmt.Errorf("config: workspace name %q has leading/trailing whitespace", ws.Name)
		}
		if _, dup := seen[name]; dup {
			return fmt.Errorf("config: duplicate workspace name %q", name)
		}
		seen[name] = struct{}{}
		if ws.WeeklyHoursGoal < 0 {
			return fmt.Errorf("config: workspace %q has a negative weekly_hours_goal", name)
		}
		// Validate and normalise per-workspace work_days (empty = inherit top-level).
		for j, d := range ws.WorkDays {
			lower := strings.ToLower(d)
			if !validDays[lower] {
				return fmt.Errorf("config: workspace %q has invalid work_day %q (must be a day-of-the-week name, e.g. \"monday\")", name, d)
			}
			cfg.Workspaces[i].WorkDays[j] = lower
		}
	}
	return nil
}

// checkDuplicates returns an error if any two values in keys are equal.
func checkDuplicates(view string, keys ...string) error {
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, dup := seen[k]; dup {
			return fmt.Errorf("config: duplicate keybind %q in [keybinds.%s]", k, view)
		}
		seen[k] = struct{}{}
	}
	return nil
}

// ── Default config file content ───────────────────────────────────────────────
