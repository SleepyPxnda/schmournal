package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// streakIterLimit returns the maximum number of backward iterations needed
// when scanning for a streak. It is the number of calendar days from the
// oldest record's date to today (inclusive), so the loop is always bounded
// by real data rather than an arbitrary constant.
func streakIterLimit(records []DayRecord, today time.Time) int {
	oldest := today
	for _, r := range records {
		if t, err := r.ParseDate(); err == nil && t.Before(oldest) {
			oldest = t
		}
	}
	return int(today.Sub(oldest).Hours()/24) + 1
}

func (m Model) View() string {
	if !m.window.Ready {
		return "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve)).Render("Loading…")
	}
	switch m.ui.Current {
	case stateList:
		return m.viewList()
	case stateDayView:
		return m.viewDayView()
	case stateWorkForm:
		return m.viewWorkLogForm()
	case stateClockForm:
		return m.viewClockForm()
	case stateTimeInput:
		return m.viewTimeInput()
	case stateNotesEditor:
		return m.viewNotesEditor()
	case stateTodoForm:
		return m.viewTodoForm()
	case stateConfirmDelete:
		return m.viewConfirmDelete()
	case stateDateInput:
		return m.viewDateInput()
	case stateWeekView:
		return m.viewWeekView()
	case stateWorkspacePicker:
		return m.viewWorkspacePicker()
	case stateStats:
		return m.viewStats()
	}
	return ""
}

func (m Model) renderDayContent() string {
	if m.day.Selection.DayTab == 1 {
		return m.renderSummaryContent()
	}
	return m.renderWorkLogContent()
}

