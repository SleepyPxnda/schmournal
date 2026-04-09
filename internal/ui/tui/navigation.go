package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/application/usecase"
	domainmodel "github.com/sleepypxnda/schmournal/internal/domain/model"
)

// effectiveIsWorkDay reports whether t is a work day for the active workspace.
func (m Model) effectiveIsWorkDay(t time.Time) bool {
	if ws := m.activeWorkspaceConfig(); ws != nil {
		wd := strings.ToLower(t.Weekday().String())
		for _, d := range ws.WorkDays {
			if d == wd {
				return true
			}
		}
		return false
	}
	for _, d := range domainmodel.DefaultWorkspaceConfig("").WorkDays {
		if d == strings.ToLower(t.Weekday().String()) {
			return true
		}
	}
	return false
}

// activeWorkspaceConfig returns a pointer to the active WorkspaceConfig, or nil
// when no workspaces are configured or none matches the active name.
func (m Model) activeWorkspaceConfig() *domainmodel.WorkspaceConfig {
	for i := range m.context.Config.Workspaces {
		if m.context.Config.Workspaces[i].Name == m.context.ActiveWorkspace {
			return &m.context.Config.Workspaces[i]
		}
	}
	return nil
}

// effectiveWeeklyHoursGoal returns the hours-per-week goal for the active
// workspace.
func (m Model) effectiveWeeklyHoursGoal() float64 {
	if ws := m.activeWorkspaceConfig(); ws != nil {
		return ws.WeeklyHoursGoal
	}
	return domainmodel.DefaultWorkspaceConfig("").WeeklyHoursGoal
}

// openWorkspacePicker opens the workspace picker dialog. If no workspaces are
// configured a status message is shown instead.
func (m Model) openWorkspacePicker() (tea.Model, tea.Cmd) {
	if len(m.context.Config.Workspaces) == 0 {
		m.status.Message = "No workspaces configured — add [[workspaces]] entries to your config file"
		m.status.IsError = false
		return m, clearStatusCmd()
	}
	// Pre-select the currently active workspace.
	m.workspacePicker.Index = 0
	for i, ws := range m.context.Config.Workspaces {
		if ws.Name == m.context.ActiveWorkspace {
			m.workspacePicker.Index = i
			break
		}
	}
	m.ui.Current = stateWorkspacePicker
	return m, nil
}

// switchWorkspace applies the named workspace: rebinds use cases to the
// workspace storage path, records the active workspace name and reloads data.
func (m Model) switchWorkspace(name string) (tea.Model, tea.Cmd) {
	storagePath := domainmodel.DefaultWorkspaceConfig("").StoragePath
	for _, ws := range m.context.Config.Workspaces {
		if ws.Name == name {
			storagePath = ws.StoragePath
			break
		}
	}
	if m.context.UseCases == nil {
		m.status.Message = "✗ use cases are not configured"
		m.status.IsError = true
		m.ui.Current = stateList
		return m, clearStatusCmd()
	}
	if err := m.context.UseCases.ReinitializeForStorage(storagePath); err != nil {
		m.status.Message = "✗ " + err.Error()
		m.status.IsError = true
		m.ui.Current = stateList
		return m, clearStatusCmd()
	}
	m.context.ActiveWorkspace = name
	m.ui.Current = stateList
	m.status.Message = fmt.Sprintf("✓ Switched to workspace %q", name)
	m.status.IsError = false
	return m, tea.Batch(
		m.loadRecordsCmd(),
		m.loadWorkspaceTodosCmd(),
		clearStatusCmd(),
		func() tea.Msg {
			if m.context.UseCases == nil {
				return errMsg{err: fmt.Errorf("use cases are not configured")}
			}
			if err := m.context.UseCases.SaveActiveWorkspace(name); err != nil {
				return errMsg{err: err}
			}
			return nil
		},
	)
}

