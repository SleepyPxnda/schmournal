package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── Consolidation ─────────────────────────────────────────────────────────────

// ConsolidateEntries merges entries that share the same task name (case-sensitive),
// preserving the original order of first occurrence and summing their durations.
func ConsolidateEntries(entries []WorkEntry) []WorkEntry {
	seen := make(map[string]int) // task → index in out
	var out []WorkEntry
	for _, e := range entries {
		if idx, ok := seen[e.Task]; ok {
			out[idx].Duration += e.Duration
		} else {
			seen[e.Task] = len(out)
			out = append(out, e)
		}
	}
	return out
}

// ── Report generation ─────────────────────────────────────────────────────────

// ExportDay builds a clean, self-contained Markdown report for one entry.
func ExportDay(entry Entry) string {
	all := parseAllWorkLogEntries(entry.Content)

	var rawWork, rawBreaks []WorkEntry
	for _, e := range all {
		if e.IsBreak {
			rawBreaks = append(rawBreaks, e)
		} else {
			rawWork = append(rawWork, e)
		}
	}

	breakItems := ConsolidateEntries(rawBreaks)

	var workTotal, breakTotal time.Duration
	for _, e := range rawWork {
		workTotal += e.Duration
	}
	for _, e := range breakItems {
		breakTotal += e.Duration
	}

	startTime, endTime := parseWorkDayTimes(entry.Content)
	dayDur, hasDayDur := calcDayDuration(startTime, endTime)

	var b strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	fmt.Fprintf(&b, "# Daily Work Report — %s\n\n",
		entry.Date.Format("Monday, January 2, 2006"))

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
		// Preserve insertion order of projects.
		type projGroup struct {
			name    string
			entries []WorkEntry
		}
		seen := make(map[string]int)
		var groups []projGroup
		for _, e := range rawWork {
			proj := e.Project
			if proj == "" || proj == "—" {
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
			consolidated := ConsolidateEntries(g.entries)
			b.WriteString("| Task | Duration |\n|:-----|:--------|\n")
			var projTotal time.Duration
			for _, e := range consolidated {
				fmt.Fprintf(&b, "| %s | %s |\n", e.Task, FormatDuration(e.Duration))
				projTotal += e.Duration
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
			fmt.Fprintf(&b, "| %s | %s |\n", e.Task, FormatDuration(e.Duration))
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

	fmt.Fprintf(&b, "_Exported %s_\n", time.Now().Format("2006-01-02 15:04"))
	return b.String()
}

func durOrDash(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	return FormatDuration(d)
}

// ── File I/O ──────────────────────────────────────────────────────────────────

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
func SaveExport(entry Entry) (string, error) {
	dir, err := ExportDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "export-"+entry.Date.Format("2006-01-02")+".md")
	return path, os.WriteFile(path, []byte(ExportDay(entry)), 0o644)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// parseWorkDayTimes reads **Start:** and **End:** values from entry content.
func parseWorkDayTimes(content string) (start, end string) {
	start, end = "—", "—"
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "**Start:**") {
			start = strings.TrimSpace(strings.TrimPrefix(t, "**Start:**"))
		}
		if strings.HasPrefix(t, "**End:**") {
			end = strings.TrimSpace(strings.TrimPrefix(t, "**End:**"))
		}
	}
	return
}

// calcDayDuration computes the wall-clock duration between two HH:MM strings.
func calcDayDuration(start, end string) (time.Duration, bool) {
	if start == "—" || end == "—" {
		return 0, false
	}
	loc := time.Now().Location()
	s, err1 := time.ParseInLocation("15:04", start, loc)
	e, err2 := time.ParseInLocation("15:04", end, loc)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	dur := e.Sub(s)
	if dur <= 0 {
		return 0, false
	}
	return dur, true
}
