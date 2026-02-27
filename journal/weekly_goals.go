package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const weeklyGoalsFile = "weekly_goals.json"

// WeeklyGoals maps a week key (Monday date "YYYY-MM-DD") to a custom hours
// goal for that week. It is loaded from and persisted to weekly_goals.json in
// the journal directory.
type WeeklyGoals map[string]float64

// LoadWeeklyGoals reads per-week hour goal overrides from the journal
// directory. If the file does not exist an empty map is returned without error.
func LoadWeeklyGoals() (WeeklyGoals, error) {
	dir, err := Dir()
	if err != nil {
		return WeeklyGoals{}, err
	}
	path := filepath.Join(dir, weeklyGoalsFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return WeeklyGoals{}, nil
	}
	if err != nil {
		return WeeklyGoals{}, err
	}
	var goals WeeklyGoals
	if err := json.Unmarshal(data, &goals); err != nil {
		return WeeklyGoals{}, err
	}
	return goals, nil
}

// SaveWeeklyGoals persists per-week hour goal overrides to the journal
// directory.
func SaveWeeklyGoals(goals WeeklyGoals) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(goals, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, weeklyGoalsFile), data, 0o644)
}
