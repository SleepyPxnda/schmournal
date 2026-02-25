package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// consolidateEntries merges entries that share the same task name (case-sensitive),
// summing their DurationMin values.
func consolidateEntries(entries []WorkEntry) []WorkEntry {
	seen := make(map[string]int) // task → index in out
	var out []WorkEntry
	for _, e := range entries {
		if idx, ok := seen[e.Task]; ok {
			out[idx].DurationMin += e.DurationMin
		} else {
			seen[e.Task] = len(out)
			out = append(out, e)
		}
	}
	return out
}

// ExportDay builds a clean, self-contained Markdown report for one DayRecord.
func ExportDay(rec DayRecord) string {
	var rawWork, rawBreaks []WorkEntry
	for _, e := range rec.Entries {
		if e.IsBreak {
			rawBreaks = append(rawBreaks, e)
		} else {
			rawWork = append(rawWork, e)
		}
	}

	breakItems := consolidateEntries(rawBreaks)

	var workTotal, breakTotal time.Duration
	for _, e := range rawWork {
		workTotal += e.Duration()
	}
	for _, e := range breakItems {
		breakTotal += e.Duration()
	}

	startTime := rec.StartTime
	if startTime == "" {
		startTime = "—"
	}
	endTime := rec.EndTime
	if endTime == "" {
		endTime = "—"
	}
	dayDur, hasDayDur := rec.DayDuration()

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
		fmt.Fprintf(&b, "| **Day duration** | %s |\n", FormatDuration(dayDur))
	}
	b.WriteString("\n")

	// ── Work Items grouped by project ─────────────────────────────────────────
	b.WriteString("## 📋 Work Items\n\n")
	if len(rawWork) == 0 {
		b.WriteString("_No work items logged._\n\n")
	} else {
		type projGroup struct {
			name    string
			entries []WorkEntry
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
				groups = append(groups, projGroup{name: proj, entries: []WorkEntry{e}})
			}
		}
		// Move "—" (no project) to the end.
		var sorted []projGroup
		for _, g := range groups {
			if g.name != "—" {
				sorted = append(sorted, g)
			}
		}
		if idx, ok := seen["—"]; ok {
			sorted = append(sorted, groups[idx])
		}

		for _, g := range sorted {
			label := g.name
			if label == "—" {
				label = "Other"
			}
			fmt.Fprintf(&b, "### 🗂 %s\n\n", label)
			consolidated := consolidateEntries(g.entries)
			b.WriteString("| Task | Duration |\n|:-----|:--------|\n")
			var projTotal time.Duration
			for _, e := range consolidated {
				fmt.Fprintf(&b, "| %s | %s |\n", e.Task, FormatDuration(e.Duration()))
				projTotal += e.Duration()
			}
			fmt.Fprintf(&b, "\n**Subtotal: %s**\n\n", FormatDuration(projTotal))
		}
		fmt.Fprintf(&b, "**Total work: %s**\n\n", durOrDash(workTotal))
	}

	// ── Breaks ────────────────────────────────────────────────────────────────
	b.WriteString("## ☕ Breaks\n\n")
	if len(breakItems) == 0 {
		b.WriteString("_No breaks logged._\n\n")
	} else {
		b.WriteString("| Break | Duration |\n|:------|:--------|\n")
		for _, e := range breakItems {
			fmt.Fprintf(&b, "| %s | %s |\n", e.Task, FormatDuration(e.Duration()))
		}
		fmt.Fprintf(&b, "\n**Total breaks: %s**\n\n", durOrDash(breakTotal))
	}

	// ── Summary ───────────────────────────────────────────────────────────────
	b.WriteString("## 📊 Summary\n\n")
	b.WriteString("| | |\n|:---|:---|\n")
	fmt.Fprintf(&b, "| **Work** | %s |\n", durOrDash(workTotal))
	fmt.Fprintf(&b, "| **Breaks** | %s |\n", durOrDash(breakTotal))
	if workTotal+breakTotal > 0 {
		fmt.Fprintf(&b, "| **Total logged** | %s |\n", FormatDuration(workTotal+breakTotal))
	}
	if hasDayDur {
		fmt.Fprintf(&b, "| **Day duration** | %s |\n", FormatDuration(dayDur))
	}
	b.WriteString("\n")

	if rec.Notes != "" {
		b.WriteString("## 📝 Notes\n\n")
		b.WriteString(rec.Notes + "\n\n")
	}

	fmt.Fprintf(&b, "_Exported %s_\n", time.Now().Format("2006-01-02 15:04"))
	return b.String()
}

func durOrDash(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	return FormatDuration(d)
}

// ExportDir returns (and creates) ~/.journal/exports/.
func ExportDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "exports")
	return p, os.MkdirAll(p, 0o755)
}

// SaveExport writes the export report to disk and returns its path.
func SaveExport(rec DayRecord) (string, error) {
	dir, err := ExportDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "export-"+rec.Date+".md")
	return path, os.WriteFile(path, []byte(ExportDay(rec)), 0o644)
}
