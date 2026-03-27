package ui

import (
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/journal"
)

const deleteTodoIdx = -2

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
		case kb.StatsView:
			return m.openStatsView()
		case kb.SwitchWorkspace:
			return m.openWorkspacePicker()
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleDayViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.dayRecord.Entries)
	kb := m.cfg.Keybinds.Day
	inTodoPane := m.dayViewTab == 0 && m.selectedPane == 1
	if inTodoPane {
		switch msg.Type {
		case tea.KeyRunes:
			if m.todoInputMode {
				m.appendTodoDraft(string(msg.Runes))
				m.viewport.SetContent(m.renderDayContent())
				return m, nil
			}
			// In TODO navigation mode, begin inline drafting immediately for printable
			// characters that are not bound to other day-view commands.
			if m.shouldStartInlineTodoDraft(msg) {
				m.todoInputMode = true
				m.todoDraft = string(msg.Runes)
				m.viewport.SetContent(m.renderDayContent())
				return m, nil
			}
		case tea.KeyBackspace:
			if m.todoInputMode {
				m.backspaceTodoDraft()
				m.viewport.SetContent(m.renderDayContent())
				return m, nil
			}
			if m.deleteSelectedTodoNow() {
				m.viewport.SetContent(m.renderDayContent())
				return m, m.saveWorkspaceTodosCmd("✓ TODO deleted")
			}
			return m, nil
		}
	}
	if inTodoPane && m.todoInputMode {
		switch msg.String() {
		case "tab", "shift+tab", "delete", "up", "down", "left", "right":
			return m, nil
		}
	}
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
		if m.dayViewTab == 0 {
			if m.selectedPane == 0 && m.selectedEntry < n-1 {
				m.selectedEntry++
				m.viewport.SetContent(m.renderDayContent())
				m.scrollToSelected()
			} else if m.selectedPane == 1 {
				m.todoMove(1)
				m.viewport.SetContent(m.renderDayContent())
			}
		}
		return m, nil
	case "k", "up":
		if m.dayViewTab == 0 {
			if m.selectedPane == 0 && m.selectedEntry > 0 {
				m.selectedEntry--
				m.viewport.SetContent(m.renderDayContent())
				m.scrollToSelected()
			} else if m.selectedPane == 1 {
				m.todoMove(-1)
				m.viewport.SetContent(m.renderDayContent())
			}
		}
		return m, nil
	case "tab":
		if m.dayViewTab == 0 {
			if m.selectedPane == 1 && m.todoInputMode {
				return m, nil
			}
			if m.selectedPane == 1 && !m.todoInputMode {
				if m.indentSelectedTodo() {
					m.viewport.SetContent(m.renderDayContent())
					return m, m.saveWorkspaceTodosCmd("✓ TODO indented")
				}
				// In focused TODO navigation mode, tab is reserved for indenting and does not cycle panes.
				return m, nil
			}
			m.selectedPane = (m.selectedPane + 1) % 2
			m.viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "shift+tab":
		if m.dayViewTab == 0 && m.selectedPane == 1 && m.outdentSelectedTodo() {
			m.viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO outdented")
		}
		return m, nil
	case "delete":
		if m.dayViewTab == 0 && m.selectedPane == 1 && m.deleteSelectedTodoNow() {
			m.viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO deleted")
		}
		return m, nil
	case "enter":
		if m.dayViewTab == 0 && m.selectedPane == 1 {
			if m.todoInputMode {
				saved := m.commitTodoDraft()
				m.exitTodoInputMode()
				m.viewport.SetContent(m.renderDayContent())
				if saved {
					return m, m.saveWorkspaceTodosCmd("✓ TODO saved")
				}
				return m, nil
			}
			m.todoInputMode = true
			m.todoDraft = ""
			m.viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case " ":
		// Some terminals report space as a dedicated key type (not KeyRunes).
		// Preserve typing spaces while drafting a TODO title.
		if m.dayViewTab == 0 && m.selectedPane == 1 && m.todoInputMode {
			m.appendTodoDraft(" ")
			m.viewport.SetContent(m.renderDayContent())
			return m, nil
		}
		if m.dayViewTab == 0 && m.selectedPane == 1 && m.toggleSelectedTodo() {
			m.viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO updated")
		}
		return m, nil
	case "a":
		if m.dayViewTab == 0 && m.selectedPane == 1 {
			m.todoInputMode = true
			m.todoDraft = ""
			m.viewport.SetContent(m.renderDayContent())
			return m, nil
		}
		return m, nil
	case "A":
		if m.dayViewTab == 0 && m.selectedPane == 1 && m.selectedTodo >= 0 && m.selectedTodo < len(m.workspaceTodos) {
			if m.selectedSub >= 0 && m.selectedSub < len(m.workspaceTodos[m.selectedTodo].Subtodos) {
				newSubIdx2 := len(m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos)
				return m.openTodoForm(m.selectedTodo, m.selectedSub, newSubIdx2)
			}
			newSubIdx := len(m.workspaceTodos[m.selectedTodo].Subtodos)
			return m.openTodoForm(m.selectedTodo, newSubIdx, -1)
		}
		return m, nil
	case kb.AddWork:
		return m.openWorkForm(false, -1)
	case kb.AddBreak:
		return m.openWorkForm(true, -1)
	case kb.Edit:
		if m.dayViewTab == 0 && m.selectedPane == 1 {
			return m.openTodoFormForSelection()
		}
		if m.selectedEntry >= 0 && m.selectedEntry < n {
			return m.openWorkForm(m.dayRecord.Entries[m.selectedEntry].IsBreak, m.selectedEntry)
		}
		return m.openNotesEditor()
	case kb.Delete:
		if m.dayViewTab == 0 && m.selectedPane == 1 {
			if m.selectedTodo >= 0 && m.selectedTodo < len(m.workspaceTodos) {
				m.deleteDay = false
				m.deleteIdx = deleteTodoIdx
				m.prevState = stateDayView
				m.state = stateConfirmDelete
			}
			return m, nil
		}
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
	case kb.TodoOverview:
		if m.dayViewTab == 0 {
			if m.selectedPane != 1 {
				m.selectedPane = 1
			} else {
				m.selectedPane = 0
				m.exitTodoInputMode()
			}
			m.viewport.SetContent(m.renderDayContent())
			return m, nil
		}
		m.dayViewTab = 0
		m.selectedPane = 1
		m.viewport.GotoTop()
		m.viewport.SetContent(m.renderDayContent())
		return m, nil
	case kb.Export:
		rec := m.dayRecord
		return m, func() tea.Msg {
			path, err := journal.SaveExport(rec)
			if err != nil {
				return errMsg{err: err}
			}
			return exportedMsg{path: path}
		}
	case kb.ClockStart:
		if !m.clockRunning {
			return m.openClockForm()
		}
		if kb.ClockStart == kb.ClockStop {
			return m.stopClock()
		}
		return m, nil
	case kb.ClockStop:
		if m.clockRunning {
			return m.stopClock()
		}
		return m, nil
	case "esc":
		clockWasRunning := m.clockRunning
		m.clockRunning = false
		m.clockTask = ""
		m.clockProject = ""
		m.state = stateList
		var cmds []tea.Cmd
		cmds = append(cmds, loadRecords)
		if clockWasRunning {
			m.statusMsg = "⏱ Clock stopped"
			m.isError = false
			cmds = append(cmds, clearStatusCmd())
		}
		return m, tea.Batch(cmds...)
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) openTodoFormForSelection() (tea.Model, tea.Cmd) {
	if m.selectedTodo >= 0 && m.selectedTodo < len(m.workspaceTodos) {
		return m.openTodoForm(m.selectedTodo, m.selectedSub, m.selectedSub2)
	}
	return m, nil
}

func (m Model) shouldStartInlineTodoDraft(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes || len(msg.Runes) == 0 {
		return false
	}
	for _, r := range msg.Runes {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	key := msg.String()
	switch key {
	case "j", "k", "a", "A", " ":
		return false
	}
	if m.isDayCommandKey(key) {
		return false
	}
	return true
}

func (m Model) isDayCommandKey(key string) bool {
	if key == "" {
		return false
	}
	kb := m.cfg.Keybinds.Day
	return key == kb.AddWork ||
		key == kb.AddBreak ||
		key == kb.Edit ||
		key == kb.Delete ||
		key == kb.SetStartNow ||
		key == kb.SetStartManual ||
		key == kb.SetEndNow ||
		key == kb.SetEndManual ||
		key == kb.Notes ||
		key == kb.TodoOverview ||
		key == kb.Export ||
		key == kb.ClockStart ||
		key == kb.ClockStop
}

func (m Model) handleTodoFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		title := strings.TrimSpace(m.todoInput.Value())
		if title == "" {
			m.statusMsg = "✗ TODO title is required"
			m.isError = true
			return m, clearStatusCmd()
		}
		if m.todoEditTop >= 0 && m.todoEditTop < len(m.workspaceTodos) {
			if m.todoEditSub >= 0 {
				if m.todoEditSub < len(m.workspaceTodos[m.todoEditTop].Subtodos) {
					if m.todoEditSub2 >= 0 && m.todoEditSub2 < len(m.workspaceTodos[m.todoEditTop].Subtodos[m.todoEditSub].Subtodos) {
						m.workspaceTodos[m.todoEditTop].Subtodos[m.todoEditSub].Subtodos[m.todoEditSub2].Title = title
					} else if m.todoEditSub2 >= 0 {
						m.workspaceTodos[m.todoEditTop].Subtodos[m.todoEditSub].Subtodos = append(m.workspaceTodos[m.todoEditTop].Subtodos[m.todoEditSub].Subtodos, journal.Todo{
							ID:       journal.NewID(),
							Title:    title,
							Subtodos: []journal.Todo{},
						})
						m.selectedSub2 = len(m.workspaceTodos[m.todoEditTop].Subtodos[m.todoEditSub].Subtodos) - 1
					} else {
						m.workspaceTodos[m.todoEditTop].Subtodos[m.todoEditSub].Title = title
					}
				} else {
					m.workspaceTodos[m.todoEditTop].Subtodos = append(m.workspaceTodos[m.todoEditTop].Subtodos, journal.Todo{
						ID:       journal.NewID(),
						Title:    title,
						Subtodos: []journal.Todo{},
					})
					m.selectedSub = len(m.workspaceTodos[m.todoEditTop].Subtodos) - 1
					m.selectedSub2 = -1
				}
				m.selectedTodo = m.todoEditTop
			} else {
				m.workspaceTodos[m.todoEditTop].Title = title
				m.selectedTodo = m.todoEditTop
				m.selectedSub = -1
				m.selectedSub2 = -1
			}
		} else {
			m.workspaceTodos = append(m.workspaceTodos, journal.Todo{
				ID:       journal.NewID(),
				Title:    title,
				Subtodos: []journal.Todo{},
			})
			m.selectedTodo = len(m.workspaceTodos) - 1
			m.selectedSub = -1
			m.selectedSub2 = -1
		}
		m.state = stateDayView
		m.selectedPane = 1
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveWorkspaceTodosCmd("✓ TODO saved")
	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.todoInput, cmd = m.todoInput.Update(msg)
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
	switch msg.String() {
	case "r":
		if m.timeInputStart {
			m.dayRecord.StartTime = ""
		} else {
			m.dayRecord.EndTime = ""
		}
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		label := "✓ End time cleared"
		if m.timeInputStart {
			label = "✓ Start time cleared"
		}
		return m, m.saveDayCmd(label)
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
		if m.deleteIdx == deleteTodoIdx {
			if m.selectedTodo >= 0 && m.selectedTodo < len(m.workspaceTodos) {
				if m.selectedSub >= 0 && m.selectedSub2 >= 0 && m.selectedSub < len(m.workspaceTodos[m.selectedTodo].Subtodos) {
					level2 := m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos
					if m.selectedSub2 >= 0 && m.selectedSub2 < len(level2) {
						m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos = append(level2[:m.selectedSub2], level2[m.selectedSub2+1:]...)
					}
					m.selectedSub2 = -1
				} else if m.selectedSub >= 0 && m.selectedSub < len(m.workspaceTodos[m.selectedTodo].Subtodos) {
					st := m.workspaceTodos[m.selectedTodo].Subtodos
					m.workspaceTodos[m.selectedTodo].Subtodos = append(st[:m.selectedSub], st[m.selectedSub+1:]...)
					m.selectedSub = -1
					m.selectedSub2 = -1
				} else {
					m.workspaceTodos = append(m.workspaceTodos[:m.selectedTodo], m.workspaceTodos[m.selectedTodo+1:]...)
					if m.selectedTodo >= len(m.workspaceTodos) {
						m.selectedTodo = len(m.workspaceTodos) - 1
					}
					if m.selectedTodo < 0 {
						m.selectedSub = -1
						m.selectedSub2 = -1
					}
				}
			}
			m.state = stateDayView
			m.selectedPane = 1
			m.viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO deleted")
		}
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

func (m Model) handleClockFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	const numFields = 2
	switch msg.Type {
	case tea.KeyTab:
		return m.focusField((m.activeInput + 1) % numFields)
	case tea.KeyShiftTab:
		return m.focusField((m.activeInput - 1 + numFields) % numFields)

	case tea.KeyEnter:
		if m.activeInput < numFields-1 {
			return m.focusField(m.activeInput + 1)
		}
		// Submit — start the clock.
		task := strings.TrimSpace(m.taskInput.Value())
		if task == "" {
			m.statusMsg = "✗ Task name is required"
			m.isError = true
			return m, clearStatusCmd()
		}
		m.clockTask = task
		m.clockProject = strings.TrimSpace(m.projectInput.Value())
		m.clockStart = time.Now()
		m.clockRunning = true
		m.state = stateDayView
		m.dayViewTab = 0
		m.viewport.SetContent(m.renderDayContent())
		return m, clockTickCmd()

	case tea.KeyEsc:
		m.state = stateDayView
		m.dayViewTab = 0
		return m, nil
	}

	var cmd tea.Cmd
	if m.activeInput == 0 {
		m.taskInput, cmd = m.taskInput.Update(msg)
	} else {
		m.projectInput, cmd = m.projectInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	const numStatsTabs = 4
	switch msg.String() {
	case "esc", m.cfg.Keybinds.List.Quit:
		m.state = stateList
		return m, nil
	case "left":
		if m.statsTab > 0 {
			m.statsTab--
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderStatsTabContent())
		}
		return m, nil
	case "right":
		if m.statsTab < numStatsTabs-1 {
			m.statsTab++
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderStatsTabContent())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
