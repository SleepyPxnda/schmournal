package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sleepypxnda/schmournal/journal"
)

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
	case stateTimeInput:
		return m.viewTimeInput()
	case stateNotesEditor:
		return m.viewNotesEditor()
	case stateConfirmDelete:
		return m.viewConfirmDelete()
	case stateDateInput:
		return m.viewDateInput()
	case stateWeekView:
		return m.viewWeekView()
	case stateWeekHoursInput:
		return m.viewWeekHoursInput()
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

	// ── Work Day section ──────────────────────────────────────────────────────
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

	// ── Work Log section ──────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("📋  Work Log") + "\n")
	b.WriteString(div + "\n")

	entries := m.dayRecord.Entries
	if len(entries) == 0 {
		b.WriteString(dayViewMutedStyle.Render("  No entries yet") + "\n")
	} else {
		// column widths: selector(2) + project(14) + task(dynamic) + duration(8)
		taskW := innerW - 2 - 14 - 8
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
			taskStr = fmt.Sprintf("%-*s", taskW, taskStr)
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
	totals := ""
	if work > 0 || breaks > 0 {
		totals = "  Work: " + journal.FormatDuration(work)
		if breaks > 0 {
			totals += "  ·  Breaks: " + journal.FormatDuration(breaks)
			totals += "  ·  Total: " + journal.FormatDuration(work+breaks)
		}
	}
	if totals != "" {
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
			warn := "  ⚠  Logged time (" + journal.FormatDuration(logged) + ") differs from day span (" +
				journal.FormatDuration(dayDur) + ") by " + sign + journal.FormatDuration(diff)
			b.WriteString(dayViewWarnStyle.Render(warn) + "\n")
		}
	}

	b.WriteString("\n")

	// ── Notes section ─────────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("📝  Notes") + "\n")
	b.WriteString(div + "\n")
	if m.dayRecord.Notes == "" {
		b.WriteString(dayViewMutedStyle.Render("  No notes") + "\n")
	} else {
		for _, line := range strings.Split(m.dayRecord.Notes, "\n") {
			b.WriteString(dayViewNotesStyle.Render("  "+line) + "\n")
		}
	}

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

	// Streak: consecutive days going back from today.
	streak := 0
	for check := now; ; check = check.AddDate(0, 0, -1) {
		if !dated[check.Format("2006-01-02")] {
			break
		}
		streak++
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
	// Use the per-week override (if any) for the current week, else global default.
	currentWeekKey := monday.Format("2006-01-02")
	goalHours := m.weeklyGoalFor(currentWeekKey)
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
	weekHoursStr := "  " + statsLabelStyle.Render("Week: ") +
		statsValueStyle.Render(journal.FormatDuration(weekWork)) +
		statsLabelStyle.Render(goalLabel) +
		progressBar +
		statsLabelStyle.Render(fmt.Sprintf("  %.0f%%", pct*100))

	return line + "\n" + weekHoursStr + "\n" + sep
}

