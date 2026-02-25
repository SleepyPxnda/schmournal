package journal

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BreakPrefix is prepended to break-entry task names so they can be identified
// visually in the rendered table and programmatically when recalculating totals.
const BreakPrefix = "☕ "

// WorkEntry is a single parsed row from the Work Log table.
type WorkEntry struct {
	Project  string
	Task     string
	IsBreak  bool
	Duration time.Duration
}

// NewEntryTemplate returns the full daily journal template.
// The Work Log uses a 3-column table: Project | Task | Duration.
// Breaks are rows identified by BreakPrefix and carry "—" as project.
func NewEntryTemplate(t time.Time) string {
	return fmt.Sprintf(
		"# %s\n\n"+
			"## 🕐 Work Day\n"+
			"**Start:** —\n"+
			"**End:** —\n\n"+
			"## 📋 Work Log\n\n"+
			"| Project | Task | Duration |\n"+
			"|---------|------|----------|\n\n"+
			"**Work:** —  ·  **Breaks:** —  ·  **Total:** —\n\n"+
			"## 📝 Notes\n\n",
		t.Format("January 2, 2006"),
	)
}

// ── Time stamping ─────────────────────────────────────────────────────────────

// StampStartTime replaces the **Start:** line with timeStr regardless of the
// current value (placeholder or an existing time).
func StampStartTime(content, timeStr string) (string, bool) {
	return stampField(content, "**Start:**", timeStr)
}

// StampEndTime replaces the **End:** line with timeStr.
func StampEndTime(content, timeStr string) (string, bool) {
	return stampField(content, "**End:**", timeStr)
}

func stampField(content, prefix, value string) (string, bool) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[i] = prefix + " " + value
			return strings.Join(lines, "\n"), true
		}
	}
	return content, false
}

// ── Work log entries ──────────────────────────────────────────────────────────

// AddWorkEntry inserts a plain task row into the Work Log table and refreshes totals.
// project may be empty or "—" to indicate no project.
func AddWorkEntry(content, task, project string, dur time.Duration) string {
	if project == "" {
		project = "—"
	}
	return insertWorkLogRow(content, task, project, dur)
}

// AddBreakEntry inserts a break row (prefixed with BreakPrefix) into the Work Log
// table and refreshes totals. Breaks always carry "—" as project.
func AddBreakEntry(content, label string, dur time.Duration) string {
	return insertWorkLogRow(content, BreakPrefix+label, "—", dur)
}

// insertWorkLogRow adds a new 3-column row just before the totals line, then recalculates.
func insertWorkLogRow(content, task, project string, dur time.Duration) string {
	newRow := fmt.Sprintf("| %s | %s | %s |", project, task, FormatDuration(dur))
	lines := strings.Split(content, "\n")

	// Find the totals line (supports both old "**Total:**" and new "**Work:**" format).
	for i, line := range lines {
		if isTotalsLine(strings.TrimSpace(line)) {
			ins := i
			if ins > 0 && strings.TrimSpace(lines[ins-1]) == "" {
				ins--
			}
			updated := make([]string, 0, len(lines)+1)
			updated = append(updated, lines[:ins]...)
			updated = append(updated, newRow)
			updated = append(updated, lines[ins:]...)
			return recalcTotals(strings.Join(updated, "\n"))
		}
	}
	// Fallback: no totals line found – just append.
	return content + "\n" + newRow
}

// isTotalsLine returns true for both the old "**Total:** …" and new "**Work:** …" formats.
func isTotalsLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "**Work:**") ||
		(strings.HasPrefix(trimmed, "**Total:**") && !strings.Contains(trimmed, "**Work:**"))
}

// recalcTotals rewrites the totals line with split work / breaks / total values.
func recalcTotals(content string) string {
	all := parseAllWorkLogEntries(content)
	var workDur, breakDur time.Duration
	for _, e := range all {
		if e.IsBreak {
			breakDur += e.Duration
		} else {
			workDur += e.Duration
		}
	}

	fmtDur := func(d time.Duration) string {
		if d == 0 {
			return "—"
		}
		return FormatDuration(d)
	}
	total := workDur + breakDur
	newLine := fmt.Sprintf("**Work:** %s  ·  **Breaks:** %s  ·  **Total:** %s",
		fmtDur(workDur), fmtDur(breakDur), fmtDur(total))

	lines := strings.Split(content, "\n")
	inWorkLog := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "## 📋 Work Log") {
			inWorkLog = true
		}
		if inWorkLog && isTotalsLine(trimmed) {
			lines[i] = newLine
			break
		}
	}
	return strings.Join(lines, "\n")
}

