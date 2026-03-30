package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/application/usecase"
)

const deleteTodoIdx = -2

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtering := m.listState.Model.FilterState() == list.Filtering
	if !filtering {
		kb := m.context.Config.Keybinds.List
		switch msg.String() {
		case kb.Quit, "esc":
			return m, tea.Quit
		case kb.OpenToday:
			return m.openDayViewToday()
		case kb.OpenDate:
			return m.openDateInput()
		case "enter":
			if item, ok := m.listState.Model.SelectedItem().(dayListItem); ok {
				return m.openDayView(item.rec)
			}
		case kb.Delete:
			idx := m.listState.Model.Index()
			if idx >= 0 && idx < len(m.listState.Records) {
				m.delete.Day = true
				m.delete.Idx = idx
				m.delete.PrevState = stateList
				m.ui.Current = stateConfirmDelete
				return m, nil
			}
		case kb.Export:
			if item, ok := m.listState.Model.SelectedItem().(dayListItem); ok {
				return m, m.exportDayCmd(item.rec)
			}
		case kb.WeekView:
			return m.openWeekView()
		case kb.StatsView:
			return m.openStatsView()
		case kb.SwitchWorkspace:
			return m.openWorkspacePicker()
		}
	}
	var cmd tea.Cmd
	m.listState.Model, cmd = m.listState.Model.Update(msg)
	return m, cmd
}

