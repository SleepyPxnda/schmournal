package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sleepypxnda/schmournal/journal"
)

// streakIterLimit returns the maximum number of backward iterations needed
// when scanning for a streak. It is the number of calendar days from the
// oldest record's date to today (inclusive), so the loop is always bounded
// by real data rather than an arbitrary constant.
func streakIterLimit(records []journal.DayRecord, today time.Time) int {
	oldest := today
	for _, r := range records {
		if t, err := r.ParseDate(); err == nil && t.Before(oldest) {
			oldest = t
		}
	}
	return int(today.Sub(oldest).Hours()/24) + 1
}

func (m Model) View() string {
	if !m.ready {
		return "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve)).Render("Loading…")
	}
	switch m.state {
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
	case stateWorkspacePicker:
		return m.viewWorkspacePicker()
	case stateStats:
		return m.viewStats()
	}
	return ""
}

func (m Model) renderHeader(title, subtitle string) string {
	left := headerTitleStyle.Render(title)
	right := headerSubtitleStyle.Render(subtitle)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	top := left + strings.Repeat(" ", gap) + right
	sep := separatorStyle.Render(strings.Repeat("─", m.width))
	return top + "\n" + sep
}

// appTitle returns the application title with the version number appended.
func (m Model) appTitle() string {
	if m.version != "" && m.version != "dev" {
		ver := m.version
		if strings.HasPrefix(ver, "v") {
			ver = ver[1:]
		}
		return "📔  Schmournal  v" + ver
	}
	return "📔  Schmournal"
}

// joinKeyLabels joins two key labels with "/" for footer display.
// If either label is empty, only the non-empty one is shown; if both are empty "?" is returned.
func joinKeyLabels(a, b string) string {
	if a == "" && b == "" {
		return "?"
	}
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "/" + b
}

func (m Model) renderFooter(keys [][2]string) string {
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k[0])+" "+helpStyle.Render(k[1]))
	}
	help := strings.Join(parts, helpStyle.Render("  ·  "))

	if m.statusMsg != "" {
		s := m.statusMsg
		if m.isError {
			s = statusErrorStyle.Render(s)
		} else {
			s = statusSuccessStyle.Render(s)
		}
		pad := m.width - lipgloss.Width(help) - lipgloss.Width(s) - 2
		if pad < 1 {
			pad = 1
		}
		return help + strings.Repeat(" ", pad) + s
	}
	return help
}

func (m Model) renderTabBar() string {
	tabs := []string{"📋  Work Log", "📊  Summary"}
	var parts []string
	for i, label := range tabs {
		if i == m.dayViewTab {
			parts = append(parts, activeTabStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+label+" "))
		}
	}
	bar := strings.Join(parts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.width))
	return bar + "\n" + sep
}

func (m Model) renderDayContent() string {
	if m.dayViewTab == 1 {
		return m.renderSummaryContent()
	}
	return m.renderWorkLogContent()
}

