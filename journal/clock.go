package journal

import (
	"strings"
	"time"
)

// ClockEntries converts a clocked task into WorkEntry values ready to be
// appended to a DayRecord. The elapsed duration is split evenly across any
// comma-separated projects using the same distribution logic as the manual
// work-log form. Entries with zero minutes are dropped. If the total elapsed
// time rounds down to zero minutes the returned slice is empty.
func ClockEntries(task, projectRaw string, elapsed time.Duration) []WorkEntry {
	rawParts := strings.Split(projectRaw, ",")
	projects := make([]string, 0, len(rawParts))
	for _, p := range rawParts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			projects = append(projects, trimmed)
		}
	}
	if len(projects) == 0 {
		projects = []string{""}
	}

	totalMin := int(elapsed.Minutes())
	if totalMin <= 0 {
		return nil
	}

	base := totalMin / len(projects)
	remainder := totalMin % len(projects)

	entries := make([]WorkEntry, 0, len(projects))
	for i, proj := range projects {
		mins := base
		if i < remainder {
			mins++ // distribute remainder: one extra minute to the first N projects
		}
		if mins == 0 {
			continue
		}
		entries = append(entries, WorkEntry{
			ID:          NewID(),
			Task:        task,
			Project:     proj,
			DurationMin: mins,
			IsBreak:     false,
		})
	}
	return entries
}