func (m Model) viewDateInput() string {
	header := m.renderHeader("📔  Schmournal", "Open Day")
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
	header := m.renderHeader("📔  Schmournal", time.Now().Format("Mon, 02 Jan 2006"))
	kb := m.cfg.Keybinds.List
	footer := m.renderFooter([][2]string{
		{kb.OpenToday, "open today"},
		{kb.OpenDate, "open date"},
		{kb.AddWork, "log work"},
		{kb.AddBreak, "log break"},
		{"enter", "view"},
		{kb.WeekView, "week"},
		{kb.Delete, "delete"},
		{kb.Export, "export"},
		{"/", "filter"},
		{"esc", "quit"},
	})
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
	header := m.renderHeader("📔  Schmournal", subtitle)
	tabBar := m.renderTabBar()

	var footerKeys [][2]string
	kb := m.cfg.Keybinds.Day
	if m.dayViewTab == 0 {
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{"j/k", "select"},
			{kb.AddWork, "work"},
			{kb.AddBreak, "break"},
			{kb.Edit, "edit"},
			{kb.Delete, "del"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{kb.Notes, "notes"},
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
	header := m.renderHeader("📔  Schmournal", badge+
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

func (m Model) viewTimeInput() string {
	label := "Set Start Time"
	badge := workLogBadgeStyle.Render(" Start ")
	if !m.timeInputStart {
		label = "Set Finish Time"
		badge = breakLogBadgeStyle.Render(" Finish ")
	}
	header := m.renderHeader("📔  Schmournal", badge)
	footer := m.renderFooter([][2]string{
		{"enter", "confirm"},
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

func (m Model) viewWeekHoursInput() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)
	sunday := monday.AddDate(0, 0, 6)
	weekRange := monday.Format("02 Jan") + " – " + sunday.Format("02 Jan 2006")

	header := m.renderHeader("📅  This Week", weekRange)
	footer := m.renderFooter([][2]string{
		{"enter", "confirm"},
		{"esc", "cancel"},
	})

	m.weekHoursInput.Width = 12
	inputBox := formActiveInputStyle.Width(14).Render(m.weekHoursInput.View())

	hint := fmt.Sprintf("hours  ·  global default: %gh  ·  leave empty to reset", m.cfg.WeeklyHoursGoal)
	dialog := formBoxStyle.Render(
		formLabelStyle.Render("Set Weekly Hours Goal") + "\n" +
			formHintStyle.Render(hint) + "\n\n" +
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
	header := m.renderHeader("📔  Schmournal", subtitle)
	footer := m.renderFooter([][2]string{
		{"ctrl+s", "save"},
		{"esc", "cancel"},
	})
	editor := editorBorderStyle.
		Width(m.width - 4).
		Render(m.textarea.View())
	return lipgloss.JoinVertical(lipgloss.Left, header, editor, footer)
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
	} else {
		if m.deleteIdx >= 0 && m.deleteIdx < len(m.dayRecord.Entries) {
			subject = `entry "` + m.dayRecord.Entries[m.deleteIdx].Task + `"`
		} else {
			subject = "this entry"
		}
	}
	header := m.renderHeader("📔  Schmournal", "Delete")

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

func (m Model) viewWeekView() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)
	sunday := monday.AddDate(0, 0, 6)

	weekRange := monday.Format("02 Jan") + " – " + sunday.Format("02 Jan 2006")
	header := m.renderHeader("📅  This Week", weekRange)

	// Navigation hint bar (matches day-view tab bar height: 2 lines).
	var navParts []string
	navParts = append(navParts, inactiveTabStyle.Render("← prev week"))
	if m.weekOffset < 0 {
		navParts = append(navParts, inactiveTabStyle.Render("→ next week"))
	}
	navBar := strings.Join(navParts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.width))
	subHeader := navBar + "\n" + sep

	kb := m.cfg.Keybinds.Week
	footer := m.renderFooter([][2]string{
		{"j/k", "scroll"},
		{"←/→", "prev/next week"},
		{kb.SetWeeklyHours, "set week goal"},
		{"esc", "back"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, subHeader, m.viewport.View(), footer)
}

func (m Model) renderWeekContent() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)

	// Build date→record map for quick lookup.
	recByDate := make(map[string]journal.DayRecord, len(m.records))
	for _, r := range m.records {
		recByDate[r.Date] = r
	}

	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	var weekWork time.Duration

	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		rec, hasRec := recByDate[dateStr]

		var work, breaks time.Duration
		if hasRec {
			work, breaks, _ = rec.WorkTotals()
		}
		weekWork += work

		// Day header line.
		dayLabel := d.Format("Mon  02 Jan 2006")
		var headerLine string
		if hasRec && (work+breaks) > 0 {
			headerLine = dayViewSectionStyle.Render(dayLabel) +
				dayViewMutedStyle.Render("  ·  ") +
				dayViewValueStyle.Render(journal.FormatDuration(work)+" work")
			if breaks > 0 {
				headerLine += dayViewMutedStyle.Render("  ·  " + journal.FormatDuration(breaks) + " breaks")
			}
		} else {
			headerLine = dayViewSectionStyle.Render(dayLabel) +
				dayViewMutedStyle.Render("  ·  no entries")
		}
		b.WriteString(headerLine + "\n")

		// Start / end time sub-line (only shown when at least one is set).
		if hasRec && (rec.StartTime != "" || rec.EndTime != "") {
			start := rec.StartTime
			end := rec.EndTime
			startStr := dayViewValueStyle.Render(start)
			if start == "" {
				startStr = dayViewMutedStyle.Render("—")
			}
			endStr := dayViewValueStyle.Render(end)
			if end == "" {
				endStr = dayViewMutedStyle.Render("—")
			}
			timeLine := "  " + dayViewLabelStyle.Render("Start:") + " " + startStr +
				"   " + dayViewLabelStyle.Render("End:") + " " + endStr
			if dur, ok := rec.DayDuration(); ok {
				timeLine += "   " + dayViewMutedStyle.Render("("+journal.FormatDuration(dur)+")")
			}
			b.WriteString(timeLine + "\n")
		}
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
					// Collect breaks under a single "Breaks" group.
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
				durStr := fmt.Sprintf("%-8s", journal.FormatDuration(g.dur))
				taskStr := `"` + strings.Join(g.tasks, ", ") + `"`
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

	// Week total + progress bar.
	b.WriteString("\n" + div + "\n")
	goalHours := m.weeklyGoal()
	weeklyGoal := time.Duration(goalHours * float64(time.Hour))
	pct := float64(weekWork) / float64(weeklyGoal)
	if pct > 1 {
		pct = 1
	}
	goalLabel := fmt.Sprintf(" / %gh  ", goalHours)
	// Annotate if this week uses a custom override.
	if _, hasOverride := m.weekGoals[m.weekKey()]; hasOverride {
		goalLabel = fmt.Sprintf(" / %gh (custom)  ", goalHours)
	}
	const barW = 24
	filledCount := int(pct * barW)
	bar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
		statsBlockEmptyStyle.Render(strings.Repeat("░", barW-filledCount))
	totalLine := "  " + dayViewLabelStyle.Render("Week total: ") +
		dayViewValueStyle.Render(journal.FormatDuration(weekWork)) +
		dayViewMutedStyle.Render(goalLabel) +
		bar +
		dayViewMutedStyle.Render(fmt.Sprintf("  %.0f%%", pct*100))
	b.WriteString(totalLine + "\n")

	return b.String()
}

// uniqueAppend appends s to slice only if not already present (case-sensitive).
func uniqueAppend(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