func (m Model) renderWorkLogContent() string {
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	// ── Work Day section (full width) ─────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("🕐  Work Day") + "\n")
	b.WriteString(div + "\n")

	start := m.dayRecord.StartTime
	end := m.dayRecord.EndTime
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
	if dur, ok := m.dayRecord.DayDuration(); ok {
		timeLine += "   " + dayViewMutedStyle.Render("("+journal.FormatDuration(dur)+")")
	}
	b.WriteString(timeLine + "\n\n")

	// ── Two-column section: entries (left) + clock panel (right) ──────────────
	// The clock panel has a left border (+1 char), so:
	//   leftW + 1 + rightW = innerW  →  leftW = innerW - rightW - 1
	const clockMinW = 28
	if innerW >= 60 {
		rightW := innerW / 2
		if rightW < clockMinW {
			rightW = clockMinW
		}
		leftW := innerW - rightW - 1
		leftBlock := lipgloss.NewStyle().Width(leftW).Render(m.renderEntriesPanel(leftW))
		rightBlock := clockPanelBorderStyle.Width(rightW).Render(m.renderClockPanel(rightW))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock))
		b.WriteString("\n")
	} else {
		// Narrow terminal: stack entries above the clock panel.
		b.WriteString(m.renderEntriesPanel(innerW))
		b.WriteString("\n" + div + "\n")
		b.WriteString(m.renderClockPanel(innerW))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// ── Notes + Todos two-column section ───────────────────────────────────────
	b.WriteString("\n" + div + "\n")
	if innerW >= 60 {
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
	} else {
		b.WriteString(m.renderNotesPanel(innerW))
		b.WriteString("\n" + div + "\n")
		b.WriteString(m.renderTodosPanel(innerW))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderNotesPanel(w int) string {
	var b strings.Builder
	b.WriteString(dayViewSectionStyle.Render("📝  Notes") + "\n")
	b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
	if m.dayRecord.Notes == "" {
		b.WriteString(dayViewMutedStyle.Render("  No notes") + "\n")
		return b.String()
	}
	for _, line := range strings.Split(m.dayRecord.Notes, "\n") {
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

	entries := m.dayRecord.Entries
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
			if i == m.selectedEntry {
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
			durStr := fmt.Sprintf("%8s", journal.FormatDuration(e.Duration()))

			line := selector + proj + taskStr + durStr

			if i == m.selectedEntry {
				line = selectedEntryStyle.Render(line)
			} else if e.IsBreak {
				line = breakEntryStyle.Render(line)
			} else {
				line = normalEntryStyle.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	work, breaks, _ := m.dayRecord.WorkTotals()
	if work > 0 || breaks > 0 {
		totals := "  Work: " + journal.FormatDuration(work)
		if breaks > 0 {
			totals += "  ·  Breaks: " + journal.FormatDuration(breaks)
			totals += "  ·  Total: " + journal.FormatDuration(work+breaks)
		}
		b.WriteString("\n" + dayViewTotalsStyle.Render(totals) + "\n")
	}
	if dayDur, ok := m.dayRecord.DayDuration(); ok {
		logged := work + breaks
		if logged != dayDur {
			diff := dayDur - logged
			sign := "+"
			if diff < 0 {
				diff = -diff
				sign = "-"
			}
			warn := fmt.Sprintf("  ⚠  Logged (%s) vs span (%s) Δ%s%s",
				journal.FormatDuration(logged), journal.FormatDuration(dayDur),
				sign, journal.FormatDuration(diff))
			b.WriteString(dayViewWarnStyle.Render(warn) + "\n")
		}
	}

	return b.String()
}

// renderClockPanel renders the clock panel for a given column width.
// It shows an animated timer when the clock is running, or idle instructions.
func (m Model) renderClockPanel(w int) string {
	div := dayViewDividerStyle.Render(strings.Repeat("─", w))
	kb := m.cfg.Keybinds.Day

	var b strings.Builder
	b.WriteString(" " + dayViewSectionStyle.Render("⏱  Clocking") + "\n")
	b.WriteString(" " + div + "\n\n")

	if !m.clockRunning {
		b.WriteString(" " + dayViewMutedStyle.Render("No active timer") + "\n\n")
		b.WriteString(" " + dayViewLabelStyle.Render("Press ") +
			helpKeyStyle.Render(kb.ClockStart) +
			dayViewLabelStyle.Render(" to start") + "\n")
		return b.String()
	}

	// Animated clock emoji cycles through 12 clock-face positions.
	clockEmojis := []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"}
	emoji := clockEmojis[m.clockFrame%12]

	elapsed := time.Since(m.clockStart)
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
	taskStr := m.clockTask
	if len(taskStr) > maxNameW {
		taskStr = taskStr[:maxNameW-1] + "…"
	}
	b.WriteString(" " + dayViewLabelStyle.Render("Task:    ") + dayViewValueStyle.Render(taskStr) + "\n")
	if m.clockProject != "" {
		projStr := m.clockProject
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
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	// ── Work Day times ────────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("🕐  Work Day") + "\n")
	b.WriteString(div + "\n")
	startD, endD := m.dayRecord.StartTime, m.dayRecord.EndTime
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
	if dur, ok := m.dayRecord.DayDuration(); ok {
		timeLine += "   " + dayViewMutedStyle.Render("("+journal.FormatDuration(dur)+")")
	}
	b.WriteString(timeLine + "\n\n")

	// ── Projects ──────────────────────────────────────────────────────────────
	type projGroup struct {
		name    string
		entries []journal.WorkEntry
	}
	seen := make(map[string]int)
	var groups []projGroup
	var breakEntries []journal.WorkEntry

	for _, e := range m.dayRecord.Entries {
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
			groups = append(groups, projGroup{name: proj, entries: []journal.WorkEntry{e}})
		}
	}

	// consolidateByName merges entries with the same task name (case-insensitive).
	consolidateByName := func(entries []journal.WorkEntry) []journal.WorkEntry {
		seen := make(map[string]int)
		var out []journal.WorkEntry
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

			durStr := fmt.Sprintf("%-8s", journal.FormatDuration(projTotal))
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
			durStr := fmt.Sprintf("%-8s", journal.FormatDuration(breakTotal))
			taskList := "\"" + strings.Join(names, ", ") + "\""
			b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
				"  " + breakEntryStyle.Render("☕  Breaks") +
				"  " + dayViewMutedStyle.Render(taskList) + "\n")
		}

		// ── Totals ────────────────────────────────────────────────────────────
		work, breaks, total := m.dayRecord.WorkTotals()
		b.WriteString(div + "\n")
		if breaks > 0 {
			b.WriteString(dayViewTotalsStyle.Render(fmt.Sprintf("  Work: %s  ·  Breaks: %s  ·  Total: %s",
				journal.FormatDuration(work), journal.FormatDuration(breaks), journal.FormatDuration(total))) + "\n")
		} else if work > 0 {
			b.WriteString(dayViewTotalsStyle.Render("  Total work: "+journal.FormatDuration(work)) + "\n")
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
	return eomBannerStyle.Width(m.width).Render("⚠  Last day of the month — don't forget to submit your times!")
}

func (m Model) renderStats() string {
	now := time.Now()

	// Build a set of dates that have records.
	dated := make(map[string]bool, len(m.records))
	for _, r := range m.records {
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
		case d.After(now):
			block = statsBlockFutureStyle.Render(empty)
		case !m.effectiveIsWorkDay(d):
			block = statsBlockNonWorkStyle.Render(empty)
		default:
			block = statsBlockEmptyStyle.Render(empty)
		}
		weekParts = append(weekParts, lbl+" "+block)
	}
	weekStr := strings.Join(weekParts, "  ")

	// Month count.
	monthKey := now.Format("2006-01")
	monthCount := 0
	for _, r := range m.records {
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
	for i := 0; i < streakIterLimit(m.records, now); i++ {
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
	sep := separatorStyle.Render(strings.Repeat("─", m.width))

	// Weekly work hours progress bar.
	var weekWork time.Duration
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		for _, r := range m.records {
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
			statsValueStyle.Render(journal.FormatDuration(weekWork))
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
			statsValueStyle.Render(journal.FormatDuration(weekWork)) +
			statsLabelStyle.Render(goalLabel) +
			progressBar +
			statsLabelStyle.Render(fmt.Sprintf("  %.0f%%", pct*100))
	}

	return line + "\n" + weekHoursStr + "\n" + sep
}

func (m Model) viewDateInput() string {
	header := m.renderHeader(m.appTitle(), "Open Day")
	prompt := dayViewLabelStyle.Render("Enter date:") + "  " + m.dateInput.View()
	box := formBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			formLabelStyle.Render("Open or create a journal entry for any day"),
			"",
			prompt,
			"",
			dayViewMutedStyle.Render("enter  confirm  ·  esc  cancel"),
		),
	)
	bh := lipgloss.Height(box)
	ch := m.contentHeight()
	topPad := (ch - bh) / 2
	if topPad < 0 {
		topPad = 0
	}
	pad := strings.Repeat("\n", topPad)
	footer := m.renderFooter([][2]string{{"enter", "open"}, {"esc", "cancel"}})
	return lipgloss.JoinVertical(lipgloss.Left, header, pad+box, footer)
}

func (m Model) viewList() string {
	subtitle := time.Now().Format("Mon, 02 Jan 2006")
	if m.activeWorkspace != "" {
		subtitle = m.activeWorkspace + "  ·  " + subtitle
	}
	header := m.renderHeader(m.appTitle(), subtitle)
	kb := m.cfg.Keybinds.List
	var footerKeys [][2]string
	footerKeys = append(footerKeys,
		[2]string{kb.OpenToday, "open today"},
		[2]string{kb.OpenDate, "open date"},
		[2]string{"enter", "view"},
		[2]string{kb.StatsView, "stats"},
		[2]string{kb.Delete, "delete"},
		[2]string{kb.Export, "export"},
		[2]string{"/", "filter"},
	)
	if len(m.cfg.Workspaces) > 0 {
		footerKeys = append(footerKeys, [2]string{kb.SwitchWorkspace, "workspace"})
	}
	footerKeys = append(footerKeys, [2]string{"esc", "quit"})
	footer := m.renderFooter(footerKeys)
	sections := []string{header, m.renderStats()}
	if eom := m.renderEOMBanner(); eom != "" {
		sections = append(sections, eom)
	}
	sections = append(sections, m.list.View(), footer)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) viewDayView() string {
	subtitle := m.dayRecord.Date
	if t, err := m.dayRecord.ParseDate(); err == nil {
		subtitle = t.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader(m.appTitle(), subtitle)
	tabBar := m.renderTabBar()

	var footerKeys [][2]string
	kb := m.cfg.Keybinds.Day
	if m.dayViewTab == 0 {
		clockKey := kb.ClockStart
		clockLabel := "start clock"
		if m.clockRunning {
			clockKey = kb.ClockStop
			clockLabel = "stop clock"
		}
		editLabel := "edit"
		deleteLabel := "del"
		if m.selectedPane == 1 {
			editLabel = "edit todo"
			deleteLabel = "del todo"
		}
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{"j/k", "select"},
			{"tab", "pane/indent"},
			{"S-tab", "outdent"},
			{"S-↑/↓", "reorder"},
			{kb.AddWork, "work"},
			{kb.AddBreak, "break"},
			{kb.Edit, editLabel},
			{kb.Delete, deleteLabel},
			{"backspace", "del todo"},
			{"space", "toggle todo"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{kb.Notes, "notes"},
			{kb.TodoOverview, "todo pane"},
			{clockKey, clockLabel},
			{kb.Export, "export"},
			{"esc", "back"},
		}
	} else {
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{kb.Export, "export"},
			{"esc", "back"},
		}
	}
	footer := m.renderFooter(footerKeys)
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, m.viewport.View(), footer)
}

func (m Model) viewWorkLogForm() string {
	var badge, taskLabel string
	dateStr := m.dayRecord.Date
	if m.isBreakEntry {
		badge = breakLogBadgeStyle.Render(" Log Break ")
		taskLabel = "Break label"
	} else {
		badge = workLogBadgeStyle.Render(" Log Work ")
		taskLabel = "What did you work on?"
	}
	header := m.renderHeader(m.appTitle(), badge+
		headerSubtitleStyle.Render("  "+dateStr))
	footer := m.renderFooter([][2]string{
		{"tab", "next field"},
		{"enter", "save"},
		{"esc", "cancel"},
	})

	formWidth := m.width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8

	m.taskInput.Width = inputWidth
	m.projectInput.Width = inputWidth
	m.durationInput.Width = inputWidth

	renderBox := func(input textinput.Model, active bool) string {
		if active {
			return formActiveInputStyle.Width(inputWidth).Render(input.View())
		}
		return formInactiveInputStyle.Width(inputWidth).Render(input.View())
	}

	taskBox := renderBox(m.taskInput, m.activeInput == 0)
	durBox := renderBox(m.durationInput, m.activeInput == m.numFormFields()-1)

	var body string
	if m.isBreakEntry {
		body = formLabelStyle.Render(taskLabel) + "\n" +
			taskBox + "\n\n" +
			formLabelStyle.Render("Duration") +
			formHintStyle.Render("  e.g. 1h 30m · 45m · 2h") + "\n" +
			durBox
	} else {
		projBox := renderBox(m.projectInput, m.activeInput == 1)
		body = formLabelStyle.Render(taskLabel) + "\n" +
			taskBox + "\n\n" +
			formLabelStyle.Render("Project") +
			formHintStyle.Render("  optional") + "\n" +
			projBox + "\n\n" +
			formLabelStyle.Render("Duration") +
			formHintStyle.Render("  e.g. 1h 30m · 45m · 2h") + "\n" +
			durBox
	}

	form := formBoxStyle.Width(formWidth).Render(body)

	fh := lipgloss.Height(form)
	topPad := (m.contentHeight() - fh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(form)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

func (m Model) viewClockForm() string {
	badge := workLogBadgeStyle.Render(" Start Clock ")
	dateStr := m.dayRecord.Date
	header := m.renderHeader(m.appTitle(), badge+
		headerSubtitleStyle.Render("  "+dateStr))
	footer := m.renderFooter([][2]string{
		{"tab", "next field"},
		{"enter", "start"},
		{"esc", "cancel"},
	})

	formWidth := m.width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8

	m.taskInput.Width = inputWidth
	m.projectInput.Width = inputWidth

	renderBox := func(input textinput.Model, active bool) string {
		if active {
			return formActiveInputStyle.Width(inputWidth).Render(input.View())
		}
		return formInactiveInputStyle.Width(inputWidth).Render(input.View())
	}

	body := formLabelStyle.Render("What are you working on?") + "\n" +
		renderBox(m.taskInput, m.activeInput == 0) + "\n\n" +
		formLabelStyle.Render("Project") +
		formHintStyle.Render("  optional · comma-separate for multiple projects") + "\n" +
		renderBox(m.projectInput, m.activeInput == 1)

	form := formBoxStyle.Width(formWidth).Render(body)

	fh := lipgloss.Height(form)
	topPad := (m.contentHeight() - fh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(form)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

func (m Model) viewTimeInput() string {
	label := "Set Start Time"
	badge := workLogBadgeStyle.Render(" Start ")
	if !m.timeInputStart {
		label = "Set Finish Time"
		badge = breakLogBadgeStyle.Render(" Finish ")
	}
	header := m.renderHeader(m.appTitle(), badge)
	footer := m.renderFooter([][2]string{
		{"enter", "confirm"},
		{"r", "reset"},
		{"esc", "cancel"},
	})

	m.timeInput.Width = 12
	inputBox := formActiveInputStyle.Width(14).Render(m.timeInput.View())

	dialog := formBoxStyle.Render(
		formLabelStyle.Render(label) + "\n" +
			formHintStyle.Render("24-hour format  ·  e.g. 09:30, 14:00") + "\n\n" +
			inputBox,
	)

	dh := lipgloss.Height(dialog)
	topPad := (m.contentHeight() - dh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(dialog)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

func (m Model) viewNotesEditor() string {
	subtitle := m.dayRecord.Date
	if t, err := m.dayRecord.ParseDate(); err == nil {
		subtitle = t.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader(m.appTitle(), subtitle)
	footer := m.renderFooter([][2]string{
		{"ctrl+s", "save"},
		{"esc", "cancel"},
	})
	editor := editorBorderStyle.
		Width(m.width - 4).
		Render(m.textarea.View())
	return lipgloss.JoinVertical(lipgloss.Left, header, editor, footer)
}

func (m Model) viewTodoForm() string {
	header := m.renderHeader(m.appTitle(), "Todo")
	footer := m.renderFooter([][2]string{
		{"enter", "save"},
		{"esc", "cancel"},
	})
	formWidth := m.width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8
	m.todoInput.Width = inputWidth
	box := formBoxStyle.Width(formWidth).Render(
		formLabelStyle.Render("Todo title") + "\n" +
			formActiveInputStyle.Width(inputWidth).Render(m.todoInput.View()) + "\n\n" +
			dayViewMutedStyle.Render("Tip: press Shift+a from a parent todo to add a subtodo"),
	)
	fh := lipgloss.Height(box)
	topPad := (m.contentHeight() - fh) / 2
	if topPad < 0 {
		topPad = 0
	}
	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

func (m Model) viewConfirmDelete() string {
	var subject string
	if m.deleteDay {
		if m.prevState == stateDayView {
			subject = m.dayRecord.Date
		} else if m.deleteIdx >= 0 && m.deleteIdx < len(m.records) {
			subject = m.records[m.deleteIdx].Date
		}
		subject = "the day " + subject
	} else if m.deleteIdx == deleteTodoIdx {
		if m.selectedTodo >= 0 && m.selectedTodo < len(m.workspaceTodos) {
			if m.selectedSub >= 0 && m.selectedSub2 >= 0 &&
				m.selectedSub < len(m.workspaceTodos[m.selectedTodo].Subtodos) &&
				m.selectedSub2 < len(m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos) {
				subject = `todo "` + m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos[m.selectedSub2].Title + `"`
			} else if m.selectedSub >= 0 && m.selectedSub < len(m.workspaceTodos[m.selectedTodo].Subtodos) {
				subject = `todo "` + m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Title + `"`
			} else {
				subject = `todo "` + m.workspaceTodos[m.selectedTodo].Title + `"`
			}
		} else {
			subject = "this todo"
		}
	} else {
		if m.deleteIdx >= 0 && m.deleteIdx < len(m.dayRecord.Entries) {
			subject = `entry "` + m.dayRecord.Entries[m.deleteIdx].Task + `"`
		} else {
			subject = "this entry"
		}
	}
	header := m.renderHeader(m.appTitle(), "Delete")

	dialog := confirmBoxStyle.Render(
		confirmTitleStyle.Render(fmt.Sprintf("Delete %s?", subject)) +
			"\n\n  " +
			confirmYesStyle.Render("[y]") + helpStyle.Render(" yes") +
			"    " +
			confirmNoStyle.Render("[n]") + helpStyle.Render(" no / esc"),
	)

	dh := lipgloss.Height(dialog)
	topPad := (m.contentHeight() - dh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(dialog)
	return header + "\n" + strings.Repeat("\n", topPad) + centered
}

func (m Model) viewWorkspacePicker() string {
	header := m.renderHeader(m.appTitle(), "Switch Workspace")
	innerW := 36
	if m.width-8 > innerW {
		innerW = m.width / 2
	}
	if innerW > 60 {
		innerW = 60
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))
	var rows []string
	rows = append(rows, formLabelStyle.Render("Select a workspace:"))
	rows = append(rows, div)
	for i, ws := range m.cfg.Workspaces {
		cursor := "  "
		if i == m.workspaceIdx {
			cursor = "▶ "
		}
		label := ws.Name
		if ws.Name == m.activeWorkspace {
			label += "  " + statusSuccessStyle.Render("✓")
		}
		line := cursor + label
		if i == m.workspaceIdx {
			line = selectedEntryStyle.Width(innerW).Render(line)
		} else {
			line = normalEntryStyle.Render(line)
		}
		rows = append(rows, line)
	}
	rows = append(rows, div)
	rows = append(rows, dayViewMutedStyle.Render("j/k  navigate  ·  enter  switch  ·  esc  cancel"))
	box := formBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	bh := lipgloss.Height(box)
	ch := m.contentHeight()
	topPad := (ch - bh) / 2
	if topPad < 0 {
		topPad = 0
	}
	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
	footer := m.renderFooter([][2]string{{"j/k", "navigate"}, {"enter", "switch"}, {"esc", "cancel"}})
	return lipgloss.JoinVertical(lipgloss.Left, header, strings.Repeat("\n", topPad)+centered, footer)
}

// statsDurFieldW is the fixed column width used for rendered durations in the
// stats view (e.g. "10h30m   ") so bars and columns align across rows.
const statsDurFieldW = 9

func (m Model) viewStats() string {
	header := m.renderHeader("📊  Stats", "Overview")
	tabBar := m.renderStatsTabBar()
	footer := m.renderFooter([][2]string{
		{"←/→", "switch tab"},
		{"j/k", "scroll"},
		{"esc", "back"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, m.viewport.View(), footer)
}

func (m Model) renderStatsTabBar() string {
	tabs := []string{"🔥  Overview", "📆  Monthly", "📈  Yearly", "🏆  All-time"}
	var parts []string
	for i, label := range tabs {
		if i == m.statsTab {
			parts = append(parts, activeTabStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+label+" "))
		}
	}
	bar := strings.Join(parts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.width))
	return bar + "\n" + sep
}

// renderStatsTabContent dispatches to the appropriate tab renderer.
func (m Model) renderStatsTabContent() string {
	switch m.statsTab {
	case 1:
		return m.renderStatsMonthly()
	case 2:
		return m.renderStatsYearly()
	case 3:
		return m.renderStatsAllTime()
	default:
		return m.renderStatsOverview()
	}
}

// renderStatsOverview renders the Overview tab: streaks + 16-week activity heatmap.
func (m Model) renderStatsOverview() string {
	now := time.Now()
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	// Build a set of dates for which records exist.
	dated := make(map[string]bool, len(m.records))
	for _, r := range m.records {
		dated[r.Date] = true
	}

	var b strings.Builder
	b.WriteString("\n")

	// ── Streaks ───────────────────────────────────────────────────────────────
	b.WriteString("  " + dayViewSectionStyle.Render("Streaks") + "\n")
	b.WriteString("  " + div + "\n")

	// Current streak: consecutive working days going back from today.
	// Non-working days are skipped so that a weekend never breaks the count.
	// The iteration limit is derived from the oldest record so the loop is
	// always bounded by real data and never underestimates long streaks.
	currentStreak := 0
	for i := 0; i < streakIterLimit(m.records, now); i++ {
		check := now.AddDate(0, 0, -i)
		dateStr := check.Format("2006-01-02")
		if dated[dateStr] {
			currentStreak++
		} else if m.effectiveIsWorkDay(check) {
			// A working day with no entry breaks the streak.
			break
		}
		// Non-working day without an entry: continue (streak passes through).
	}

	// Longest streak: scan all dated records.
	longestStreak := 0
	if len(m.records) > 0 {
		var days []time.Time
		for _, r := range m.records {
			if t, err := r.ParseDate(); err == nil {
				days = append(days, t)
			}
		}
		sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
		if len(days) > 0 {
			run := 1
			for i := 1; i < len(days); i++ {
				// Fast path: directly adjacent days are always consecutive.
				// Otherwise check that every day in the gap is a non-working day
				// (e.g. Fri → Mon across a weekend).
				gapOK := days[i-1].AddDate(0, 0, 1).Equal(days[i])
				if !gapOK {
					gapOK = true
					for d := days[i-1].AddDate(0, 0, 1); d.Before(days[i]); d = d.AddDate(0, 0, 1) {
						if m.effectiveIsWorkDay(d) {
							gapOK = false
							break
						}
					}
				}
				if gapOK {
					run++
					if run > longestStreak {
						longestStreak = run
					}
				} else {
					if run > longestStreak {
						longestStreak = run
					}
					run = 1
				}
			}
			if run > longestStreak {
				longestStreak = run
			}
		}
	}

	totalDays := len(m.records)
	var streakLine string
	if currentStreak > 0 {
		streakLine = "  🔥 " + statsStreakStyle.Render(fmt.Sprintf("%d", currentStreak)) +
			statsLabelStyle.Render(" day streak")
	} else {
		streakLine = "  " + statsLabelStyle.Render("No active streak")
	}
	streakLine += statsLabelStyle.Render("  ·  Longest: ") +
		statsValueStyle.Render(fmt.Sprintf("%d", longestStreak)) +
		statsLabelStyle.Render(" days") +
		statsLabelStyle.Render("  ·  Total logged: ") +
		statsValueStyle.Render(fmt.Sprintf("%d", totalDays)) +
		statsLabelStyle.Render(" days")
	b.WriteString(streakLine + "\n\n")

	// ── Activity heatmap (last 16 weeks) ──────────────────────────────────────
	b.WriteString("  " + dayViewSectionStyle.Render("Activity  (last 16 weeks)") + "\n")
	b.WriteString("  " + div + "\n")

	// Find Monday of current week.
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	thisMonday := now.AddDate(0, 0, -(weekday - 1))

	const heatmapWeeks = 16
	const filled = "▓"
	const empty = "░"

	// Header row: day initials.
	dayInitials := [7]string{"M", "T", "W", "T", "F", "S", "S"}
	var headerCols []string
	headerCols = append(headerCols, "        ") // month label column width
	for _, lbl := range dayInitials {
		headerCols = append(headerCols, statsLabelStyle.Render(lbl))
	}
	b.WriteString("  " + strings.Join(headerCols, " ") + "\n")

	for w := heatmapWeeks - 1; w >= 0; w-- {
		monday := thisMonday.AddDate(0, 0, -w*7)
		// Month label for this row (show if first week or month changes).
		var monthLabel string
		if w == heatmapWeeks-1 || monday.Month() != monday.AddDate(0, 0, -7).Month() {
			monthLabel = monday.Format("Jan")
		}
		row := fmt.Sprintf("  %-6s  ", monthLabel)
		for i := 0; i < 7; i++ {
			d := monday.AddDate(0, 0, i)
			dateStr := d.Format("2006-01-02")
			var block string
			switch {
			case dated[dateStr]:
				block = statsBlockFilledStyle.Render(filled)
			case d.After(now):
				block = statsBlockFutureStyle.Render(empty)
			case !m.effectiveIsWorkDay(d):
				block = statsBlockNonWorkStyle.Render(empty)
			default:
				block = statsBlockEmptyStyle.Render(empty)
			}
			if i > 0 {
				row += " "
			}
			row += block
		}
		b.WriteString(row + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderStatsMonthly renders the Monthly tab.
func (m Model) renderStatsMonthly() string {
	now := time.Now()
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	monthKey := now.Format("2006-01")
	monthName := now.Format("January 2006")

	projMap := make(map[string]time.Duration)
	var totalWork time.Duration
	days := 0
	for _, r := range m.records {
		if !strings.HasPrefix(r.Date, monthKey) {
			continue
		}
		days++
		for _, e := range r.Entries {
			if e.IsBreak {
				continue
			}
			proj := e.Project
			if proj == "" {
				proj = "Other"
			}
			projMap[proj] += e.Duration()
			totalWork += e.Duration()
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dayViewSectionStyle.Render("This Month") +
		dayViewMutedStyle.Render("  "+monthName) +
		"    " +
		statsLabelStyle.Render("Total: ") + statsValueStyle.Render(journal.FormatDuration(totalWork)) +
		statsLabelStyle.Render(fmt.Sprintf("  ·  %d days", days)) +
		"\n")
	b.WriteString("  " + div + "\n")
	if len(projMap) == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No work entries this month") + "\n")
	} else {
		b.WriteString(renderProjectBars(projMap, totalWork))
	}
	return b.String()
}

// renderStatsYearly renders the Yearly tab.
func (m Model) renderStatsYearly() string {
	now := time.Now()
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	yearKey := now.Format("2006")

	projMap := make(map[string]time.Duration)
	monthlyWork := make(map[string]time.Duration)
	var totalWork time.Duration
	days := 0
	for _, r := range m.records {
		if !strings.HasPrefix(r.Date, yearKey) {
			continue
		}
		days++
		mk := r.Date[:7]
		for _, e := range r.Entries {
			if e.IsBreak {
				continue
			}
			proj := e.Project
			if proj == "" {
				proj = "Other"
			}
			projMap[proj] += e.Duration()
			totalWork += e.Duration()
			monthlyWork[mk] += e.Duration()
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dayViewSectionStyle.Render("This Year") +
		dayViewMutedStyle.Render("  "+yearKey) +
		"    " +
		statsLabelStyle.Render("Total: ") + statsValueStyle.Render(journal.FormatDuration(totalWork)) +
		statsLabelStyle.Render(fmt.Sprintf("  ·  %d days", days)) +
		"\n")
	b.WriteString("  " + div + "\n")

	if totalWork == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No work entries this year") + "\n")
		return b.String()
	}

	// Month-by-month bar chart (relative to busiest month).
	var maxMonthWork time.Duration
	for _, d := range monthlyWork {
		if d > maxMonthWork {
			maxMonthWork = d
		}
	}
	const monthBarW = 16
	for mo := 1; mo <= int(now.Month()); mo++ {
		mk := fmt.Sprintf("%s-%02d", yearKey, mo)
		d := monthlyWork[mk]
		moLabel := time.Month(mo).String()[:3]
		pct := float64(d) / float64(maxMonthWork)
		filledCount := int(pct * monthBarW)
		bar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
			statsBlockEmptyStyle.Render(strings.Repeat("░", monthBarW-filledCount))
		durStr := fmt.Sprintf("%-*s", statsDurFieldW, journal.FormatDuration(d))
		b.WriteString("  " + statsLabelStyle.Render(moLabel+"  ") +
			bar + "  " +
			statsValueStyle.Render(durStr) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + div + "\n")
	b.WriteString(renderProjectBars(projMap, totalWork))
	return b.String()
}

// renderStatsAllTime renders the All-time tab.
func (m Model) renderStatsAllTime() string {
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	projMap := make(map[string]time.Duration)
	var totalWork time.Duration
	for _, r := range m.records {
		for _, e := range r.Entries {
			if e.IsBreak {
				continue
			}
			proj := e.Project
			if proj == "" {
				proj = "Other"
			}
			projMap[proj] += e.Duration()
			totalWork += e.Duration()
		}
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + dayViewSectionStyle.Render("All-time Top Projects") +
		"    " +
		statsLabelStyle.Render("Total: ") + statsValueStyle.Render(journal.FormatDuration(totalWork)) +
		"\n")
	b.WriteString("  " + div + "\n")
	if len(projMap) == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No work entries") + "\n")
	} else {
		b.WriteString(renderProjectBars(projMap, totalWork))
	}
	return b.String()
}

// renderProjectBars renders a sorted list of project durations with inline bar
// charts relative to the total work duration. All bars start at the same column.
func renderProjectBars(projMap map[string]time.Duration, total time.Duration) string {
	type ps struct {
		name string
		dur  time.Duration
	}
	var projects []ps
	for name, dur := range projMap {
		projects = append(projects, ps{name, dur})
	}
	// Sort descending by duration.
	sort.Slice(projects, func(i, j int) bool { return projects[i].dur > projects[j].dur })

	// Find the widest rendered project name so all bars start at the same column.
	maxNameW := 0
	for _, p := range projects {
		if w := lipgloss.Width(p.name); w > maxNameW {
			maxNameW = w
		}
	}

	const barW = 20
	var sb strings.Builder
	for _, p := range projects {
		pct := float64(0)
		if total > 0 {
			pct = float64(p.dur) / float64(total)
		}
		filledCount := int(pct * barW)
		bar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
			statsBlockEmptyStyle.Render(strings.Repeat("░", barW-filledCount))
		durStr := fmt.Sprintf("%-*s", statsDurFieldW, journal.FormatDuration(p.dur))
		pctStr := fmt.Sprintf("%3.0f%%", pct*100)
		// Width() on the style pads the name to a uniform column width.
		nameStr := dayViewSectionStyle.Width(maxNameW).Render(p.name)
		sb.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
			"  " + nameStr +
			"  " + bar +
			"  " + statsLabelStyle.Render(pctStr) + "\n")
	}
	return sb.String()
}
