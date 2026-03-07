package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/config"
	"github.com/sleepypxnda/schmournal/journal"
)

// weekKey returns the key used to store per-week goal overrides.
// It is the Monday date of the week currently shown (format "YYYY-MM-DD").
func (m Model) weekKey() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)
	return monday.Format("2006-01-02")
}

// weeklyGoalFor returns the effective hours goal for the week identified by
// mondayKey ("YYYY-MM-DD" of the week's Monday): per-week override if set,
// otherwise the workspace-specific goal (or the global config default).
func (m Model) weeklyGoalFor(mondayKey string) float64 {
	if h, ok := m.weekGoals[mondayKey]; ok && h > 0 {
		return h
	}
	return m.effectiveWeeklyHoursGoal()
}

// activeWorkspaceConfig returns a pointer to the active WorkspaceConfig, or nil
// when no workspaces are configured or none matches the active name.
func (m Model) activeWorkspaceConfig() *config.WorkspaceConfig {
	for i := range m.cfg.Workspaces {
		if m.cfg.Workspaces[i].Name == m.activeWorkspace {
			return &m.cfg.Workspaces[i]
		}
	}
	return nil
}

// effectiveWeeklyHoursGoal returns the hours-per-week goal for the active
// workspace, falling back to the global config default when not overridden.
func (m Model) effectiveWeeklyHoursGoal() float64 {
	if ws := m.activeWorkspaceConfig(); ws != nil && ws.WeeklyHoursGoal > 0 {
		return ws.WeeklyHoursGoal
	}
	return m.cfg.WeeklyHoursGoal
}

// openWorkspacePicker opens the workspace picker dialog. If no workspaces are
// configured a status message is shown instead.
func (m Model) openWorkspacePicker() (tea.Model, tea.Cmd) {
	if len(m.cfg.Workspaces) == 0 {
		m.statusMsg = "No workspaces configured — add [[workspaces]] entries to your config file"
		m.isError = false
		return m, clearStatusCmd()
	}
	// Pre-select the currently active workspace.
	m.workspaceIdx = 0
	for i, ws := range m.cfg.Workspaces {
		if ws.Name == m.activeWorkspace {
			m.workspaceIdx = i
			break
		}
	}
	m.state = stateWorkspacePicker
	return m, nil
}

// switchWorkspace applies the named workspace: updates the journal storage
// path, records the active workspace name and reloads all data.
func (m Model) switchWorkspace(name string) (tea.Model, tea.Cmd) {
	storagePath := m.cfg.StoragePath
	for _, ws := range m.cfg.Workspaces {
		if ws.Name == name {
			if ws.StoragePath != "" {
				storagePath = ws.StoragePath
			}
			break
		}
	}
	if err := journal.SetStoragePath(storagePath); err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		m.state = stateList
		return m, clearStatusCmd()
	}
	m.activeWorkspace = name
	m.state = stateList
	m.statusMsg = fmt.Sprintf("✓ Switched to workspace %q", name)
	m.isError = false
	return m, tea.Batch(
		loadRecords,
		loadWeeklyGoals,
		clearStatusCmd(),
		func() tea.Msg {
			_ = config.SaveState(config.AppState{ActiveWorkspace: name})
			return nil
		},
	)
}

// weeklyGoal returns the effective hours goal for the currently displayed week.
func (m Model) weeklyGoal() float64 {
	return m.weeklyGoalFor(m.weekKey())
}

func (m Model) openWeekHoursInput() (tea.Model, tea.Cmd) {
	current := m.weeklyGoal()
	m.weekHoursInput.SetValue(fmt.Sprintf("%g", current))
	m.weekHoursInput.Placeholder = fmt.Sprintf("%g", m.effectiveWeeklyHoursGoal())
	cmd := m.weekHoursInput.Focus()
	m.state = stateWeekHoursInput
	return m, cmd
}

func (m Model) openDayView(rec journal.DayRecord) (tea.Model, tea.Cmd) {
	// Reload from disk to get freshest data.
	fresh, err := journal.Load(rec.Path)
	if err != nil {
		fresh = rec
	}
	m.dayRecord = fresh
	if m.dayRecord.Path == "" {
		m.dayRecord.Path = rec.Path
	}
	m.dayViewTab = 0
	m.selectedEntry = -1
	if len(m.dayRecord.Entries) > 0 {
		m.selectedEntry = 0
	}
	m.state = stateDayView
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderDayContent())
	return m, nil
}

func (m Model) openWeekView() (tea.Model, tea.Cmd) {
	m.weekOffset = 0
	m.state = stateWeekView
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderWeekContent())
	return m, nil
}

func (m Model) openStatsView() (tea.Model, tea.Cmd) {
	m.state = stateStats
	m.statsTab = 0
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderStatsTabContent())
	return m, nil
}

func (m Model) openDayViewToday() (tea.Model, tea.Cmd) {
	path, err := journal.TodayPath()
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	rec, err := journal.Load(path)
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	if rec.Date == "" {
		rec.Date = time.Now().Format("2006-01-02")
	}
	rec.Path = path
	return m.openDayView(rec)
}

