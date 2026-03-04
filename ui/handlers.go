package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/journal"
)

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtering := m.list.FilterState() == list.Filtering
	if !filtering {
		kb := m.cfg.Keybinds.List
		switch msg.String() {
		case kb.Quit, "esc":
			return m, tea.Quit
		case kb.OpenToday:
			return m.openDayViewToday()
		case kb.OpenDate:
			return m.openDateInput()
		case "enter":
			if item, ok := m.list.SelectedItem().(dayListItem); ok {
				return m.openDayView(item.rec)
			}
		case kb.Delete:
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.records) {
				m.deleteDay = true
				m.deleteIdx = idx
				m.prevState = stateList
				m.state = stateConfirmDelete
				return m, nil
			}
		case kb.AddWork:
			return m.openWorkFormForToday(false)
		case kb.AddBreak:
			return m.openWorkFormForToday(true)
		case kb.Export:
			if item, ok := m.list.SelectedItem().(dayListItem); ok {
				rec := item.rec
				return m, func() tea.Msg {
					path, err := journal.SaveExport(rec)
					if err != nil {
						return errMsg{err: err}
					}
					return exportedMsg{path: path}
				}
			}
		case kb.WeekView:
			return m.openWeekView()
		case kb.SwitchWorkspace:
			return m.openWorkspacePicker()
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleWeekViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := m.cfg.Keybinds
	switch msg.String() {
	case "esc", kb.List.Quit:
		m.state = stateList
		return m, nil
	case "left", kb.Week.PrevWeek:
		m.weekOffset--
		m.viewport.GotoTop()
		m.viewport.SetContent(m.renderWeekContent())
		return m, nil
	case "right", kb.Week.NextWeek:
		if m.weekOffset < 0 {
			m.weekOffset++
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderWeekContent())
		}
		return m, nil
	case kb.Week.SetWeeklyHours:
		return m.openWeekHoursInput()
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleWeekHoursInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		raw := strings.TrimSpace(m.weekHoursInput.Value())
		m.weekHoursInput.Blur()

		var hours float64
		if raw == "" {
			// Empty input resets to the workspace/global default.
			delete(m.weekGoals, m.weekKey())
			m.state = stateWeekView
			m.viewport.SetContent(m.renderWeekContent())
			goalsCopy := copyWeeklyGoals(m.weekGoals)
			return m, func() tea.Msg {
				if err := journal.SaveWeeklyGoals(goalsCopy); err != nil {
					return errMsg{err: err}
				}
				return weekGoalsLoadedMsg{goals: goalsCopy}
			}
		}
		if _, err := fmt.Sscanf(raw, "%f", &hours); err != nil || hours <= 0 {
			m.statusMsg = "✗ Invalid hours — enter a positive number (e.g. 32 or 37.5)"
			m.isError = true
			m.state = stateWeekView
			m.viewport.SetContent(m.renderWeekContent())
			return m, clearStatusCmd()
		}
		m.weekGoals[m.weekKey()] = hours
		m.state = stateWeekView
		m.viewport.SetContent(m.renderWeekContent())
		goalsCopy := copyWeeklyGoals(m.weekGoals)
		return m, func() tea.Msg {
			if err := journal.SaveWeeklyGoals(goalsCopy); err != nil {
				return errMsg{err: err}
			}
			return weekGoalsLoadedMsg{goals: goalsCopy}
		}
	case tea.KeyEsc:
		m.weekHoursInput.Blur()
		m.state = stateWeekView
		return m, nil
	}
	var cmd tea.Cmd
	m.weekHoursInput, cmd = m.weekHoursInput.Update(msg)
	return m, cmd
}

