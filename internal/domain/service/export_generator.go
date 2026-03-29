package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

// ExportGenerator generates Markdown exports from DayRecords.
// This is a domain service that contains the business logic for
// formatting work data into a human-readable report.
//
// Design Decision: Export format is a business concern (domain),
// but saving to disk is infrastructure (handled by ExportRepository).
type ExportGenerator struct {
	formatter    *DurationFormatter
	consolidator *EntryConsolidator
	timeProvider TimeProvider
}

// NewExportGenerator creates a new ExportGenerator.
func NewExportGenerator(
	formatter *DurationFormatter,
	consolidator *EntryConsolidator,
	timeProvider TimeProvider,
) *ExportGenerator {
	return &ExportGenerator{
		formatter:    formatter,
		consolidator: consolidator,
		timeProvider: timeProvider,
	}
}

// GenerateMarkdown builds a clean, self-contained Markdown report for one DayRecord.
// Work items are grouped by project, duplicates are consolidated, breaks are merged.
func (g *ExportGenerator) GenerateMarkdown(rec model.DayRecord) string {
	// Separate work and breaks
	var rawWork, rawBreaks []model.WorkEntry
	for _, e := range rec.Entries {
		if e.IsBreak {
			rawBreaks = append(rawBreaks, e)
		} else {
			rawWork = append(rawWork, e)
		}
	}

	// Consolidate breaks by task name
	breakItems := g.consolidator.Consolidate(rawBreaks)

	// Calculate totals
	var workTotal, breakTotal time.Duration
	for _, e := range rawWork {
		workTotal += e.Duration()
	}
	for _, e := range breakItems {
		breakTotal += e.Duration()
	}

	// Format times
	startTime := rec.StartTime
	if startTime == "" {
		startTime = "—"
	}
	endTime := rec.EndTime
	if endTime == "" {
		endTime = "—"
	}
	dayDur, hasDayDur := rec.DayDuration()

	// Format date
	t, err := rec.ParseDate()
	dateLabel := rec.Date
	if err == nil {
		dateLabel = t.Format("Monday, January 2, 2006")
	}

	var b strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	fmt.Fprintf(&b, "# Daily Work Report — %s\n\n", dateLabel)

	// ── Work Day ──────────────────────────────────────────────────────────────
	b.WriteString("## 🕐 Work Day\n\n")
	b.WriteString("| | |\n|:---|:---|\n")
	fmt.Fprintf(&b, "| **Start** | %s |\n", startTime)
	fmt.Fprintf(&b, "| **End** | %s |\n", endTime)
	if hasDayDur {
		fmt.Fprintf(&b, "| **Day duration** | %s |\n", g.formatter.Format(dayDur))
	}
	b.WriteString("\n")

	// ── Work Items grouped by project ─────────────────────────────────────────
	b.WriteString("## 📋 Work Items\n\n")
	if len(rawWork) == 0 {
		b.WriteString("_No work items logged._\n\n")
	} else {
		g.writeWorkItems(&b, rawWork, workTotal)
	}

	// ── Breaks ────────────────────────────────────────────────────────────────
	b.WriteString("## ☕ Breaks\n\n")
	if len(breakItems) == 0 {
		b.WriteString("_No breaks logged._\n\n")
	} else {
		b.WriteString("| Break | Duration |\n|:------|:--------|\n")
		for _, e := range breakItems {
			fmt.Fprintf(&b, "| %s | %s |\n", e.Task, g.formatter.Format(e.Duration()))
		}
		fmt.Fprintf(&b, "\n**Total breaks: %s**\n\n", g.durOrDash(breakTotal))
	}

	// ── Summary ───────────────────────────────────────────────────────────────
	b.WriteString("## 📊 Summary\n\n")
	b.WriteString("| | |\n|:---|:---|\n")
	fmt.Fprintf(&b, "| **Work** | %s |\n", g.durOrDash(workTotal))
	fmt.Fprintf(&b, "| **Breaks** | %s |\n", g.durOrDash(breakTotal))
	if workTotal+breakTotal > 0 {
		fmt.Fprintf(&b, "| **Total logged** | %s |\n", g.formatter.Format(workTotal+breakTotal))
	}
	if hasDayDur {
		fmt.Fprintf(&b, "| **Day duration** | %s |\n", g.formatter.Format(dayDur))
	}
	b.WriteString("\n")

	// ── Notes ─────────────────────────────────────────────────────────────────
	if rec.Notes != "" {
		b.WriteString("## 📝 Notes\n\n")
		b.WriteString(rec.Notes + "\n\n")
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	now := g.timeProvider.Now().Format("2006-01-02 15:04")
	fmt.Fprintf(&b, "_Exported %s_\n", now)

	return b.String()
}

// writeWorkItems writes the work items section, grouped by project.
func (g *ExportGenerator) writeWorkItems(b *strings.Builder, rawWork []model.WorkEntry, workTotal time.Duration) {
	// Group by project
	type projGroup struct {
		name    string
		entries []model.WorkEntry
	}
	seen := make(map[string]int)
	var groups []projGroup

	for _, e := range rawWork {
		proj := e.Project
		if proj == "" {
			proj = "—"
		}
		if idx, ok := seen[proj]; ok {
			groups[idx].entries = append(groups[idx].entries, e)
		} else {
			seen[proj] = len(groups)
			groups = append(groups, projGroup{name: proj, entries: []model.WorkEntry{e}})
		}
	}

	// Move "—" (no project) to the end
	var sorted []projGroup
	for _, g := range groups {
		if g.name != "—" {
			sorted = append(sorted, g)
		}
	}
	if idx, ok := seen["—"]; ok {
		sorted = append(sorted, groups[idx])
	}

	// Write each project group
	for _, group := range sorted {
		label := group.name
		if label == "—" {
			label = "Other"
		}
		fmt.Fprintf(b, "### 🗂 %s\n\n", label)

		// Consolidate entries within this project
		consolidated := g.consolidator.Consolidate(group.entries)
		b.WriteString("| Task | Duration |\n|:-----|:--------|\n")

		var projTotal time.Duration
		for _, e := range consolidated {
			fmt.Fprintf(b, "| %s | %s |\n", e.Task, g.formatter.Format(e.Duration()))
			projTotal += e.Duration()
		}
		fmt.Fprintf(b, "\n**Subtotal: %s**\n\n", g.formatter.Format(projTotal))
	}

	fmt.Fprintf(b, "**Total work: %s**\n\n", g.durOrDash(workTotal))
}

// durOrDash returns formatted duration or "—" if zero.
func (g *ExportGenerator) durOrDash(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	return g.formatter.Format(d)
}