func (m Model) handleDayViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.day.Record.Entries)
	kb := m.context.Config.Keybinds.Day
	inTodoPane := m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1
	if inTodoPane {
		switch msg.Type {
		case tea.KeyRunes:
			if m.todoEditor.InputMode {
				m.appendTodoDraft(string(msg.Runes))
				m.day.Viewport.SetContent(m.renderDayContent())
				return m, nil
			}
			// In TODO navigation mode, begin inline drafting immediately for printable
			// characters that are not bound to other day-view commands.
			if m.shouldStartInlineTodoDraft(msg) {
				m.todoEditor.InputMode = true
				m.todoEditor.Draft = string(msg.Runes)
				m.day.Viewport.SetContent(m.renderDayContent())
				return m, nil
			}
		case tea.KeyBackspace:
			if m.todoEditor.InputMode {
				m.backspaceTodoDraft()
				m.day.Viewport.SetContent(m.renderDayContent())
				return m, nil
			}
			if m.deleteSelectedTodoNow() {
				m.day.Viewport.SetContent(m.renderDayContent())
				return m, m.saveWorkspaceTodosCmd("✓ TODO deleted")
			}
			return m, nil
		}
	}
	if inTodoPane && m.todoEditor.InputMode {
		switch msg.String() {
		case "tab", "shift+tab", "delete", "up", "down", "left", "right", "shift+up", "shift+down":
			return m, nil
		}
	}
	switch msg.String() {
	case "left":
		if m.day.Selection.DayTab > 0 {
			m.day.Selection.DayTab--
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "right":
		if m.day.Selection.DayTab < 1 {
			m.day.Selection.DayTab++
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "j", "down":
		if m.day.Selection.DayTab == 0 {
			if m.day.Selection.Pane == 0 && m.day.Selection.EntryIdx < n-1 {
				m.day.Selection.EntryIdx++
				m.day.Viewport.SetContent(m.renderDayContent())
				m.scrollToSelected()
			} else if m.day.Selection.Pane == 1 {
				m.todoMove(1)
				m.day.Viewport.SetContent(m.renderDayContent())
			}
		}
		return m, nil
	case "k", "up":
		if m.day.Selection.DayTab == 0 {
			if m.day.Selection.Pane == 0 && m.day.Selection.EntryIdx > 0 {
				m.day.Selection.EntryIdx--
				m.day.Viewport.SetContent(m.renderDayContent())
				m.scrollToSelected()
			} else if m.day.Selection.Pane == 1 {
				m.todoMove(-1)
				m.day.Viewport.SetContent(m.renderDayContent())
			}
		}
		return m, nil
	case "tab":
		if m.day.Selection.DayTab == 0 {
			if m.day.Selection.Pane == 1 && m.todoEditor.InputMode {
				return m, nil
			}
			if m.day.Selection.Pane == 1 && !m.todoEditor.InputMode {
				if m.indentSelectedTodo() {
					m.day.Viewport.SetContent(m.renderDayContent())
					return m, m.saveWorkspaceTodosCmd("✓ TODO indented")
				}
				// In focused TODO navigation mode, tab is reserved for indenting and does not cycle panes.
				return m, nil
			}
			m.day.Selection.Pane = (m.day.Selection.Pane + 1) % 2
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "shift+tab":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.outdentSelectedTodo() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO outdented")
		}
		return m, nil
	case "shift+up":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.moveSelectedTodoDelta(-1) {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO moved up")
		}
		return m, nil
	case "shift+down":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.moveSelectedTodoDelta(1) {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO moved down")
		}
		return m, nil
	case "delete":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.deleteSelectedTodoNow() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO deleted")
		}
		return m, nil
	case "enter":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			if m.todoEditor.InputMode {
				saved := m.commitTodoDraft()
				m.exitTodoInputMode()
				m.day.Viewport.SetContent(m.renderDayContent())
				if saved {
					return m, m.saveWorkspaceTodosCmd("✓ TODO saved")
				}
				return m, nil
			}
			m.todoEditor.InputMode = true
			m.todoEditor.Draft = ""
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case " ":
		// Some terminals report space as a dedicated key type (not KeyRunes).
		// Preserve typing spaces while drafting a TODO title.
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.todoEditor.InputMode {
			m.appendTodoDraft(" ")
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil
		}
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.toggleSelectedTodo() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO updated")
		}
		return m, nil
	case "a":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			m.todoEditor.InputMode = true
			m.todoEditor.Draft = ""
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil
		}
		return m, nil
	case "A":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
			if m.todoSelection.Sub >= 0 && m.todoSelection.Sub < len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
				newSubIdx2 := len(m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos)
				return m.openTodoForm(m.todoSelection.Top, m.todoSelection.Sub, newSubIdx2)
			}
			newSubIdx := len(m.workspace.Todos[m.todoSelection.Top].Subtodos)
			return m.openTodoForm(m.todoSelection.Top, newSubIdx, -1)
		}
		return m, nil
	case "X":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && len(m.workspace.Archived) > 0 {
			m.workspace.Archived = []Todo{}
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ Archive cleared")
		}
		return m, nil
	case kb.AddWork:
		return m.openWorkForm(false, -1)
	case kb.AddBreak:
		return m.openWorkForm(true, -1)
	case kb.Edit:
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			return m.openTodoFormForSelection()
		}
		if m.day.Selection.EntryIdx >= 0 && m.day.Selection.EntryIdx < n {
			return m.openWorkForm(m.day.Record.Entries[m.day.Selection.EntryIdx].IsBreak, m.day.Selection.EntryIdx)
		}
		return m.openNotesEditor()
	case kb.Delete:
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			if m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
				m.delete.Day = false
				m.delete.Idx = deleteTodoIdx
				m.delete.PrevState = stateDayView
				m.ui.Current = stateConfirmDelete
			}
			return m, nil
		}
		if m.day.Selection.EntryIdx >= 0 && m.day.Selection.EntryIdx < n {
			m.delete.Day = false
			m.delete.Idx = m.day.Selection.EntryIdx
			m.delete.PrevState = stateDayView
			m.ui.Current = stateConfirmDelete
			return m, nil
		}
		m.delete.Day = true
		m.delete.Idx = -1 // current day
		m.delete.PrevState = stateDayView
		m.ui.Current = stateConfirmDelete
		return m, nil
	case kb.SetStartNow:
		m.day.Record.StartTime = time.Now().Format("15:04")
		m.day.Viewport.SetContent(m.renderDayContent())
		if m.context.UseCases == nil || m.context.UseCases.SetDayTimes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("set day times use case is not configured")} }
		}
		start := m.day.Record.StartTime
		end := m.day.Record.EndTime
		return m, func() tea.Msg {
			_, err := m.context.UseCases.SetDayTimes.Execute(usecase.SetDayTimesInput{
				Date:      m.day.Record.Date,
				StartTime: start,
				EndTime:   end,
			})
			if err != nil {
				return errMsg{err: err}
			}
			return daySavedMsg{label: "✓ Start time set to " + start}
		}
	case kb.SetStartManual:
		return m.openTimeInput(true)
	case kb.SetEndNow:
		m.day.Record.EndTime = time.Now().Format("15:04")
		m.day.Viewport.SetContent(m.renderDayContent())
		if m.context.UseCases == nil || m.context.UseCases.SetDayTimes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("set day times use case is not configured")} }
		}
		start := m.day.Record.StartTime
		end := m.day.Record.EndTime
		return m, func() tea.Msg {
			_, err := m.context.UseCases.SetDayTimes.Execute(usecase.SetDayTimesInput{
				Date:      m.day.Record.Date,
				StartTime: start,
				EndTime:   end,
			})
			if err != nil {
				return errMsg{err: err}
			}
			return daySavedMsg{label: "✓ End time set to " + end}
		}
	case kb.SetEndManual:
		return m.openTimeInput(false)
	case kb.Notes:
		return m.openNotesEditor()
	case kb.TodoOverview:
		if m.day.Selection.DayTab == 0 {
			if m.day.Selection.Pane != 1 {
				m.day.Selection.Pane = 1
			} else {
				m.day.Selection.Pane = 0
				m.exitTodoInputMode()
			}
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil
		}
		m.day.Selection.DayTab = 0
		m.day.Selection.Pane = 1
		m.day.Viewport.GotoTop()
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, nil
	case kb.Export:
		return m, m.exportDayCmd(m.day.Record)
	case kb.ClockStart:
		if !m.clock.Running {
			return m.openClockForm()
		}
		if kb.ClockStart == kb.ClockStop {
			return m.stopClock()
		}
		return m, nil
	case kb.ClockStop:
		if m.clock.Running {
			return m.stopClock()
		}
		return m, nil
	case "esc":
		clockWasRunning := m.clock.Running
		m.clock.Running = false
		m.clock.Task = ""
		m.clock.Project = ""
		m.ui.Current = stateList
		var cmds []tea.Cmd
		cmds = append(cmds, m.loadRecordsCmd())
		if clockWasRunning {
			m.status.Message = "⏱ Clock stopped"
			m.status.IsError = false
			cmds = append(cmds, clearStatusCmd())
		}
		// Move fully-completed todos to the archive, then prune them from active.
		harvested := collectFullyCompleted(m.workspace.Todos)
		pruned := pruneCompletedTodos(m.workspace.Todos)
		if len(pruned) != len(m.workspace.Todos) {
			m.workspace.Archived = append(m.workspace.Archived, harvested...)
			m.workspace.Todos = pruned
			cmds = append(cmds, m.saveWorkspaceTodosCmd(""))
		}
		return m, tea.Batch(cmds...)
	}
	var cmd tea.Cmd
	m.day.Viewport, cmd = m.day.Viewport.Update(msg)
	return m, cmd
}