func (m Model) openDayView(rec DayRecord) (tea.Model, tea.Cmd) {
	// Reload from disk to get freshest data.
	fresh := rec
	if rec.Date != "" {
		loaded, err := m.loadDayRecord(rec.Date)
		if err == nil {
			fresh = loaded
		}
	}
	m.day.Record = fresh
	m.day.Selection.DayTab = 0
	m.day.Selection.Pane = 0
	m.day.Selection.EntryIdx = -1
	if len(m.day.Record.Entries) > 0 {
		m.day.Selection.EntryIdx = 0
	}
	m.todoSelection.Top = 0
	m.todoSelection.Sub = -1
	m.todoSelection.Sub2 = -1
	m.exitTodoInputMode()
	if len(m.workspace.Todos) == 0 {
		m.todoSelection.Top = -1
	}
	m.ui.Current = stateDayView
	m.day.Viewport.GotoTop()
	m.day.Viewport.SetContent(m.renderDayContent())
	return m, nil
}

func (m Model) openStatsView() (tea.Model, tea.Cmd) {
	m.ui.Current = stateStats
	m.stats.Tab = 0
	m.day.Viewport.GotoTop()
	m.day.Viewport.SetContent(m.renderStatsTabContent())
	return m, nil
}

func (m Model) openWeekView() (tea.Model, tea.Cmd) {
	m.weekOffset = 0
	m.ui.Current = stateWeekView
	m.day.Viewport.GotoTop()
	m.day.Viewport.SetContent(m.renderWeekContent())
	return m, nil
}

func (m Model) loadDayRecord(date string) (DayRecord, error) {
	if m.context.UseCases == nil || m.context.UseCases.LoadDayRecord == nil {
		return DayRecord{}, fmt.Errorf("load day record use case is not configured")
	}
	rec, err := m.context.UseCases.LoadDayRecord.ExecuteDTO(usecase.LoadDayRecordInput{Date: date})
	if err != nil {
		return DayRecord{}, err
	}
	return toUIDayRecord(rec), nil
}

func (m Model) openDayViewToday() (tea.Model, tea.Cmd) {
	today := time.Now().Format("2006-01-02")
	rec, err := m.loadDayRecord(today)
	if err != nil {
		m.status.Message = "✗ " + err.Error()
		m.status.IsError = true
		return m, nil
	}
	return m.openDayView(rec)
}

func (m Model) openWorkForm(isBreak bool, editIdx int) (tea.Model, tea.Cmd) {
	m.workForm.IsBreakEntry = isBreak
	m.workForm.EditEntryIdx = editIdx
	m.workForm.TaskInput.SetValue("")
	m.workForm.ProjectInput.SetValue("")
	m.workForm.DurationInput.SetValue("")
	if editIdx >= 0 && editIdx < len(m.day.Record.Entries) {
		e := m.day.Record.Entries[editIdx]
		m.workForm.TaskInput.SetValue(e.Task)
		m.workForm.ProjectInput.SetValue(e.Project)
		m.workForm.DurationInput.SetValue(formatDuration(e.Duration()))
	}
	if isBreak {
		m.workForm.TaskInput.Placeholder = "e.g. Lunch, coffee break, walk…"
	} else {
		m.workForm.TaskInput.Placeholder = "e.g. Feature development, meeting, code review…"
	}
	m.ui.Current = stateWorkForm
	return m.focusField(0)
}

func (m Model) openWorkFormForToday(isBreak bool) (tea.Model, tea.Cmd) {
	today := time.Now().Format("2006-01-02")
	rec, err := m.loadDayRecord(today)
	if err != nil {
		m.status.Message = "✗ " + err.Error()
		m.status.IsError = true
		return m, nil
	}
	m.day.Record = rec
	if m.day.Selection.EntryIdx < 0 || m.day.Selection.EntryIdx >= len(rec.Entries) {
		m.day.Selection.EntryIdx = len(rec.Entries) - 1
	}
	// After form submission we'll land in stateDayView.
	return m.openWorkForm(isBreak, -1)
}

