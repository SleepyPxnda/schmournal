package ui

import (
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/service"
)

type uiTimeProvider struct{}

func (uiTimeProvider) Now() time.Time {
	return time.Now()
}

func (uiTimeProvider) GenerateID() string {
	return newID()
}

var (
	uiDurationFormatter = service.NewDurationFormatter()
	uiDurationParser    = service.NewDurationParser()
	uiClockConverter    = service.NewClockConverter(uiTimeProvider{})
)

func formatDuration(d time.Duration) string {
	return uiDurationFormatter.Format(d)
}

func parseDuration(s string) (time.Duration, error) {
	return uiDurationParser.Parse(s)
}

func clockEntries(task, projectsRaw string, elapsed time.Duration) []WorkEntry {
	domainEntries := uiClockConverter.ConvertToEntries(task, projectsRaw, elapsed)
	entries := make([]WorkEntry, 0, len(domainEntries))
	for _, entry := range domainEntries {
		entries = append(entries, WorkEntry{
			ID:          entry.ID,
			Task:        entry.Task,
			Project:     entry.Project,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		})
	}
	return entries
}