func (m Model) openTodoFormForSelection() (tea.Model, tea.Cmd) {
	if m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
		return m.openTodoForm(m.todoSelection.Top, m.todoSelection.Sub, m.todoSelection.Sub2)
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
	case "j", "k", "a", "A", "X", " ":
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
	kb := m.context.Config.Keybinds.Day
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
		title := strings.TrimSpace(m.todoEditor.Input.Value())
		if title == "" {
			m.status.Message = "✗ TODO title is required"
			m.status.IsError = true
			return m, clearStatusCmd()
		}
		if m.todoEditor.EditTop >= 0 && m.todoEditor.EditTop < len(m.workspace.Todos) {
			if m.todoEditor.EditSub >= 0 {
				if m.todoEditor.EditSub < len(m.workspace.Todos[m.todoEditor.EditTop].Subtodos) {
					if m.todoEditor.EditSub2 >= 0 && m.todoEditor.EditSub2 < len(m.workspace.Todos[m.todoEditor.EditTop].Subtodos[m.todoEditor.EditSub].Subtodos) {
						m.workspace.Todos[m.todoEditor.EditTop].Subtodos[m.todoEditor.EditSub].Subtodos[m.todoEditor.EditSub2].Title = title
					} else if m.todoEditor.EditSub2 >= 0 {
						m.workspace.Todos[m.todoEditor.EditTop].Subtodos[m.todoEditor.EditSub].Subtodos = append(m.workspace.Todos[m.todoEditor.EditTop].Subtodos[m.todoEditor.EditSub].Subtodos, Todo{
							ID:       newID(),
							Title:    title,
							Subtodos: []Todo{},
						})
						m.todoSelection.Sub2 = len(m.workspace.Todos[m.todoEditor.EditTop].Subtodos[m.todoEditor.EditSub].Subtodos) - 1
					} else {
						m.workspace.Todos[m.todoEditor.EditTop].Subtodos[m.todoEditor.EditSub].Title = title
					}
				} else {
					m.workspace.Todos[m.todoEditor.EditTop].Subtodos = append(m.workspace.Todos[m.todoEditor.EditTop].Subtodos, Todo{
						ID:       newID(),
						Title:    title,
						Subtodos: []Todo{},
					})
					m.todoSelection.Sub = len(m.workspace.Todos[m.todoEditor.EditTop].Subtodos) - 1
					m.todoSelection.Sub2 = -1
				}
				m.todoSelection.Top = m.todoEditor.EditTop
			} else {
				m.workspace.Todos[m.todoEditor.EditTop].Title = title
				m.todoSelection.Top = m.todoEditor.EditTop
				m.todoSelection.Sub = -1
				m.todoSelection.Sub2 = -1
			}
		} else {
			m.workspace.Todos = append(m.workspace.Todos, Todo{
				ID:       newID(),
				Title:    title,
				Subtodos: []Todo{},
			})
			m.todoSelection.Top = len(m.workspace.Todos) - 1
			m.todoSelection.Sub = -1
			m.todoSelection.Sub2 = -1
		}
		m.ui.Current = stateDayView
		m.day.Selection.Pane = 1
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, m.saveWorkspaceTodosCmd("✓ TODO saved")
	case tea.KeyEsc:
		m.ui.Current = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.todoEditor.Input, cmd = m.todoEditor.Input.Update(msg)
	return m, cmd
}

