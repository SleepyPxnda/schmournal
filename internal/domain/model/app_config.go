package model

import (
	"fmt"
	"strings"
	"time"
)

// WorkspaceConfig holds per-workspace settings.
type WorkspaceConfig struct {
	Name            string   `toml:"name"`
	StoragePath     string   `toml:"storage_path"`
	WeeklyHoursGoal float64  `toml:"weekly_hours_goal"`
	WorkDays        []string `toml:"work_days"`
}

// ListKeybinds holds configurable keys for the main list view.
type ListKeybinds struct {
	Quit            string `toml:"quit"`
	OpenToday       string `toml:"open_today"`
	OpenDate        string `toml:"open_date"`
	Delete          string `toml:"delete"`
	Export          string `toml:"export"`
	WeekView        string `toml:"week_view"`
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

// Modules holds feature-toggle settings for optional UI modules.
type Modules struct {
	ClockEnabled bool `toml:"clock_enabled"`
	TodoEnabled  bool `toml:"todo_enabled"`
}

// AppConfig is the top-level application configuration.
type AppConfig struct {
	Modules    Modules           `toml:"modules"`
	Keybinds   Keybinds          `toml:"keybinds"`
	Workspaces []WorkspaceConfig `toml:"workspaces"`
}

// DefaultWorkspaceConfig returns default settings for a workspace.
func DefaultWorkspaceConfig(name string) WorkspaceConfig {
	if name == "" {
		name = "default"
	}
	return WorkspaceConfig{
		Name:            name,
		StoragePath:     "~/.journal",
		WeeklyHoursGoal: 40,
		WorkDays:        []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
	}
}

// DefaultAppConfig returns the default configuration.
func DefaultAppConfig() AppConfig {
	return AppConfig{
		Modules: Modules{
			ClockEnabled: true,
			TodoEnabled:  true,
		},
		Workspaces: []WorkspaceConfig{
			DefaultWorkspaceConfig("default"),
		},
		Keybinds: Keybinds{
			List: ListKeybinds{
				Quit:            "q",
				OpenToday:       "n",
				OpenDate:        "c",
				Delete:          "d",
				Export:          "x",
				WeekView:        "v",
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

// IsWorkDay reports whether t falls on a configured working day.
func (cfg AppConfig) IsWorkDay(t time.Time) bool {
	workDays := DefaultWorkspaceConfig("").WorkDays
	if len(cfg.Workspaces) > 0 && len(cfg.Workspaces[0].WorkDays) > 0 {
		workDays = cfg.Workspaces[0].WorkDays
	}
	wd := strings.ToLower(t.Weekday().String())
	for _, d := range workDays {
		if d == wd {
			return true
		}
	}
	return false
}

// ValidateAndNormalize validates configuration and applies defaults.
func (cfg *AppConfig) ValidateAndNormalize() error {
	def := DefaultAppConfig()
	workspaceDef := def.Workspaces[0]

	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true,
		"thursday": true, "friday": true, "saturday": true, "sunday": true,
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
	fill(&lk.WeekView, dl.WeekView)
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

	if err := checkDuplicates(lk.Quit, lk.OpenToday, lk.OpenDate, lk.Delete, lk.Export, lk.WeekView, lk.StatsView, lk.SwitchWorkspace); err != nil {
		return err
	}
	dayKeys := []string{
		dk.AddWork, dk.AddBreak, dk.Edit, dk.Delete,
		dk.SetStartNow, dk.SetStartManual, dk.SetEndNow, dk.SetEndManual,
		dk.Notes, dk.TodoOverview, dk.Export, dk.ClockStart,
	}
	if dk.ClockStop != dk.ClockStart {
		dayKeys = append(dayKeys, dk.ClockStop)
	}
	if err := checkDuplicates(dayKeys...); err != nil {
		return err
	}

	if len(cfg.Workspaces) == 0 {
		cfg.Workspaces = []WorkspaceConfig{DefaultWorkspaceConfig("default")}
	}

	seen := make(map[string]struct{}, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		name := strings.TrimSpace(ws.Name)
		if name == "" {
			return fmt.Errorf("workspace at index %d has an empty name", i)
		}
		if name != ws.Name {
			return fmt.Errorf("workspace name %q has leading/trailing whitespace", ws.Name)
		}
		if _, dup := seen[name]; dup {
			return fmt.Errorf("duplicate workspace name %q", name)
		}
		seen[name] = struct{}{}
		if ws.StoragePath == "" {
			cfg.Workspaces[i].StoragePath = workspaceDef.StoragePath
		}
		if ws.WeeklyHoursGoal < 0 {
			return fmt.Errorf("workspace %q has a negative weekly_hours_goal", name)
		}
		if ws.WeeklyHoursGoal == 0 {
			cfg.Workspaces[i].WeeklyHoursGoal = workspaceDef.WeeklyHoursGoal
		}
		if len(ws.WorkDays) == 0 {
			cfg.Workspaces[i].WorkDays = append([]string(nil), workspaceDef.WorkDays...)
		} else {
			for j, d := range ws.WorkDays {
				lower := strings.ToLower(d)
				if !validDays[lower] {
					return fmt.Errorf("workspace %q has invalid work_day %q", name, d)
				}
				cfg.Workspaces[i].WorkDays[j] = lower
			}
		}
	}

	return nil
}

func checkDuplicates(keys ...string) error {
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, dup := seen[k]; dup {
			return fmt.Errorf("duplicate keybind %q", k)
		}
		seen[k] = struct{}{}
	}
	return nil
}