// UpdateWorkTotal is kept for backward compatibility; it now calls recalcTotals.
func UpdateWorkTotal(content string) string {
	return recalcTotals(content)
}

// ── Parsing ───────────────────────────────────────────────────────────────────

// ParseWorkEntries returns only non-break rows from the Work Log table.
func ParseWorkEntries(content string) []WorkEntry {
	var out []WorkEntry
	for _, e := range parseAllWorkLogEntries(content) {
		if !e.IsBreak {
			out = append(out, e)
		}
	}
	return out
}

// ParseBreakEntries returns only break rows (☕-prefixed) from the Work Log table.
func ParseBreakEntries(content string) []WorkEntry {
	var out []WorkEntry
	for _, e := range parseAllWorkLogEntries(content) {
		if e.IsBreak {
			out = append(out, e)
		}
	}
	return out
}

// parseAllWorkLogEntries returns every data row from the Work Log table.
// It handles both the legacy 2-column format (Task | Duration) and the
// current 3-column format (Project | Task | Duration).
func parseAllWorkLogEntries(content string) []WorkEntry {
	var entries []WorkEntry
	inWorkLog := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "## 📋 Work Log") || trimmed == "## Work Log" {
			inWorkLog = true
			continue
		}
		if inWorkLog && strings.HasPrefix(trimmed, "##") {
			break
		}
		if !inWorkLog || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		// Skip header and separator rows.
		if strings.Contains(trimmed, "Task") || strings.Contains(trimmed, "---") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		// parts[0] and parts[last] are empty due to leading/trailing pipes.
		// 3-col new: | Project | Task | Duration | → len==5
		// 2-col old: | Task | Duration |          → len==4
		var project, task, durStr string
		switch len(parts) {
		case 5: // 3-column: project | task | duration
			project = strings.TrimSpace(parts[1])
			task = strings.TrimSpace(parts[2])
			durStr = strings.TrimSpace(parts[3])
		case 4: // 2-column legacy: task | duration
			project = "—"
			task = strings.TrimSpace(parts[1])
			durStr = strings.TrimSpace(parts[2])
		default:
			continue
		}
		if task == "" || durStr == "" || durStr == "—" {
			continue
		}
		dur, err := ParseDuration(durStr)
		if err != nil {
			continue
		}
		isBreak := strings.HasPrefix(task, BreakPrefix)
		if isBreak {
			task = strings.TrimPrefix(task, BreakPrefix)
		}
		if project == "" {
			project = "—"
		}
		entries = append(entries, WorkEntry{
			Project:  project,
			Task:     task,
			IsBreak:  isBreak,
			Duration: dur,
		})
	}
	return entries
}

// ── Duration helpers ──────────────────────────────────────────────────────────

// ParseDuration parses flexible duration strings:
// "1h 30m", "1h30m", "90m", "90min", "1.5h", "2h", "45m".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "—" {
		return 0, fmt.Errorf("empty duration")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	var hours, mins float64
	if n, _ := fmt.Sscanf(s, "%fh %fm", &hours, &mins); n == 2 {
		return time.Duration((hours*60+mins)*float64(time.Minute)), nil
	}
	if n, _ := fmt.Sscanf(s, "%fh %fmin", &hours, &mins); n == 2 {
		return time.Duration((hours*60+mins)*float64(time.Minute)), nil
	}
	if strings.HasSuffix(s, "h") {
		if h, err := strconv.ParseFloat(strings.TrimSuffix(s, "h"), 64); err == nil {
			return time.Duration(h * float64(time.Hour)), nil
		}
	}
	s2 := strings.TrimSuffix(strings.TrimSuffix(s, "min"), "m")
	if m, err := strconv.ParseFloat(strings.TrimSpace(s2), 64); err == nil && m > 0 {
		return time.Duration(m * float64(time.Minute)), nil
	}
	return 0, fmt.Errorf("cannot parse duration %q – try: 1h 30m, 45m, 2h", s)
}

// FormatDuration converts a duration to a human-readable string like "1h 30m".
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}


