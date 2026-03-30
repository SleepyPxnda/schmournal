package service

import (
	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// EntryConsolidator consolidates work entries by task name.
// This is used for export generation to merge duplicate tasks.
//
// Design Decision: This is a stateless domain service.
// Consolidation logic belongs in domain, not infrastructure.
type EntryConsolidator struct{}

// NewEntryConsolidator creates a new EntryConsolidator.
func NewEntryConsolidator() *EntryConsolidator {
	return &EntryConsolidator{}
}

// Consolidate merges entries that share the same task name (case-sensitive),
// summing their DurationMin values.
//
// Example:
//   Input:  [WorkEntry{Task:"Review", 30m}, WorkEntry{Task:"Review", 20m}]
//   Output: [WorkEntry{Task:"Review", 50m}]
//
// Note: The first occurrence of each task is kept, subsequent ones are merged in.
func (c *EntryConsolidator) Consolidate(entries []model.WorkEntry) []model.WorkEntry {
	if len(entries) == 0 {
		return []model.WorkEntry{}
	}

	seen := make(map[string]int) // task → index in out
	var out []model.WorkEntry

	for _, e := range entries {
		if idx, ok := seen[e.Task]; ok {
			// Task already exists - sum durations
			out[idx].DurationMin += e.DurationMin
		} else {
			// First occurrence of this task
			seen[e.Task] = len(out)
			out = append(out, e)
		}
	}

	return out
}

// ConsolidateByProject consolidates entries by both project and task.
// This is useful for more detailed exports where project distinction matters.
//
// Example:
//   Input:  [WorkEntry{Project:"A", Task:"Review", 30m}, 
//            WorkEntry{Project:"A", Task:"Review", 20m},
//            WorkEntry{Project:"B", Task:"Review", 10m}]
//   Output: [WorkEntry{Project:"A", Task:"Review", 50m},
//            WorkEntry{Project:"B", Task:"Review", 10m}]
func (c *EntryConsolidator) ConsolidateByProject(entries []model.WorkEntry) []model.WorkEntry {
	if len(entries) == 0 {
		return []model.WorkEntry{}
	}

	type key struct {
		project string
		task    string
	}

	seen := make(map[key]int) // (project, task) → index in out
	var out []model.WorkEntry

	for _, e := range entries {
		k := key{project: e.Project, task: e.Task}
		if idx, ok := seen[k]; ok {
			out[idx].DurationMin += e.DurationMin
		} else {
			seen[k] = len(out)
			out = append(out, e)
		}
	}

	return out
}