func (m Model) handleWorkFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		return m.focusField((m.workForm.ActiveInput + 1) % m.numFormFields())
	case tea.KeyShiftTab:
		return m.focusField((m.workForm.ActiveInput - 1 + m.numFormFields()) % m.numFormFields())

	case tea.KeyEnter:
		lastField := m.numFormFields() - 1
		if m.workForm.ActiveInput < lastField {
			return m.focusField(m.workForm.ActiveInput + 1)
		}
		// ── Submit ────────────────────────────────────────────────────────────
		task := strings.TrimSpace(m.workForm.TaskInput.Value())
		durStr := strings.TrimSpace(m.workForm.DurationInput.Value())
		if task == "" {
			m.status.Message = "✗ Task name is required"
			m.status.IsError = true
			return m, clearStatusCmd()
		}
		dur, err := parseDuration(durStr)
		if err != nil {
			m.status.Message = "✗ " + err.Error()
			m.status.IsError = true
			return m, clearStatusCmd()
		}
		projectRaw := strings.TrimSpace(m.workForm.ProjectInput.Value())
		isBreak := m.workForm.IsBreakEntry
		editIdx := m.workForm.EditEntryIdx
		wasSplit := false
		originalEntries := append([]WorkEntry(nil), m.day.Record.Entries...)
		mergedBreakIdx := -1
		distributedEntries := []WorkEntry{}

		if editIdx >= 0 && editIdx < len(m.day.Record.Entries) && isBreak {
			// Update existing break entry in-place.
			m.day.Record.Entries[editIdx].Task = task
			m.day.Record.Entries[editIdx].DurationMin = int(dur.Minutes())
			m.day.Selection.EntryIdx = editIdx
		} else if isBreak {
			// For new breaks: merge into an existing break with the same label (case-insensitive).
			taskLower := strings.ToLower(task)
			merged := false
			for i, e := range m.day.Record.Entries {
				if e.IsBreak && strings.ToLower(e.Task) == taskLower {
					m.day.Record.Entries[i].DurationMin += int(dur.Minutes())
					m.day.Selection.EntryIdx = i
					mergedBreakIdx = i
					merged = true
					break
				}
			}
			if !merged {
				entry := WorkEntry{
					ID:          newID(),
					Task:        task,
					DurationMin: int(dur.Minutes()),
					IsBreak:     true,
				}
				m.day.Record.Entries = append(m.day.Record.Entries, entry)
				m.day.Selection.EntryIdx = len(m.day.Record.Entries) - 1
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
			newEntries := make([]WorkEntry, 0, len(projects))
			for i, proj := range projects {
				mins := base
				if i < remainder {
					mins++ // distribute remainder evenly: one extra minute to first N projects
				}
				if mins == 0 {
					continue // skip zero-duration entries when duration < number of projects
				}
				newEntries = append(newEntries, WorkEntry{
					ID:          newID(),
					Task:        task,
					Project:     proj,
					DurationMin: mins,
					IsBreak:     false,
				})
			}
			if len(newEntries) == 0 {
				m.status.Message = "✗ Duration too short to distribute across projects"
				m.status.IsError = true
				return m, clearStatusCmd()
			}
			distributedEntries = append(distributedEntries, newEntries...)
			wasSplit = len(projects) > 1
			if editIdx >= 0 && editIdx < len(m.day.Record.Entries) {
				// Replace the edited entry with the split entries.
				updated := make([]WorkEntry, 0, len(m.day.Record.Entries)-1+len(newEntries))
				updated = append(updated, m.day.Record.Entries[:editIdx]...)
				updated = append(updated, newEntries...)
				updated = append(updated, m.day.Record.Entries[editIdx+1:]...)
				m.day.Record.Entries = updated
				m.day.Selection.EntryIdx = editIdx + len(newEntries) - 1
			} else {
				m.day.Record.Entries = append(m.day.Record.Entries, newEntries...)
				m.day.Selection.EntryIdx = len(m.day.Record.Entries) - 1
			}
		}

		m.ui.Current = stateDayView
		m.day.Viewport.SetContent(m.renderDayContent())
		m.scrollToSelected()

		label := "✓ Work entry logged"
		if editIdx >= 0 && !wasSplit {
			label = "✓ Entry updated"
		} else if isBreak {
			label = "✓ Break logged"
		} else if wasSplit {
			label = "✓ Work entries split across projects"
		}

		if m.context.UseCases == nil ||
			m.context.UseCases.AddWorkEntry == nil ||
			m.context.UseCases.UpdateWorkEntry == nil ||
			m.context.UseCases.DeleteWorkEntry == nil ||
			m.context.UseCases.LoadDayRecord == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("work entry use cases are not configured")} }
		}
		var persistErr error
		date := m.day.Record.Date
		durationMin := int(dur.Minutes())

		if isBreak {
			switch {
			case editIdx >= 0 && editIdx < len(originalEntries):
				_, persistErr = m.context.UseCases.UpdateWorkEntry.Execute(usecase.UpdateWorkEntryInput{
					Date:        date,
					EntryID:     originalEntries[editIdx].ID,
					Task:        task,
					DurationMin: durationMin,
				})
			case mergedBreakIdx >= 0 && mergedBreakIdx < len(m.day.Record.Entries):
				_, persistErr = m.context.UseCases.UpdateWorkEntry.Execute(usecase.UpdateWorkEntryInput{
					Date:        date,
					EntryID:     m.day.Record.Entries[mergedBreakIdx].ID,
					DurationMin: m.day.Record.Entries[mergedBreakIdx].DurationMin,
				})
			case m.day.Selection.EntryIdx >= 0 && m.day.Selection.EntryIdx < len(m.day.Record.Entries):
				out, err := m.context.UseCases.AddWorkEntry.Execute(usecase.AddWorkEntryInput{
					Date:        date,
					Task:        task,
					DurationMin: durationMin,
					IsBreak:     true,
				})
				if err != nil {
					persistErr = err
				} else {
					m.day.Record.Entries[m.day.Selection.EntryIdx].ID = out.EntryID
				}
			default:
				persistErr = fmt.Errorf("unable to resolve break persistence target")
			}
		} else {
			if len(distributedEntries) == 0 {
				persistErr = fmt.Errorf("no entries to persist after project distribution")
			} else if editIdx >= 0 && editIdx < len(originalEntries) {
				if len(distributedEntries) == 1 {
					project := distributedEntries[0].Project
					if project == "" {
						project = "-"
					}
					_, persistErr = m.context.UseCases.UpdateWorkEntry.Execute(usecase.UpdateWorkEntryInput{
						Date:        date,
						EntryID:     originalEntries[editIdx].ID,
						Task:        task,
						Project:     project,
						DurationMin: distributedEntries[0].DurationMin,
					})
				} else {
					_, persistErr = m.context.UseCases.DeleteWorkEntry.Execute(usecase.DeleteWorkEntryInput{
						Date:    date,
						EntryID: originalEntries[editIdx].ID,
					})
					if persistErr == nil {
						for i, entry := range distributedEntries {
							out, addErr := m.context.UseCases.AddWorkEntry.Execute(usecase.AddWorkEntryInput{
								Date:        date,
								Task:        entry.Task,
								Project:     entry.Project,
								DurationMin: entry.DurationMin,
								IsBreak:     false,
							})
							if addErr != nil {
								persistErr = addErr
								break
							}
							newIdx := editIdx + i
							if newIdx >= 0 && newIdx < len(m.day.Record.Entries) {
								m.day.Record.Entries[newIdx].ID = out.EntryID
							}
						}
					}
				}
			} else {
				startIdx := len(m.day.Record.Entries) - len(distributedEntries)
				for i, entry := range distributedEntries {
					out, addErr := m.context.UseCases.AddWorkEntry.Execute(usecase.AddWorkEntryInput{
						Date:        date,
						Task:        entry.Task,
						Project:     entry.Project,
						DurationMin: entry.DurationMin,
						IsBreak:     false,
					})
					if addErr != nil {
						persistErr = addErr
						break
					}
					newIdx := startIdx + i
					if newIdx >= 0 && newIdx < len(m.day.Record.Entries) {
						m.day.Record.Entries[newIdx].ID = out.EntryID
					}
				}
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
			m.day.Viewport.SetContent(m.renderDayContent())
			m.status.Message = "✗ " + persistErr.Error()
			m.status.IsError = true
			return m, clearStatusCmd()
		}
		return m, func() tea.Msg { return daySavedMsg{label: label} }

	case tea.KeyEsc:
		m.ui.Current = stateDayView
		return m, nil
	}

	var cmd tea.Cmd
	switch {
	case m.workForm.ActiveInput == 0:
		m.workForm.TaskInput, cmd = m.workForm.TaskInput.Update(msg)
	case m.workForm.ActiveInput == 1 && !m.workForm.IsBreakEntry:
		m.workForm.ProjectInput, cmd = m.workForm.ProjectInput.Update(msg)
	default:
		m.workForm.DurationInput, cmd = m.workForm.DurationInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleTimeInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.timeForm.Input.Value())
		if !isValidHHMM(val) {
			m.status.Message = "✗ Invalid time — use HH:MM (e.g. 09:30)"
			m.status.IsError = true
			m.ui.Current = stateDayView
			return m, clearStatusCmd()
		}

		// Update local state
		if m.timeForm.IsStart {
			m.day.Record.StartTime = val
		} else {
			m.day.Record.EndTime = val
		}
		m.ui.Current = stateDayView
		m.day.Viewport.SetContent(m.renderDayContent())

		label := "✓ End time set to " + val
		if m.timeForm.IsStart {
			label = "✓ Start time set to " + val
		}

		if m.context.UseCases == nil || m.context.UseCases.SetDayTimes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("set day times use case is not configured")} }
		}
		return m, func() tea.Msg {
			input := usecase.SetDayTimesInput{
				Date:      m.day.Record.Date,
				StartTime: m.day.Record.StartTime,
				EndTime:   m.day.Record.EndTime,
			}
			_, err := m.context.UseCases.SetDayTimes.Execute(input)
			if err != nil {
				return errMsg{err: err}
			}
			return daySavedMsg{label: label}
		}

	case tea.KeyEsc:
		m.ui.Current = stateDayView
		return m, nil
	}
	switch msg.String() {
	case "r":
		// Reset time
		if m.timeForm.IsStart {
			m.day.Record.StartTime = ""
		} else {
			m.day.Record.EndTime = ""
		}
		m.ui.Current = stateDayView
		m.day.Viewport.SetContent(m.renderDayContent())

		label := "✓ End time cleared"
		if m.timeForm.IsStart {
			label = "✓ Start time cleared"
		}

		if m.context.UseCases == nil || m.context.UseCases.SetDayTimes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("set day times use case is not configured")} }
		}
		return m, func() tea.Msg {
			input := usecase.SetDayTimesInput{
				Date:      m.day.Record.Date,
				StartTime: m.day.Record.StartTime,
				EndTime:   m.day.Record.EndTime,
			}
			_, err := m.context.UseCases.SetDayTimes.Execute(input)
			if err != nil {
				return errMsg{err: err}
			}
			return daySavedMsg{label: label}
		}
	}
	var cmd tea.Cmd
	m.timeForm.Input, cmd = m.timeForm.Input.Update(msg)
	return m, cmd
}

