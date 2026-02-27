package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ── Keybind structs ───────────────────────────────────────────────────────────

// ListKeybinds holds configurable keys for the main list view.
type ListKeybinds struct {
	Quit      string `toml:"quit"`
	OpenToday string `toml:"open_today"`
	OpenDate  string `toml:"open_date"`
	Delete    string `toml:"delete"`
	AddWork   string `toml:"add_work"`
	AddBreak  string `toml:"add_break"`
	Export    string `toml:"export"`
	WeekView  string `toml:"week_view"`
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
	Export         string `toml:"export"`
}

// WeekKeybinds holds configurable keys for the week overview.
type WeekKeybinds struct {
	PrevWeek        string `toml:"prev_week"`
	NextWeek        string `toml:"next_week"`
	SetWeeklyHours  string `toml:"set_weekly_hours"`
}

// Keybinds groups all view-specific keybind configurations.
type Keybinds struct {
	List ListKeybinds `toml:"list"`
	Day  DayKeybinds  `toml:"day"`
	Week WeekKeybinds `toml:"week"`
}

// ── Config ────────────────────────────────────────────────────────────────────

// Config is the top-level configuration structure.
type Config struct {
	StoragePath      string   `toml:"storage_path"`
	WeeklyHoursGoal  float64  `toml:"weekly_hours_goal"`
	Keybinds         Keybinds `toml:"keybinds"`
}

// Default returns a Config populated with the application defaults.
func Default() Config {
	return Config{
		StoragePath:     "~/.journal",
		WeeklyHoursGoal: 40,
		Keybinds: Keybinds{
			List: ListKeybinds{
				Quit:      "q",
				OpenToday: "n",
				OpenDate:  "c",
				Delete:    "d",
				AddWork:   "w",
				AddBreak:  "b",
				Export:    "x",
				WeekView:  "v",
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
				Export:         "x",
			},
			Week: WeekKeybinds{
				PrevWeek:       "h",
				NextWeek:       "l",
				SetWeeklyHours: "g",
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
// application, those keys are appended to the file with their default values so
// that users can discover and customise them.
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

	// Append any keys that are new since the user's config was created.
	patchMissingKeys(path, md, cfg)

	return cfg, nil
}

// patchMissingKeys appends any config keys that are absent from the existing
// file (i.e. introduced after the file was first written) so that users can see
// and customise them. Existing content is never modified.
func patchMissingKeys(path string, md toml.MetaData, cfg Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	patched := string(data)

	if !md.IsDefined("weekly_hours_goal") {
		patched += "\n# Default weekly working hours goal used in the stats bar and weekly summary.\n" +
			"# Can be overridden per-week from the weekly summary view.\n" +
			fmt.Sprintf("weekly_hours_goal = %g\n", cfg.WeeklyHoursGoal)
	}

	if !md.IsDefined("keybinds", "week", "set_weekly_hours") {
		keyLine := fmt.Sprintf("set_weekly_hours = %q   # Set a custom hours goal for the displayed week\n", cfg.Keybinds.Week.SetWeeklyHours)
		if strings.Contains(patched, "[keybinds.week]") {
			patched = insertIntoSection(patched, "[keybinds.week]", keyLine)
		} else {
			patched += fmt.Sprintf("\n[keybinds.week]\n%s", keyLine)
		}
	}

	if patched != string(data) {
		_ = os.WriteFile(path, []byte(patched), 0o644)
	}
}

// insertIntoSection inserts text at the end of a named TOML section (just
// before the next section header, or at the end of the file when the section
// is the last one).
func insertIntoSection(content, sectionHeader, text string) string {
	lines := strings.Split(content, "\n")
	sectionIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			sectionIdx = i
			break
		}
	}
	if sectionIdx == -1 {
		return content + text
	}

	// Walk forward to find where the section ends (next header or EOF).
	insertAt := len(lines)
	for i := sectionIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && strings.HasPrefix(trimmed, "[") {
			insertAt = i
			break
		}
	}

	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:insertAt]...)
	result = append(result, strings.TrimRight(text, "\n"))
	result = append(result, lines[insertAt:]...)
	return strings.Join(result, "\n")
}

// WriteDefault writes a commented default config file to path, creating
// intermediate directories as needed.
func WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(defaultConfigContent), 0o644)
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
	fill(&lk.AddWork, dl.AddWork)
	fill(&lk.AddBreak, dl.AddBreak)
	fill(&lk.Export, dl.Export)
	fill(&lk.WeekView, dl.WeekView)

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
	fill(&dk.Export, dd.Export)

	wk := &cfg.Keybinds.Week
	dw := def.Keybinds.Week
	fill(&wk.PrevWeek, dw.PrevWeek)
	fill(&wk.NextWeek, dw.NextWeek)
	fill(&wk.SetWeeklyHours, dw.SetWeeklyHours)

	if err := checkDuplicates("list", lk.Quit, lk.OpenToday, lk.OpenDate, lk.Delete, lk.AddWork, lk.AddBreak, lk.Export, lk.WeekView); err != nil {
		return err
	}
	if err := checkDuplicates("day", dk.AddWork, dk.AddBreak, dk.Edit, dk.Delete, dk.SetStartNow, dk.SetStartManual, dk.SetEndNow, dk.SetEndManual, dk.Notes, dk.Export); err != nil {
		return err
	}
	if err := checkDuplicates("week", wk.PrevWeek, wk.NextWeek, wk.SetWeeklyHours); err != nil {
		return err
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

const defaultConfigContent = `# Schmournal Configuration
# Location: ~/.config/schmournal.config

# Directory where journal JSON files are stored.
# The ~ is expanded to your home directory.
storage_path = "~/.journal"

# Default weekly working hours goal used in the stats bar and weekly summary.
# Can be overridden per-week from the weekly summary view.
weekly_hours_goal = 40

# ── Keybinds ──────────────────────────────────────────────────────────────────
# Each value is a single key string as understood by the terminal
# (e.g. "q", "x", "ctrl+s").  Arrow keys, Enter, Esc and Tab are not
# configurable here — they always keep their default role.

[keybinds.list]
quit       = "q"   # Quit the application
open_today = "n"   # Open / create today's entry
open_date  = "c"   # Open / create an entry for a specific date
delete     = "d"   # Delete the selected day record
add_work   = "w"   # Log a work entry for today
add_break  = "b"   # Log a break entry for today
export     = "x"   # Export the selected day to Markdown
week_view  = "v"   # Open the weekly overview

[keybinds.day]
add_work        = "w"   # Add a new work entry
add_break       = "b"   # Add a new break entry
edit            = "e"   # Edit selected entry (or open notes when none selected)
delete          = "d"   # Delete selected entry (or the whole day when none selected)
set_start_now   = "s"   # Set start time to now
set_start_manual = "S"  # Set start time manually
set_end_now     = "f"   # Set end time to now
set_end_manual  = "F"   # Set end time manually
notes           = "n"   # Open the notes editor
export          = "x"   # Export day to Markdown

[keybinds.week]
prev_week        = "h"   # Go to the previous week (also ←)
next_week        = "l"   # Go to the next week  (also →)
set_weekly_hours = "g"   # Set a custom hours goal for the displayed week
`