func (m Model) openNotesEditor() (tea.Model, tea.Cmd) {
	m.day.Notes.SetValue(m.day.Record.Notes)
	blinkCmd := m.day.Notes.Focus()
	m.ui.Current = stateNotesEditor
	return m, blinkCmd
}

func (m Model) openTodoForm(editTop, editSub, editSub2 int) (tea.Model, tea.Cmd) {
	m.todoEditor.EditTop = editTop
	m.todoEditor.EditSub = editSub
	m.todoEditor.EditSub2 = editSub2
	m.todoEditor.Input.SetValue("")
	m.todoEditor.Input.Placeholder = "TODO title…"
	if editTop >= 0 && editTop < len(m.workspace.Todos) {
		if editSub >= 0 && editSub < len(m.workspace.Todos[editTop].Subtodos) {
			if editSub2 >= 0 && editSub2 < len(m.workspace.Todos[editTop].Subtodos[editSub].Subtodos) {
				m.todoEditor.Input.SetValue(m.workspace.Todos[editTop].Subtodos[editSub].Subtodos[editSub2].Title)
			} else {
				m.todoEditor.Input.SetValue(m.workspace.Todos[editTop].Subtodos[editSub].Title)
			}
		} else {
			m.todoEditor.Input.SetValue(m.workspace.Todos[editTop].Title)
		}
	}
	m.ui.Current = stateTodoForm
	return m, m.todoEditor.Input.Focus()
}

func (m Model) openTimeInput(isStart bool) (tea.Model, tea.Cmd) {
	m.timeForm.IsStart = isStart
	existing := m.day.Record.StartTime
	if !isStart {
		existing = m.day.Record.EndTime
	}
	if existing != "" {
		m.timeForm.Input.SetValue(existing)
	} else {
		m.timeForm.Input.SetValue(time.Now().Format("15:04"))
	}
	m.timeForm.Input.CursorEnd()
	m.ui.Current = stateTimeInput
	return m, m.timeForm.Input.Focus()
}

func (m Model) openDateInput() (tea.Model, tea.Cmd) {
	m.dateForm.Input.SetValue("")
	m.dateForm.Input.Placeholder = time.Now().Format("2006-01-02")
	cmd := m.dateForm.Input.Focus()
	m.ui.Current = stateDateInput
	return m, cmd
}

// openClockForm opens the task-entry form used to start the clock timer.
func (m Model) openClockForm() (tea.Model, tea.Cmd) {
	m.workForm.IsBreakEntry = false
	m.workForm.TaskInput.SetValue("")
	m.workForm.ProjectInput.SetValue("")
	m.workForm.TaskInput.Placeholder = "e.g. Feature development, meeting, code review…"
	m.workForm.ProjectInput.Placeholder = "e.g. Backend, Frontend  (optional, comma-separated)"
	m.ui.Current = stateClockForm
	return m.focusField(0)
}