func (m Model) handleNotesEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		notes := m.day.Notes.Value()
		m.day.Record.Notes = notes
		m.ui.Current = stateDayView
		m.day.Viewport.SetContent(m.renderDayContent())

		if m.context.UseCases == nil || m.context.UseCases.UpdateNotes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("update notes use case is not configured")} }
		}
		return m, func() tea.Msg {
			input := usecase.UpdateNotesInput{
				Date:  m.day.Record.Date,
				Notes: notes,
			}
			_, err := m.context.UseCases.UpdateNotes.Execute(input)
			if err != nil {
				return errMsg{err: err}
			}
			return daySavedMsg{label: "✓ Notes saved"}
		}

	case tea.KeyEsc:
		m.ui.Current = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.day.Notes, cmd = m.day.Notes.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.delete.Day {
			// Delete the whole day file/record.
			var date string
			if m.delete.PrevState == stateDayView {
				date = m.day.Record.Date
			} else if m.delete.Idx >= 0 && m.delete.Idx < len(m.listState.Records) {
				date = m.listState.Records[m.delete.Idx].Date
			}

			if m.context.UseCases == nil || m.context.UseCases.DeleteDayRecord == nil {
				return m, func() tea.Msg { return errMsg{err: fmt.Errorf("delete day record use case is not configured")} }
			}
			if date == "" {
				m.ui.Current = m.delete.PrevState
				return m, nil
			}
			return m, func() tea.Msg {
				if err := m.context.UseCases.DeleteDayRecord.Execute(usecase.DeleteDayRecordInput{Date: date}); err != nil {
					return errMsg{err: err}
				}
				return dayDeletedMsg{}
			}
		}
		// Delete a single entry from the current day.
		if m.delete.Idx == deleteTodoIdx {
			if m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
				if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 >= 0 && m.todoSelection.Sub < len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
					level2 := m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos
					if m.todoSelection.Sub2 >= 0 && m.todoSelection.Sub2 < len(level2) {
						m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos = append(level2[:m.todoSelection.Sub2], level2[m.todoSelection.Sub2+1:]...)
					}
					m.todoSelection.Sub2 = -1
				} else if m.todoSelection.Sub >= 0 && m.todoSelection.Sub < len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
					st := m.workspace.Todos[m.todoSelection.Top].Subtodos
					m.workspace.Todos[m.todoSelection.Top].Subtodos = append(st[:m.todoSelection.Sub], st[m.todoSelection.Sub+1:]...)
					m.todoSelection.Sub = -1
					m.todoSelection.Sub2 = -1
				} else {
					m.workspace.Todos = append(m.workspace.Todos[:m.todoSelection.Top], m.workspace.Todos[m.todoSelection.Top+1:]...)
					if m.todoSelection.Top >= len(m.workspace.Todos) {
						m.todoSelection.Top = len(m.workspace.Todos) - 1
					}
					if m.todoSelection.Top < 0 {
						m.todoSelection.Sub = -1
						m.todoSelection.Sub2 = -1
					}
				}
			}
			m.ui.Current = stateDayView
			m.day.Selection.Pane = 1
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO deleted")
		}
		if m.delete.Idx >= 0 && m.delete.Idx < len(m.day.Record.Entries) {
			entryID := m.day.Record.Entries[m.delete.Idx].ID

			// Update local state
			m.day.Record.Entries = append(
				m.day.Record.Entries[:m.delete.Idx],
				m.day.Record.Entries[m.delete.Idx+1:]...,
			)
			if m.day.Selection.EntryIdx >= len(m.day.Record.Entries) {
				m.day.Selection.EntryIdx = len(m.day.Record.Entries) - 1
			}

			m.ui.Current = stateDayView
			m.day.Viewport.SetContent(m.renderDayContent())

			if m.context.UseCases == nil || m.context.UseCases.DeleteWorkEntry == nil {
				return m, func() tea.Msg { return errMsg{err: fmt.Errorf("delete work entry use case is not configured")} }
			}
			return m, func() tea.Msg {
				input := usecase.DeleteWorkEntryInput{
					Date:    m.day.Record.Date,
					EntryID: entryID,
				}
				_, err := m.context.UseCases.DeleteWorkEntry.Execute(input)
				if err != nil {
					return errMsg{err: err}
				}
				return daySavedMsg{label: "✓ Entry deleted"}
			}
		}
		m.ui.Current = stateDayView
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, nil
	case "n", "N", "esc":
		m.ui.Current = m.delete.PrevState
	}
	return m, nil
}