func (m Model) handleDayViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.dayRecord.Entries)
	kb := m.cfg.Keybinds.Day
	switch msg.String() {
	case "left":
		if m.dayViewTab > 0 {
			m.dayViewTab--
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "right":
		if m.dayViewTab < 1 {
			m.dayViewTab++
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "j", "down":
		if m.dayViewTab == 0 && m.selectedEntry < n-1 {
			m.selectedEntry++
			m.viewport.SetContent(m.renderDayContent())
			m.scrollToSelected()
		}
		return m, nil
	case "k", "up":
		if m.dayViewTab == 0 && m.selectedEntry > 0 {
			m.selectedEntry--
			m.viewport.SetContent(m.renderDayContent())
			m.scrollToSelected()
		}
		return m, nil
	case kb.AddWork:
		return m.openWorkForm(false, -1)
	case kb.AddBreak:
		return m.openWorkForm(true, -1)
	case kb.Edit:
		if m.selectedEntry >= 0 && m.selectedEntry < n {
			return m.openWorkForm(m.dayRecord.Entries[m.selectedEntry].IsBreak, m.selectedEntry)
		}
		return m.openNotesEditor()
	case kb.Delete:
		if m.selectedEntry >= 0 && m.selectedEntry < n {
			m.deleteDay = false
			m.deleteIdx = m.selectedEntry
			m.prevState = stateDayView
			m.state = stateConfirmDelete
			return m, nil
		}
		m.deleteDay = true
		m.deleteIdx = -1 // current day
		m.prevState = stateDayView
		m.state = stateConfirmDelete
		return m, nil
	case kb.SetStartNow:
		m.dayRecord.StartTime = time.Now().Format("15:04")
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ Start time set to " + m.dayRecord.StartTime)
	case kb.SetStartManual:
		return m.openTimeInput(true)
	case kb.SetEndNow:
		m.dayRecord.EndTime = time.Now().Format("15:04")
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ End time set to " + m.dayRecord.EndTime)
	case kb.SetEndManual:
		return m.openTimeInput(false)
	case kb.Notes:
		return m.openNotesEditor()
	case kb.Export:
		rec := m.dayRecord
		return m, func() tea.Msg {
			path, err := journal.SaveExport(rec)
			if err != nil {
				return errMsg{err: err}
			}
			return exportedMsg{path: path}
		}
	case "esc":
		m.state = stateList
		return m, loadRecords
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleWorkFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		return m.focusField((m.activeInput + 1) % m.numFormFields())
	case tea.KeyShiftTab:
		return m.focusField((m.activeInput - 1 + m.numFormFields()) % m.numFormFields())

	case tea.KeyEnter:
		lastField := m.numFormFields() - 1
		if m.activeInput < lastField {
			return m.focusField(m.activeInput + 1)
		}
		// ── Submit ────────────────────────────────────────────────────────────
		task := strings.TrimSpace(m.taskInput.Value())
		durStr := strings.TrimSpace(m.durationInput.Value())
		if task == "" {
			m.statusMsg = "✗ Task name is required"
			m.isError = true
			return m, clearStatusCmd()
		}
		dur, err := journal.ParseDuration(durStr)
		if err != nil {
			m.statusMsg = "✗ " + err.Error()
			m.isError = true
			return m, clearStatusCmd()
		}
		projectRaw := strings.TrimSpace(m.projectInput.Value())
		isBreak := m.isBreakEntry
		editIdx := m.editEntryIdx
		wasSplit := false

		if editIdx >= 0 && editIdx < len(m.dayRecord.Entries) && isBreak {
			// Update existing break entry in-place.
			m.dayRecord.Entries[editIdx].Task = task
			m.dayRecord.Entries[editIdx].DurationMin = int(dur.Minutes())
			m.selectedEntry = editIdx
		} else if isBreak {
			// For new breaks: merge into an existing break with the same label (case-insensitive).
			taskLower := strings.ToLower(task)
			merged := false
			for i, e := range m.dayRecord.Entries {
				if e.IsBreak && strings.ToLower(e.Task) == taskLower {
					m.dayRecord.Entries[i].DurationMin += int(dur.Minutes())
					m.selectedEntry = i
					merged = true
					break
				}
			}
			if !merged {
				entry := journal.WorkEntry{
					ID:          journal.NewID(),
					Task:        task,
					DurationMin: int(dur.Minutes()),
					IsBreak:     true,
				}
				m.dayRecord.Entries = append(m.dayRecord.Entries, entry)
				m.selectedEntry = len(m.dayRecord.Entries) - 1
			}
		} else {
			// Split comma-separated projects and distribute duration evenly.
			// This applies both to new entries and edited entries.
			rawParts := strings.Split(projectRaw, ",")
			projects := make([]string, 0, len(rawParts))
			for _, p := range rawParts {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					projects = append(projects, trimmed)
				}
			}
			if len(projects) == 0 {
				projects = []string{""}
			}
			totalMin := int(dur.Minutes())
			base := totalMin / len(projects)
			remainder := totalMin % len(projects)
			newEntries := make([]journal.WorkEntry, 0, len(projects))
			for i, proj := range projects {
				mins := base
				if i < remainder {
					mins++ // distribute remainder evenly: one extra minute to first N projects
				}
				if mins == 0 {
					continue // skip zero-duration entries when duration < number of projects
				}
				newEntries = append(newEntries, journal.WorkEntry{
					ID:          journal.NewID(),
					Task:        task,
					Project:     proj,
					DurationMin: mins,
					IsBreak:     false,
				})
			}
			if len(newEntries) == 0 {
				m.statusMsg = "✗ Duration too short to distribute across projects"
				m.isError = true
				return m, clearStatusCmd()
			}
			wasSplit = len(projects) > 1
			if editIdx >= 0 && editIdx < len(m.dayRecord.Entries) {
				// Replace the edited entry with the split entries.
				updated := make([]journal.WorkEntry, 0, len(m.dayRecord.Entries)-1+len(newEntries))
				updated = append(updated, m.dayRecord.Entries[:editIdx]...)
				updated = append(updated, newEntries...)
				updated = append(updated, m.dayRecord.Entries[editIdx+1:]...)
				m.dayRecord.Entries = updated
				m.selectedEntry = editIdx + len(newEntries) - 1
			} else {
				m.dayRecord.Entries = append(m.dayRecord.Entries, newEntries...)
				m.selectedEntry = len(m.dayRecord.Entries) - 1
			}
		}

		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		m.scrollToSelected()

		label := "✓ Work entry logged"
		if editIdx >= 0 && !wasSplit {
			label = "✓ Entry updated"
		} else if isBreak {
			label = "✓ Break logged"
		} else if wasSplit {
			label = "✓ Work entries split across projects"
		}
		return m, m.saveDayCmd(label)

	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}

	var cmd tea.Cmd
	switch {
	case m.activeInput == 0:
		m.taskInput, cmd = m.taskInput.Update(msg)
	case m.activeInput == 1 && !m.isBreakEntry:
		m.projectInput, cmd = m.projectInput.Update(msg)
	default:
		m.durationInput, cmd = m.durationInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleTimeInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.timeInput.Value())
		if !isValidHHMM(val) {
			m.statusMsg = "✗ Invalid time — use HH:MM (e.g. 09:30)"
			m.isError = true
			m.state = stateDayView
			return m, clearStatusCmd()
		}
		if m.timeInputStart {
			m.dayRecord.StartTime = val
		} else {
			m.dayRecord.EndTime = val
		}
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		label := "✓ End time set to " + val
		if m.timeInputStart {
			label = "✓ Start time set to " + val
		}
		return m, m.saveDayCmd(label)
	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.timeInput, cmd = m.timeInput.Update(msg)
	return m, cmd
}