func (m Model) openWorkForm(isBreak bool, editIdx int) (tea.Model, tea.Cmd) {
	m.isBreakEntry = isBreak
	m.editEntryIdx = editIdx
	m.taskInput.SetValue("")
	m.projectInput.SetValue("")
	m.durationInput.SetValue("")
	if editIdx >= 0 && editIdx < len(m.dayRecord.Entries) {
		e := m.dayRecord.Entries[editIdx]
		m.taskInput.SetValue(e.Task)
		m.projectInput.SetValue(e.Project)
		m.durationInput.SetValue(journal.FormatDuration(e.Duration()))
	}
	if isBreak {
		m.taskInput.Placeholder = "e.g. Lunch, coffee break, walk…"
	} else {
		m.taskInput.Placeholder = "e.g. Feature development, meeting, code review…"
	}
	m.state = stateWorkForm
	return m.focusField(0)
}

func (m Model) openWorkFormForToday(isBreak bool) (tea.Model, tea.Cmd) {
	path, err := journal.TodayPath()
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	rec, err := journal.Load(path)
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	if rec.Date == "" {
		rec.Date = time.Now().Format("2006-01-02")
	}
	rec.Path = path
	m.dayRecord = rec
	if m.selectedEntry < 0 || m.selectedEntry >= len(rec.Entries) {
		m.selectedEntry = len(rec.Entries) - 1
	}
	// After form submission we'll land in stateDayView.
	return m.openWorkForm(isBreak, -1)
}

func (m Model) openNotesEditor() (tea.Model, tea.Cmd) {
	m.textarea.SetValue(m.dayRecord.Notes)
	blinkCmd := m.textarea.Focus()
	m.state = stateNotesEditor
	return m, blinkCmd
}

func (m Model) openTimeInput(isStart bool) (tea.Model, tea.Cmd) {
	m.timeInputStart = isStart
	existing := m.dayRecord.StartTime
	if !isStart {
		existing = m.dayRecord.EndTime
	}
	if existing != "" {
		m.timeInput.SetValue(existing)
	} else {
		m.timeInput.SetValue(time.Now().Format("15:04"))
	}
	m.timeInput.CursorEnd()
	m.state = stateTimeInput
	return m, m.timeInput.Focus()
}

func (m Model) openDateInput() (tea.Model, tea.Cmd) {
	m.dateInput.SetValue("")
	m.dateInput.Placeholder = time.Now().Format("2006-01-02")
	cmd := m.dateInput.Focus()
	m.state = stateDateInput
	return m, cmd
}

// openClockForm opens the task-entry form used to start the clock timer.
func (m Model) openClockForm() (tea.Model, tea.Cmd) {
	m.isBreakEntry = false
	m.taskInput.SetValue("")
	m.projectInput.SetValue("")
	m.taskInput.Placeholder = "e.g. Feature development, meeting, code review…"
	m.projectInput.Placeholder = "e.g. Backend, Frontend  (optional, comma-separated)"
	m.state = stateClockForm
	return m.focusField(0)
}

// stopClock stops the running clock, computes the elapsed duration, creates
// the appropriate WorkEntry values (splitting across projects if needed) and
// saves the day record. If the elapsed time rounds down to zero minutes a
// status message is shown instead.
func (m Model) stopClock() (tea.Model, tea.Cmd) {
	elapsed := time.Since(m.clockStart)
	entries := journal.ClockEntries(m.clockTask, m.clockProject, elapsed)

	m.clockRunning = false
	m.clockTask = ""
	m.clockProject = ""

	if len(entries) == 0 {
		m.statusMsg = "✗ Clock stopped — duration too short to log (< 1 minute)"
		m.isError = true
		m.state = stateDayView
		m.dayViewTab = 0
		m.viewport.SetContent(m.renderDayContent())
		return m, clearStatusCmd()
	}

	m.dayRecord.Entries = append(m.dayRecord.Entries, entries...)
	m.selectedEntry = len(m.dayRecord.Entries) - 1
	m.state = stateDayView
	m.dayViewTab = 0 // switch to Work Log so the new entry is visible
	m.viewport.SetContent(m.renderDayContent())
	m.scrollToSelected()

	label := "✓ Clocked entry logged"
	if len(entries) > 1 {
		label = "✓ Clocked entries split across projects"
	}
	return m, m.saveDayCmd(label)
}

// numFormFields returns how many fields the current work-log form has.
func (m Model) numFormFields() int {
	if m.isBreakEntry {
		return 2
	}
	return 3
}

// focusField blurs all inputs and focuses the one at index n.
func (m Model) focusField(n int) (Model, tea.Cmd) {
	m.taskInput.Blur()
	m.projectInput.Blur()
	m.durationInput.Blur()
	m.activeInput = n
	switch {
	case n == 0:
		return m, m.taskInput.Focus()
	case n == 1 && !m.isBreakEntry:
		return m, m.projectInput.Focus()
	default:
		return m, m.durationInput.Focus()
	}
}

func isValidHHMM(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}

// scrollToSelected adjusts the viewport so the selected entry is visible.
func (m *Model) scrollToSelected() {
	const entryStartLine = 7
	if m.selectedEntry < 0 {
		return
	}
	targetLine := entryStartLine + m.selectedEntry
	if targetLine < m.viewport.YOffset {
		m.viewport.YOffset = targetLine
	} else if targetLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = targetLine - m.viewport.Height + 1
	}
}

// copyWeeklyGoals returns a shallow copy of goals so that async tea.Cmd
// closures can safely marshal it without racing against future Update calls.
func copyWeeklyGoals(goals journal.WeeklyGoals) journal.WeeklyGoals {
	cp := make(journal.WeeklyGoals, len(goals))
	for k, v := range goals {
		cp[k] = v
	}
	return cp
}