func (m Model) handleDateInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.dateForm.Input.Blur()
		m.ui.Current = stateList
		return m, nil
	case "enter":
		raw := strings.TrimSpace(m.dateForm.Input.Value())
		if raw == "" {
			raw = time.Now().Format("2006-01-02")
		}
		// Accept YYYY-MM-DD
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			m.status.Message = "✗ Invalid date — use YYYY-MM-DD"
			m.status.IsError = true
			m.ui.Current = stateList
			return m, clearStatusCmd()
		}
		m.dateForm.Input.Blur()
		dateStr := date.Format("2006-01-02")
		rec, err := m.loadDayRecord(dateStr)
		if err != nil {
			m.status.Message = "✗ " + err.Error()
			m.status.IsError = true
			m.ui.Current = stateList
			return m, clearStatusCmd()
		}
		return m.openDayView(rec)
	}
	var cmd tea.Cmd
	m.dateForm.Input, cmd = m.dateForm.Input.Update(msg)
	return m, cmd
}

func (m Model) handleWorkspacePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.context.Config.Workspaces)
	switch msg.String() {
	case "j", "down":
		if m.workspacePicker.Index < n-1 {
			m.workspacePicker.Index++
		}
		return m, nil
	case "k", "up":
		if m.workspacePicker.Index > 0 {
			m.workspacePicker.Index--
		}
		return m, nil
	case "enter":
		if m.workspacePicker.Index >= 0 && m.workspacePicker.Index < n {
			return m.switchWorkspace(m.context.Config.Workspaces[m.workspacePicker.Index].Name)
		}
	case "esc", m.context.Config.Keybinds.List.Quit:
		m.ui.Current = stateList
	}
	return m, nil
}

