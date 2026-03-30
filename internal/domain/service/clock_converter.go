package service

import (
	"strings"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// ClockConverter converts clocked time into WorkEntry entries.
// This is a domain service that handles the business logic of distributing
// elapsed time across multiple projects.
type ClockConverter struct {
	timeProvider TimeProvider
}

// NewClockConverter creates a new ClockConverter.
func NewClockConverter(timeProvider TimeProvider) *ClockConverter {
	return &ClockConverter{
		timeProvider: timeProvider,
	}
}

// ConvertToEntries converts a clocked task into WorkEntry values.
// The elapsed duration is split evenly across any comma-separated projects
// using the same distribution logic as the manual work-log form.
//
// Rules:
// - Entries with zero minutes are dropped
// - If total elapsed time rounds down to zero minutes, returns empty slice
// - Remainder minutes are distributed to first N projects
//
// Example:
//
//	task="Code Review", projects="ProjectA, ProjectB", elapsed=91m
//	→ [WorkEntry{ProjectA, 46m}, WorkEntry{ProjectB, 45m}]
func (c *ClockConverter) ConvertToEntries(task, projectsRaw string, elapsed time.Duration) []model.WorkEntry {
	// Parse comma-separated projects
	rawParts := strings.Split(projectsRaw, ",")
	projects := make([]string, 0, len(rawParts))
	for _, p := range rawParts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			projects = append(projects, trimmed)
		}
	}
	if len(projects) == 0 {
		projects = []string{""} // No project specified
	}

	// Calculate total minutes
	totalMin := int(elapsed.Minutes())
	if totalMin <= 0 {
		return nil
	}

	// Distribute minutes across projects
	base := totalMin / len(projects)
	remainder := totalMin % len(projects)

	entries := make([]model.WorkEntry, 0, len(projects))
	for i, proj := range projects {
		mins := base
		if i < remainder {
			// Distribute remainder: one extra minute to the first N projects
			mins++
		}
		if mins == 0 {
			continue
		}
		entries = append(entries, model.WorkEntry{
			ID:          c.timeProvider.GenerateID(),
			Task:        task,
			Project:     proj,
			DurationMin: mins,
			IsBreak:     false,
		})
	}
	return entries
}