func (m Model) handleNotesEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		m.dayRecord.Notes = m.textarea.Value()
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ Notes saved")
	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteDay {
			// Delete the whole day file.
			var path string
			if m.prevState == stateDayView {
				path = m.dayRecord.Path
			} else if m.deleteIdx >= 0 && m.deleteIdx < len(m.records) {
				path = m.records[m.deleteIdx].Path
			}
			if path == "" {
				m.state = m.prevState
				return m, nil
			}
			return m, func() tea.Msg {
				if err := journal.Delete(path); err != nil {
					return errMsg{err: err}
				}
				return dayDeletedMsg{}
			}
		}
		// Delete a single entry from the current day.
		if m.deleteIdx >= 0 && m.deleteIdx < len(m.dayRecord.Entries) {
			m.dayRecord.Entries = append(
				m.dayRecord.Entries[:m.deleteIdx],
				m.dayRecord.Entries[m.deleteIdx+1:]...,
			)
			if m.selectedEntry >= len(m.dayRecord.Entries) {
				m.selectedEntry = len(m.dayRecord.Entries) - 1
			}
		}
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ Entry deleted")
	case "n", "N", "esc":
		m.state = m.prevState
	}
	return m, nil
}

func (m Model) handleDateInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.dateInput.Blur()
		m.state = stateList
		return m, nil
	case "enter":
		raw := strings.TrimSpace(m.dateInput.Value())
		if raw == "" {
			raw = time.Now().Format("2006-01-02")
		}
		// Accept YYYY-MM-DD
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			m.statusMsg = "✗ Invalid date — use YYYY-MM-DD"
			m.isError = true
			m.state = stateList
			return m, clearStatusCmd()
		}
		m.dateInput.Blur()
		dateStr := date.Format("2006-01-02")
		path, err := journal.PathForDate(dateStr)
		if err != nil {
			m.statusMsg = "✗ " + err.Error()
			m.isError = true
			m.state = stateList
			return m, clearStatusCmd()
		}
		rec, _ := journal.Load(path)
		if rec.Date == "" {
			rec.Date = dateStr
		}
		rec.Path = path
		return m.openDayView(rec)
	}
	var cmd tea.Cmd
	m.dateInput, cmd = m.dateInput.Update(msg)
	return m, cmd
}

func (m Model) handleWorkspacePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.cfg.Workspaces)
	switch msg.String() {
	case "j", "down":
		if m.workspaceIdx < n-1 {
			m.workspaceIdx++
		}
		return m, nil
	case "k", "up":
		if m.workspaceIdx > 0 {
			m.workspaceIdx--
		}
		return m, nil
	case "enter":
		if m.workspaceIdx >= 0 && m.workspaceIdx < n {
			return m.switchWorkspace(m.cfg.Workspaces[m.workspaceIdx].Name)
		}
	case "esc", m.cfg.Keybinds.List.Quit:
		m.state = stateList
	}
	return m, nil
}