func (m Model) handleClockFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	const numFields = 2
	switch msg.Type {
	case tea.KeyTab:
		return m.focusField((m.workForm.ActiveInput + 1) % numFields)
	case tea.KeyShiftTab:
		return m.focusField((m.workForm.ActiveInput - 1 + numFields) % numFields)

	case tea.KeyEnter:
		if m.workForm.ActiveInput < numFields-1 {
			return m.focusField(m.workForm.ActiveInput + 1)
		}
		// Submit — start the clock.
		task := strings.TrimSpace(m.workForm.TaskInput.Value())
		if task == "" {
			m.status.Message = "✗ Task name is required"
			m.status.IsError = true
			return m, clearStatusCmd()
		}
		m.clock.Task = task
		m.clock.Project = strings.TrimSpace(m.workForm.ProjectInput.Value())
		m.clock.Start = time.Now()
		m.clock.Running = true
		m.ui.Current = stateDayView
		m.day.Selection.DayTab = 0
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, clockTickCmd()

	case tea.KeyEsc:
		m.ui.Current = stateDayView
		m.day.Selection.DayTab = 0
		return m, nil
	}

	var cmd tea.Cmd
	if m.workForm.ActiveInput == 0 {
		m.workForm.TaskInput, cmd = m.workForm.TaskInput.Update(msg)
	} else {
		m.workForm.ProjectInput, cmd = m.workForm.ProjectInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleStatsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	const numStatsTabs = 4
	switch msg.String() {
	case "esc", m.context.Config.Keybinds.List.Quit:
		m.ui.Current = stateList
		return m, nil
	case "left":
		if m.stats.Tab > 0 {
			m.stats.Tab--
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderStatsTabContent())
		}
		return m, nil
	case "right":
		if m.stats.Tab < numStatsTabs-1 {
			m.stats.Tab++
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderStatsTabContent())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.day.Viewport, cmd = m.day.Viewport.Update(msg)
	return m, cmd
}

func (m Model) handleWeekViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", m.context.Config.Keybinds.List.Quit:
		m.ui.Current = stateList
		return m, nil
	case "left":
		m.weekOffset--
		m.day.Viewport.GotoTop()
		m.day.Viewport.SetContent(m.renderWeekContent())
		return m, nil
	case "right":
		if m.weekOffset < 0 {
			m.weekOffset++
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderWeekContent())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.day.Viewport, cmd = m.day.Viewport.Update(msg)
	return m, cmd
}