// stopClock stops the running clock, computes the elapsed duration, creates
// the appropriate WorkEntry values (splitting across projects if needed) and
// saves the day record. If the elapsed time rounds down to zero minutes a
// status message is shown instead.
func (m Model) stopClock() (tea.Model, tea.Cmd) {
	elapsed := time.Since(m.clock.Start)
	entries := clockEntries(m.clock.Task, m.clock.Project, elapsed)
	originalEntries := append([]WorkEntry(nil), m.day.Record.Entries...)

	m.clock.Running = false
	m.clock.Task = ""
	m.clock.Project = ""

	if len(entries) == 0 {
		m.status.Message = "✗ Clock stopped — duration too short to log (< 1 minute)"
		m.status.IsError = true
		m.ui.Current = stateDayView
		m.day.Selection.DayTab = 0
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, clearStatusCmd()
	}

	m.day.Record.Entries = append(m.day.Record.Entries, entries...)
	m.day.Selection.EntryIdx = len(m.day.Record.Entries) - 1
	m.ui.Current = stateDayView
	m.day.Selection.DayTab = 0 // switch to Work Log so the new entry is visible
	m.day.Viewport.SetContent(m.renderDayContent())
	m.scrollToSelected()

	label := "✓ Clocked entry logged"
	if len(entries) > 1 {
		label = "✓ Clocked entries split across projects"
	}

	if m.context.UseCases == nil || m.context.UseCases.AddWorkEntry == nil || m.context.UseCases.LoadDayRecord == nil {
		m.status.Message = "✗ day entry use cases are not configured"
		m.status.IsError = true
		m.ui.Current = stateDayView
		m.day.Selection.DayTab = 0
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, clearStatusCmd()
	}

	date := m.day.Record.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
		m.day.Record.Date = date
	}

	startIdx := len(m.day.Record.Entries) - len(entries)
	var persistErr error
	for i, entry := range entries {
		out, err := m.context.UseCases.AddWorkEntry.Execute(usecase.AddWorkEntryInput{
			Date:        date,
			Task:        entry.Task,
			Project:     entry.Project,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		})
		if err != nil {
			persistErr = err
			break
		}
		newIdx := startIdx + i
		if newIdx >= 0 && newIdx < len(m.day.Record.Entries) {
			m.day.Record.Entries[newIdx].ID = out.EntryID
		}
	}

	if persistErr != nil {
		if m.day.Record.Date != "" {
			if fresh, loadErr := m.context.UseCases.LoadDayRecord.ExecuteDTO(usecase.LoadDayRecordInput{Date: m.day.Record.Date}); loadErr == nil {
				m.day.Record = toUIDayRecord(fresh)
			} else {
				m.day.Record.Entries = originalEntries
			}
		} else {
			m.day.Record.Entries = originalEntries
		}
		if len(m.day.Record.Entries) == 0 {
			m.day.Selection.EntryIdx = -1
		} else if m.day.Selection.EntryIdx >= len(m.day.Record.Entries) {
			m.day.Selection.EntryIdx = len(m.day.Record.Entries) - 1
		}
		m.ui.Current = stateDayView
		m.day.Selection.DayTab = 0
		m.day.Viewport.SetContent(m.renderDayContent())
		m.status.Message = "✗ " + persistErr.Error()
		m.status.IsError = true
		return m, clearStatusCmd()
	}

	return m, func() tea.Msg { return daySavedMsg{label: label} }
}

// numFormFields returns how many fields the current work-log form has.
func (m Model) numFormFields() int {
	if m.workForm.IsBreakEntry {
		return 2
	}
	return 3
}

// focusField blurs all inputs and focuses the one at index n.
func (m Model) focusField(n int) (Model, tea.Cmd) {
	m.workForm.TaskInput.Blur()
	m.workForm.ProjectInput.Blur()
	m.workForm.DurationInput.Blur()
	m.workForm.ActiveInput = n
	switch {
	case n == 0:
		return m, m.workForm.TaskInput.Focus()
	case n == 1 && !m.workForm.IsBreakEntry:
		return m, m.workForm.ProjectInput.Focus()
	default:
		return m, m.workForm.DurationInput.Focus()
	}
}

func isValidHHMM(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}

// scrollToSelected adjusts the viewport so the selected entry is visible.
func (m *Model) scrollToSelected() {
	const entryStartLine = 7
	if m.day.Selection.EntryIdx < 0 {
		return
	}
	targetLine := entryStartLine + m.day.Selection.EntryIdx
	if targetLine < m.day.Viewport.YOffset {
		m.day.Viewport.YOffset = targetLine
	} else if targetLine >= m.day.Viewport.YOffset+m.day.Viewport.Height {
		m.day.Viewport.YOffset = targetLine - m.day.Viewport.Height + 1
	}
}
