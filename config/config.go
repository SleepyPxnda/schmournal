package config

import (
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
	PrevWeek string `toml:"prev_week"`
	NextWeek string `toml:"next_week"`
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
	StoragePath string   `toml:"storage_path"`
	Keybinds    Keybinds `toml:"keybinds"`
}

// Default returns a Config populated with the application defaults.
func Default() Config {
	return Config{
		StoragePath: "~/.journal",
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
				PrevWeek: "h",
				NextWeek: "l",
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
func Load() (Config, error) {
	cfg := Default()

	path, err := FilePath()
	if err != nil {
		return cfg, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Best-effort: write the default config for the user to edit later.
		_ = WriteDefault(path)
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
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
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}

// ── Default config file content ───────────────────────────────────────────────

const defaultConfigContent = `# Schmournal Configuration
# Location: ~/.config/schmournal.config

# Directory where journal JSON files are stored.
# The ~ is expanded to your home directory.
storage_path = "~/.journal"

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
prev_week = "h"   # Go to the previous week (also ←)
next_week = "l"   # Go to the next week  (also →)
`