func (m Model) renderWorkLogContent() string {
	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	// ── Work Day section (full width) ─────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("🕐  Work Day") + "\n")
	b.WriteString(div + "\n")

	start := m.day.Record.StartTime
	end := m.day.Record.EndTime
	startDisplay := start
	if startDisplay == "" {
		startDisplay = dayViewMutedStyle.Render("—")
	} else {
		startDisplay = dayViewValueStyle.Render(start)
	}
	endDisplay := end
	if endDisplay == "" {
		endDisplay = dayViewMutedStyle.Render("—")
	} else {
		endDisplay = dayViewValueStyle.Render(end)
	}
	timeLine := "  " + dayViewLabelStyle.Render("Start:") + " " + startDisplay +
		"   " + dayViewLabelStyle.Render("End:") + " " + endDisplay
	if dur, ok := m.day.Record.DayDuration(); ok {
		timeLine += "   " + dayViewMutedStyle.Render("("+formatDuration(dur)+")")
	}
	b.WriteString(timeLine + "\n\n")

	// ── Two-column section: entries (left) + clock panel (right) ──────────────
	// The clock panel has a left border (+1 char), so:
	//   leftW + 1 + rightW = innerW  →  leftW = innerW - rightW - 1
	const clockMinW = 28
	clockEnabled := m.context.Config.Modules.ClockEnabled
	if clockEnabled && innerW >= 60 {
		rightW := innerW / 2
		if rightW < clockMinW {
			rightW = clockMinW
		}
		leftW := innerW - rightW - 1
		leftBlock := lipgloss.NewStyle().Width(leftW).Render(m.renderEntriesPanel(leftW))
		rightBlock := clockPanelBorderStyle.Width(rightW).Render(m.renderClockPanel(rightW))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock))
		b.WriteString("\n")
	} else if clockEnabled {
		// Narrow terminal: stack entries above the clock panel.
		b.WriteString(m.renderEntriesPanel(innerW))
		b.WriteString("\n" + div + "\n")
		b.WriteString(m.renderClockPanel(innerW))
		b.WriteString("\n")
	} else {
		// Clock disabled: entries take full width.
		b.WriteString(m.renderEntriesPanel(innerW))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// ── Notes + Todos two-column section ───────────────────────────────────────
	todoEnabled := m.context.Config.Modules.TodoEnabled
	b.WriteString("\n" + div + "\n")
	if todoEnabled && innerW >= 60 {
		leftW := (innerW - 1) / 2
		rightW := innerW - leftW - 1
		leftPanel := m.renderNotesPanel(leftW)
		rightPanel := m.renderTodosPanel(rightW)
		maxH := lipgloss.Height(leftPanel)
		if h := lipgloss.Height(rightPanel); h > maxH {
			maxH = h
		}
		padToHeight := func(s string, h int) string {
			cur := lipgloss.Height(s)
			if cur >= h {
				return s
			}
			return s + strings.Repeat("\n", h-cur)
		}
		leftPanel = padToHeight(leftPanel, maxH)
		rightPanel = padToHeight(rightPanel, maxH)
		leftBlock := lipgloss.NewStyle().Width(leftW).Height(maxH).Render(leftPanel)
		rightBlock := clockPanelBorderStyle.Width(rightW).Height(maxH).Render(rightPanel)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock))
		b.WriteString("\n")
	} else if todoEnabled {
		b.WriteString(m.renderNotesPanel(innerW))
		b.WriteString("\n" + div + "\n")
		b.WriteString(m.renderTodosPanel(innerW))
		b.WriteString("\n")
	} else {
		// Todo disabled: notes take full width.
		b.WriteString(m.renderNotesPanel(innerW))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderNotesPanel(w int) string {
	var b strings.Builder
	b.WriteString(dayViewSectionStyle.Render("📝  Notes") + "\n")
	b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
	if m.day.Record.Notes == "" {
		b.WriteString(dayViewMutedStyle.Render("  No notes") + "\n")
		return b.String()
	}
	for _, line := range strings.Split(m.day.Record.Notes, "\n") {
		b.WriteString(dayViewNotesStyle.Render("  "+line) + "\n")
	}
	return b.String()
}

// renderEntriesPanel renders the Work Log entry table and totals for the given
// column width. It is used both in the full-width (narrow terminal) path and as
// the left column of the two-column layout on wider terminals.
func (m Model) renderEntriesPanel(w int) string {
	var b strings.Builder
	b.WriteString(dayViewSectionStyle.Render("📋  Work Log") + "\n")

	entries := m.day.Record.Entries
	if len(entries) == 0 {
		b.WriteString(dayViewMutedStyle.Render("  No entries yet") + "\n")
	} else {
		// column widths: selector(2) + project(14) + task(dynamic) + duration(8)
		taskW := w - 2 - 14 - 8
		if taskW < 10 {
			taskW = 10
		}
		for i, e := range entries {
			selector := "  "
			if i == m.day.Selection.EntryIdx {
				selector = "▶ "
			}

			proj := fmt.Sprintf("%-14s", e.Project)
			taskStr := e.Task
			if e.IsBreak {
				taskStr = "☕  " + taskStr
			}
			if len(taskStr) > taskW {
				taskStr = taskStr[:taskW-1] + "…"
			}
			// ☕ is a wide character (2 terminal cols) but counts as 1 rune,
			// so reduce the pad width by 1 for break entries to stay within w.
			fmtW := taskW
			if e.IsBreak {
				fmtW = taskW - 1
			}
			taskStr = fmt.Sprintf("%-*s", fmtW, taskStr)
			durStr := fmt.Sprintf("%8s", formatDuration(e.Duration()))

			line := selector + proj + taskStr + durStr

			if i == m.day.Selection.EntryIdx {
				line = selectedEntryStyle.Render(line)
			} else if e.IsBreak {
				line = breakEntryStyle.Render(line)
			} else {
				line = normalEntryStyle.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	work, breaks, _ := m.day.Record.WorkTotals()
	if work > 0 || breaks > 0 {
		totals := "  Work: " + formatDuration(work)
		if breaks > 0 {
			totals += "  ·  Breaks: " + formatDuration(breaks)
			totals += "  ·  Total: " + formatDuration(work+breaks)
		}
		b.WriteString("\n" + dayViewTotalsStyle.Render(totals) + "\n")
	}
	if dayDur, ok := m.day.Record.DayDuration(); ok {
		logged := work + breaks
		if logged != dayDur {
			diff := dayDur - logged
			sign := "+"
			if diff < 0 {
				diff = -diff
				sign = "-"
			}
			warn := fmt.Sprintf("  ⚠  Logged (%s) vs span (%s) Δ%s%s",
				formatDuration(logged), formatDuration(dayDur),
				sign, formatDuration(diff))
			b.WriteString(dayViewWarnStyle.Render(warn) + "\n")
		}
	}

	return b.String()
}

// renderClockPanel renders the clock panel for a given column width.
// It shows an animated timer when the clock is running, or idle instructions.
func (m Model) renderClockPanel(w int) string {
	div := dayViewDividerStyle.Render(strings.Repeat("─", w))
	kb := m.context.Config.Keybinds.Day

	var b strings.Builder
	b.WriteString(" " + dayViewSectionStyle.Render("⏱  Clocking") + "\n")
	b.WriteString(" " + div + "\n\n")

	if !m.clock.Running {
		b.WriteString(" " + dayViewMutedStyle.Render("No active timer") + "\n\n")
		b.WriteString(" " + dayViewLabelStyle.Render("Press ") +
			helpKeyStyle.Render(kb.ClockStart) +
			dayViewLabelStyle.Render(" to start") + "\n")
		return b.String()
	}

	// Animated clock emoji cycles through 12 clock-face positions.
	clockEmojis := []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
	emoji := clockEmojis[m.clock.Frame%12]

	elapsed := time.Since(m.clock.Start)
	totalSec := int(elapsed.Seconds())
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	elapsedStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	b.WriteString(" " + emoji + "  " + dayViewValueStyle.Render(elapsedStr) + "\n\n")

	// Truncate task/project names so they don't overflow the column.
	// 12 = 1 (leading space) + 9 ("Project: ") + 1 (trailing space) + 1 (ellipsis reserve)
	maxNameW := w - 12
	if maxNameW < 6 {
		maxNameW = 6
	}
	taskStr := m.clock.Task
	if len(taskStr) > maxNameW {
		taskStr = taskStr[:maxNameW-1] + "…"
	}
	b.WriteString(" " + dayViewLabelStyle.Render("Task:    ") + dayViewValueStyle.Render(taskStr) + "\n")
	if m.clock.Project != "" {
		projStr := m.clock.Project
		if len(projStr) > maxNameW {
			projStr = projStr[:maxNameW-1] + "…"
		}
		b.WriteString(" " + dayViewLabelStyle.Render("Project: ") + dayViewValueStyle.Render(projStr) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(" " + dayViewMutedStyle.Render(kb.ClockStop+" stop & log") + "\n")

	return b.String()
}

func (m Model) renderSummaryContent() string {
	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	// ── Work Day times ────────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("🕐  Work Day") + "\n")
	b.WriteString(div + "\n")
	startD, endD := m.day.Record.StartTime, m.day.Record.EndTime
	if startD == "" {
		startD = dayViewMutedStyle.Render("—")
	} else {
		startD = dayViewValueStyle.Render(startD)
	}
	if endD == "" {
		endD = dayViewMutedStyle.Render("—")
	} else {
		endD = dayViewValueStyle.Render(endD)
	}
	timeLine := "  " + dayViewLabelStyle.Render("Start:") + " " + startD +
		"   " + dayViewLabelStyle.Render("End:") + " " + endD
	if dur, ok := m.day.Record.DayDuration(); ok {
		timeLine += "   " + dayViewMutedStyle.Render("("+formatDuration(dur)+")")
	}
	b.WriteString(timeLine + "\n\n")

	// ── Projects ──────────────────────────────────────────────────────────────
	type projGroup struct {
		name    string
		entries []WorkEntry
	}
	seen := make(map[string]int)
	var groups []projGroup
	var breakEntries []WorkEntry

	for _, e := range m.day.Record.Entries {
		if e.IsBreak {
			breakEntries = append(breakEntries, e)
			continue
		}
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

	// consolidateByName merges entries with the same task name (case-insensitive).
	consolidateByName := func(entries []WorkEntry) []WorkEntry {
		seen := make(map[string]int)
		var out []WorkEntry
		for _, e := range entries {
			key := strings.ToLower(e.Task)
			if idx, ok := seen[key]; ok {
				out[idx].DurationMin += e.DurationMin
			} else {
				seen[key] = len(out)
				out = append(out, e)
			}
		}
		return out
	}

	if len(groups) == 0 && len(breakEntries) == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No entries yet") + "\n")
	} else {
		b.WriteString(dayViewSectionStyle.Render("🗂  By Project") + "\n")
		b.WriteString(div + "\n")

		for _, g := range groups {
			tasks := consolidateByName(g.entries)
			var projTotal time.Duration
			var names []string
			for _, t := range tasks {
				projTotal += t.Duration()
				names = append(names, t.Task)
			}

			projLabel := g.name
			if projLabel == "—" {
				projLabel = "Other"
			}

			durStr := fmt.Sprintf("%-8s", formatDuration(projTotal))
			taskList := "\"" + strings.Join(names, ", ") + "\""
			b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
				"  " + dayViewSectionStyle.Render(projLabel) +
				"  " + dayViewMutedStyle.Render(taskList) + "\n")
		}

		// ── Breaks block ──────────────────────────────────────────────────────
		if len(breakEntries) > 0 {
			bkList := consolidateByName(breakEntries)
			var breakTotal time.Duration
			var names []string
			for _, e := range bkList {
				breakTotal += e.Duration()
				names = append(names, e.Task)
			}
			durStr := fmt.Sprintf("%-8s", formatDuration(breakTotal))
			taskList := "\"" + strings.Join(names, ", ") + "\""
			b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
				"  " + breakEntryStyle.Render("☕  Breaks") +
				"  " + dayViewMutedStyle.Render(taskList) + "\n")
		}

		// ── Totals ────────────────────────────────────────────────────────────
		work, breaks, total := m.day.Record.WorkTotals()
		b.WriteString(div + "\n")
		if breaks > 0 {
			b.WriteString(dayViewTotalsStyle.Render(fmt.Sprintf("  Work: %s  ·  Breaks: %s  ·  Total: %s",
				formatDuration(work), formatDuration(breaks), formatDuration(total))) + "\n")
		} else if work > 0 {
			b.WriteString(dayViewTotalsStyle.Render("  Total work: "+formatDuration(work)) + "\n")
		}
	}

	return b.String()
}

func (m Model) renderEOMBanner() string {
	now := time.Now()
	// last day of the month: first day of next month minus one day
	firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if now.Before(firstOfNext.AddDate(0, 0, -1)) {
		return ""
	}
	return eomBannerStyle.Width(m.window.Width).Render("⚠  Last day of the month — don't forget to submit your times!")
}

func (m Model) renderStats() string {
	now := time.Now()

	// Build a set of dates that have records.
	dated := make(map[string]bool, len(m.listState.Records))
	for _, r := range m.listState.Records {
		dated[r.Date] = true
	}

	// Week bar: ISO week starting Monday.
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	dayInitials := [7]string{"M", "T", "W", "T", "F", "S", "S"}
	const filled = "▓"
	const empty = "░"
	var weekParts []string
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		lbl := statsLabelStyle.Render(dayInitials[i])
		var block string
		switch {
		case dated[dateStr]:
			block = statsBlockFilledStyle.Render(filled)
		case !m.effectiveIsWorkDay(d):
			block = statsBlockNonWorkStyle.Render(empty)
		case d.After(now):
			block = statsBlockFutureStyle.Render(empty)
		default:
			block = statsBlockEmptyStyle.Render(empty)
		}
		weekParts = append(weekParts, lbl+" "+block)
	}
	weekStr := strings.Join(weekParts, "  ")

	// Month count.
	monthKey := now.Format("2006-01")
	monthCount := 0
	for _, r := range m.listState.Records {
		if strings.HasPrefix(r.Date, monthKey) {
			monthCount++
		}
	}
	monthStr := statsLabelStyle.Render(now.Format("Jan")+": ") +
		statsValueStyle.Render(fmt.Sprintf("%d", monthCount))

	// Streak: consecutive working days going back from today.
	// Non-working days are skipped — they neither add to the count nor break
	// the streak — so a weekend or public holiday never resets the counter.
	// The iteration limit is derived from the oldest record so the loop is
	// always bounded by real data and never underestimates long streaks.
	streak := 0
	for i := 0; i < streakIterLimit(m.listState.Records, now); i++ {
		check := now.AddDate(0, 0, -i)
		dateStr := check.Format("2006-01-02")
		if dated[dateStr] {
			streak++
		} else if m.effectiveIsWorkDay(check) {
			// A working day with no entry breaks the streak.
			break
		}
		// Non-working day without an entry: continue (streak passes through).
	}
	var streakStr string
	if streak > 0 {
		streakStr = "🔥 " + statsStreakStyle.Render(fmt.Sprintf("%d", streak)) +
			statsLabelStyle.Render(" day streak")
	} else {
		streakStr = statsLabelStyle.Render("No active streak")
	}

	dot := helpStyle.Render("  ·  ")
	line := "  " + weekStr + dot + monthStr + dot + streakStr
	sep := separatorStyle.Render(strings.Repeat("─", m.window.Width))

	// Weekly work hours progress bar.
	var weekWork time.Duration
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		for _, r := range m.listState.Records {
			if r.Date == dateStr {
				work, _, _ := r.WorkTotals()
				weekWork += work
				break
			}
		}
	}
	goalHours := m.effectiveWeeklyHoursGoal()
	var weekHoursStr string
	if goalHours == 0 {
		weekHoursStr = "  " + statsLabelStyle.Render("Week: ") +
			statsValueStyle.Render(formatDuration(weekWork))
	} else {
		weeklyGoal := time.Duration(goalHours * float64(time.Hour))
		pct := float64(weekWork) / float64(weeklyGoal)
		if pct > 1 {
			pct = 1
		}
		goalLabel := fmt.Sprintf(" / %gh  ", goalHours)
		const progressBarWidth = 20
		filledCount := int(pct * progressBarWidth)
		progressBar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
			statsBlockEmptyStyle.Render(strings.Repeat("░", progressBarWidth-filledCount))
		weekHoursStr = "  " + statsLabelStyle.Render("Week: ") +
			statsValueStyle.Render(formatDuration(weekWork)) +
			statsLabelStyle.Render(goalLabel) +
			progressBar +
			statsLabelStyle.Render(fmt.Sprintf("  %.0f%%", pct*100))
	}

	return line + "\n" + weekHoursStr + "\n" + sep
}

// viewList has been moved to views_list.go
// viewDayView has been moved to views_day.go
// All form views (work, clock, time, date, todo, confirm, workspace) have been moved to views_forms.go
// viewNotesEditor has been moved to views_notes.go
// viewStats and all renderStats* functions have been moved to views_stats.go

func (m Model) viewWeekView() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)
	sunday := monday.AddDate(0, 0, 6)

	weekRange := monday.Format("02 Jan") + " – " + sunday.Format("02 Jan 2006")
	header := m.renderHeader("🗓  This Week", weekRange)

	var navParts []string
	navParts = append(navParts, inactiveTabStyle.Render("← prev week"))
	if m.weekOffset < 0 {
		navParts = append(navParts, inactiveTabStyle.Render("→ next week"))
	}
	navBar := strings.Join(navParts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.window.Width))
	subHeader := navBar + "\n" + sep

	footer := m.renderFooter([][2]string{
		{"j/k", "scroll"},
		{"←/→", "prev/next week"},
		{"esc", "back"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, subHeader, m.day.Viewport.View(), footer)
}

func (m Model) renderWeekContent() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)

	recByDate := make(map[string]DayRecord, len(m.listState.Records))
	for _, r := range m.listState.Records {
		recByDate[r.Date] = r
	}

	innerW := m.window.Width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	var weekWork time.Duration
	var weekBreaks time.Duration

	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		rec, hasRec := recByDate[dateStr]

		var work, breaks time.Duration
		if hasRec {
			work, breaks, _ = rec.WorkTotals()
		}
		weekWork += work
		weekBreaks += breaks

		dayLabel := d.Format("Mon  02 Jan 2006")
		isWorkDay := m.effectiveIsWorkDay(d)
		dayLabelStyle := dayViewSectionStyle
		if !isWorkDay {
			dayLabelStyle = weekNonWorkDayStyle
		}
		var headerLine string
		if hasRec && (work+breaks) > 0 {
			headerLine = dayLabelStyle.Render(dayLabel) +
				dayViewMutedStyle.Render("  ·  ") +
				dayViewValueStyle.Render(formatDuration(work)+" work")
			if breaks > 0 {
				headerLine += dayViewMutedStyle.Render("  ·  " + formatDuration(breaks) + " breaks")
			}
		} else {
			headerLine = dayLabelStyle.Render(dayLabel) +
				dayViewMutedStyle.Render("  ·  no entries")
		}
		b.WriteString(headerLine + "\n")

		if hasRec && len(rec.Entries) > 0 {
			type projGroup struct {
				name    string
				dur     time.Duration
				tasks   []string
				isBreak bool
			}
			seenProj := make(map[string]int)
			var groups []projGroup

			for _, e := range rec.Entries {
				if e.IsBreak {
					found := false
					for gi, g := range groups {
						if g.isBreak {
							groups[gi].dur += e.Duration()
							groups[gi].tasks = uniqueAppend(groups[gi].tasks, e.Task)
							found = true
							break
						}
					}
					if !found {
						groups = append(groups, projGroup{
							name:    "Breaks",
							dur:     e.Duration(),
							tasks:   []string{e.Task},
							isBreak: true,
						})
					}
					continue
				}

				proj := e.Project
				if proj == "" {
					proj = "Other"
				}
				if idx, ok := seenProj[proj]; ok {
					groups[idx].dur += e.Duration()
					groups[idx].tasks = uniqueAppend(groups[idx].tasks, e.Task)
				} else {
					seenProj[proj] = len(groups)
					groups = append(groups, projGroup{
						name:  proj,
						dur:   e.Duration(),
						tasks: []string{e.Task},
					})
				}
			}

			for _, g := range groups {
				durStr := fmt.Sprintf("%-8s", formatDuration(g.dur))
				taskStr := "\"" + strings.Join(g.tasks, ", ") + "\""
				if g.isBreak {
					b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
						"  " + breakEntryStyle.Render("☕  "+g.name) +
						"  " + dayViewMutedStyle.Render(taskStr) + "\n")
				} else {
					b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
						"  " + dayViewSectionStyle.Render(g.name) +
						"  " + dayViewMutedStyle.Render(taskStr) + "\n")
				}
			}
		}

		if i < 6 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n" + div + "\n")
	totalStr := "  Week total work: " + formatDuration(weekWork)
	if weekBreaks > 0 {
		totalStr += "  ·  breaks: " + formatDuration(weekBreaks)
		totalStr += "  ·  logged: " + formatDuration(weekWork+weekBreaks)
	}
	b.WriteString(dayViewTotalsStyle.Render(totalStr) + "\n")

	return b.String()
}

func uniqueAppend(ss []string, s string) []string {
	for _, x := range ss {
		if x == s {
			return ss
		}
	}
	return append(ss, s)
}
