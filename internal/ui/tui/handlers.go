package tui

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
		switch listActionForKey(msg.String(), kb) {
		case listActionQuit:
			return m, tea.Quit
		case listActionOpenToday:
			return m.openDayViewToday()
		case listActionOpenDate:
			return m.openDateInput()
		case listActionOpenSelected:
			if item, ok := m.listState.Model.SelectedItem().(dayListItem); ok {
				return m.openDayView(item.rec)
			}
		case listActionDeleteSelected:
			idx := m.listState.Model.Index()
			if idx >= 0 && idx < len(m.listState.Records) {
				m.delete.Day = true
				m.delete.Idx = idx
				m.delete.PrevState = stateList
				m.ui.Current = stateConfirmDelete
				return m, nil
			}
		case listActionExportSelected:
			if item, ok := m.listState.Model.SelectedItem().(dayListItem); ok {
				return m, m.exportDayCmd(item.rec)
			}
		case listActionOpenWeekView:
			return m.openWeekView()
		case listActionOpenStatsView:
			return m.openStatsView()
		case listActionOpenWorkspacePicker:
			return m.openWorkspacePicker()
		}
	}
	var cmd tea.Cmd
	m.listState.Model, cmd = m.listState.Model.Update(msg)
	return m, cmd
}

func (m Model) handleDayViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.day.Record.Entries)
	inTodoPane := m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1

	updated, cmd, handled := m.handleDayTodoInlineInput(msg, inTodoPane)
	if handled {
		return updated, cmd
	}

	if inTodoPane && m.todoEditor.InputMode && isBlockedTodoInputKey(msg.String()) {
		return m, nil
	}

	updated, cmd, handled = m.handleDayNavigationKey(msg.String(), n)
	if handled {
		return updated, cmd
	}

	updated, cmd, handled = m.handleDayConfiguredCommandKey(msg.String(), n)
	if handled {
		return updated, cmd
	}

	var viewportCmd tea.Cmd
	m.day.Viewport, viewportCmd = m.day.Viewport.Update(msg)
	return m, viewportCmd
}

func (m Model) handleDayTodoInlineInput(msg tea.KeyMsg, inTodoPane bool) (Model, tea.Cmd, bool) {
	if !inTodoPane {
		return m, nil, false
	}

	switch msg.Type {
	case tea.KeyRunes:
		if m.todoEditor.InputMode {
			m.appendTodoDraft(string(msg.Runes))
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil, true
		}
		// In TODO navigation mode, begin inline drafting immediately for printable
		// characters that are not bound to other day-view commands.
		if m.shouldStartInlineTodoDraft(msg) {
			m.todoEditor.InputMode = true
			m.todoEditor.Draft = string(msg.Runes)
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil, true
		}
	case tea.KeyBackspace:
		if m.todoEditor.InputMode {
			m.backspaceTodoDraft()
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil, true
		}
		if m.deleteSelectedTodoNow() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO deleted"), true
		}
		return m, nil, true
	}

	return m, nil, false
}

func isBlockedTodoInputKey(key string) bool {
	switch key {
	case "tab", "shift+tab", "delete", "up", "down", "left", "right", "shift+up", "shift+down":
		return true
	default:
		return false
	}
}

func (m Model) handleDayNavigationKey(key string, n int) (Model, tea.Cmd, bool) {
	switch key {
	case "left":
		if m.day.Selection.DayTab > 0 {
			m.day.Selection.DayTab--
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil, true
	case "right":
		if m.day.Selection.DayTab < 1 {
			m.day.Selection.DayTab++
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil, true
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
		return m, nil, true
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
		return m, nil, true
	case "tab":
		if m.day.Selection.DayTab == 0 {
			if m.day.Selection.Pane == 1 && m.todoEditor.InputMode {
				return m, nil, true
			}
			if m.day.Selection.Pane == 1 && !m.todoEditor.InputMode {
				if m.indentSelectedTodo() {
					m.day.Viewport.SetContent(m.renderDayContent())
					return m, m.saveWorkspaceTodosCmd("✓ TODO indented"), true
				}
				// In focused TODO navigation mode, tab is reserved for indenting and does not cycle panes.
				return m, nil, true
			}
			m.day.Selection.Pane = (m.day.Selection.Pane + 1) % 2
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil, true
	case "shift+tab":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.outdentSelectedTodo() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO outdented"), true
		}
		return m, nil, true
	case "shift+up":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.moveSelectedTodoDelta(-1) {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO moved up"), true
		}
		return m, nil, true
	case "shift+down":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.moveSelectedTodoDelta(1) {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO moved down"), true
		}
		return m, nil, true
	case "delete":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.deleteSelectedTodoNow() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO deleted"), true
		}
		return m, nil, true
	case "enter":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			if m.todoEditor.InputMode {
				saved := m.commitTodoDraft()
				m.exitTodoInputMode()
				m.day.Viewport.SetContent(m.renderDayContent())
				if saved {
					return m, m.saveWorkspaceTodosCmd("✓ TODO saved"), true
				}
				return m, nil, true
			}
			m.todoEditor.InputMode = true
			m.todoEditor.Draft = ""
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil, true
	case " ":
		// Some terminals report space as a dedicated key type (not KeyRunes).
		// Preserve typing spaces while drafting a TODO title.
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.todoEditor.InputMode {
			m.appendTodoDraft(" ")
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil, true
		}
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.toggleSelectedTodo() {
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, m.saveWorkspaceTodosCmd("✓ TODO updated"), true
		}
		return m, nil, true
	case "a":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			m.todoEditor.InputMode = true
			m.todoEditor.Draft = ""
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil, true
		}
		return m, nil, true
	case "A":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
			if m.todoSelection.Sub >= 0 && m.todoSelection.Sub < len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
				newSubIdx2 := len(m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos)
				modelOut, cmd := m.openTodoForm(m.todoSelection.Top, m.todoSelection.Sub, newSubIdx2)
				return modelOut.(Model), cmd, true
			}
			newSubIdx := len(m.workspace.Todos[m.todoSelection.Top].Subtodos)
			modelOut, cmd := m.openTodoForm(m.todoSelection.Top, newSubIdx, -1)
			return modelOut.(Model), cmd, true
		}
		return m, nil, true
	case "X":
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 && len(m.workspace.Archived) > 0 {
			return m, m.clearArchiveCmd("✓ Archive cleared"), true
		}
		return m, nil, true
	}

	return m, nil, false
}

func (m Model) handleDayConfiguredCommandKey(key string, n int) (Model, tea.Cmd, bool) {
	kb := m.context.Config.Keybinds.Day
	switch key {
	case kb.AddWork:
		modelOut, cmd := m.openWorkForm(false, -1)
		return modelOut.(Model), cmd, true
	case kb.AddBreak:
		modelOut, cmd := m.openWorkForm(true, -1)
		return modelOut.(Model), cmd, true
	case kb.Edit:
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			modelOut, cmd := m.openTodoFormForSelection()
			return modelOut.(Model), cmd, true
		}
		if m.day.Selection.EntryIdx >= 0 && m.day.Selection.EntryIdx < n {
			modelOut, cmd := m.openWorkForm(m.day.Record.Entries[m.day.Selection.EntryIdx].IsBreak, m.day.Selection.EntryIdx)
			return modelOut.(Model), cmd, true
		}
		modelOut, cmd := m.openNotesEditor()
		return modelOut.(Model), cmd, true
	case kb.Delete:
		if m.day.Selection.DayTab == 0 && m.day.Selection.Pane == 1 {
			if m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
				m.delete.Day = false
				m.delete.Idx = deleteTodoIdx
				m.delete.PrevState = stateDayView
				m.ui.Current = stateConfirmDelete
			}
			return m, nil, true
		}
		if m.day.Selection.EntryIdx >= 0 && m.day.Selection.EntryIdx < n {
			m.delete.Day = false
			m.delete.Idx = m.day.Selection.EntryIdx
			m.delete.PrevState = stateDayView
			m.ui.Current = stateConfirmDelete
			return m, nil, true
		}
		m.delete.Day = true
		m.delete.Idx = -1 // current day
		m.delete.PrevState = stateDayView
		m.ui.Current = stateConfirmDelete
		return m, nil, true
	case kb.SetStartNow:
		m.day.Record.StartTime = time.Now().Format("15:04")
		m.day.Viewport.SetContent(m.renderDayContent())
		if m.context.UseCases == nil || m.context.UseCases.SetDayTimes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("set day times use case is not configured")} }, true
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
		}, true
	case kb.SetStartManual:
		modelOut, cmd := m.openTimeInput(true)
		return modelOut.(Model), cmd, true
	case kb.SetEndNow:
		m.day.Record.EndTime = time.Now().Format("15:04")
		m.day.Viewport.SetContent(m.renderDayContent())
		if m.context.UseCases == nil || m.context.UseCases.SetDayTimes == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("set day times use case is not configured")} }, true
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
		}, true
	case kb.SetEndManual:
		modelOut, cmd := m.openTimeInput(false)
		return modelOut.(Model), cmd, true
	case kb.Notes:
		modelOut, cmd := m.openNotesEditor()
		return modelOut.(Model), cmd, true
	case kb.TodoOverview:
		if m.day.Selection.DayTab == 0 {
			if m.day.Selection.Pane != 1 {
				m.day.Selection.Pane = 1
			} else {
				m.day.Selection.Pane = 0
				m.exitTodoInputMode()
			}
			m.day.Viewport.SetContent(m.renderDayContent())
			return m, nil, true
		}
		m.day.Selection.DayTab = 0
		m.day.Selection.Pane = 1
		m.day.Viewport.GotoTop()
		m.day.Viewport.SetContent(m.renderDayContent())
		return m, nil, true
	case kb.Export:
		return m, m.exportDayCmd(m.day.Record), true
	case kb.ClockStart:
		if !m.clock.Running {
			modelOut, cmd := m.openClockForm()
			return modelOut.(Model), cmd, true
		}
		if kb.ClockStart == kb.ClockStop {
			modelOut, cmd := m.stopClock()
			return modelOut.(Model), cmd, true
		}
		return m, nil, true
	case kb.ClockStop:
		if m.clock.Running {
			modelOut, cmd := m.stopClock()
			return modelOut.(Model), cmd, true
		}
		return m, nil, true
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
		if m.context.UseCases != nil && m.context.UseCases.ManageTodos != nil {
			cmds = append(cmds, m.archiveCompletedTodosCmd(""))
		} else {
			// Fallback path: preserve existing behavior if use case wiring is unavailable.
			harvested := collectFullyCompleted(m.workspace.Todos)
			pruned := pruneCompletedTodos(m.workspace.Todos)
			if len(pruned) != len(m.workspace.Todos) {
				m.workspace.Archived = append(m.workspace.Archived, harvested...)
				m.workspace.Todos = pruned
				cmds = append(cmds, m.saveWorkspaceTodosCmd(""))
			}
		}
		return m, tea.Batch(cmds...), true
	}

	return m, nil, false
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
		if _, err := parseDuration(durStr); err != nil {
			m.status.Message = "✗ " + err.Error()
			m.status.IsError = true
			return m, clearStatusCmd()
		}
		if m.context.UseCases == nil || m.context.UseCases.SubmitWorkForm == nil {
			return m, func() tea.Msg { return errMsg{err: fmt.Errorf("submit work form use case is not configured")} }
		}

		date := m.day.Record.Date
		projectRaw := strings.TrimSpace(m.workForm.ProjectInput.Value())
		isBreak := m.workForm.IsBreakEntry
		editIdx := m.workForm.EditEntryIdx
		return m, func() tea.Msg {
			out, err := m.context.UseCases.SubmitWorkForm.Execute(usecase.SubmitWorkFormInput{
				Date:       date,
				Task:       task,
				ProjectRaw: projectRaw,
				Duration:   durStr,
				IsBreak:    isBreak,
				EditEntry:  editIdx,
			})
			if err != nil {
				return errMsg{err: err}
			}
			return workFormSubmittedMsg{
				record:   toUIDayRecord(out.Record),
				label:    out.Label,
				entryIdx: out.SelectedEntryIdx,
			}
		}

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
	switch workspacePickerActionForKey(msg.String(), m.context.Config.Keybinds.List.Quit) {
	case workspacePickerActionMoveDown:
		if m.workspacePicker.Index < n-1 {
			m.workspacePicker.Index++
		}
		return m, nil
	case workspacePickerActionMoveUp:
		if m.workspacePicker.Index > 0 {
			m.workspacePicker.Index--
		}
		return m, nil
	case workspacePickerActionConfirm:
		if m.workspacePicker.Index >= 0 && m.workspacePicker.Index < n {
			return m.switchWorkspace(m.context.Config.Workspaces[m.workspacePicker.Index].Name)
		}
	case workspacePickerActionCancel:
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
	switch statsActionForKey(msg.String(), m.context.Config.Keybinds.List.Quit) {
	case statsActionBack:
		m.ui.Current = stateList
		return m, nil
	case statsActionLeft:
		if m.stats.Tab > 0 {
			m.stats.Tab--
			m.day.Viewport.GotoTop()
			m.day.Viewport.SetContent(m.renderStatsTabContent())
		}
		return m, nil
	case statsActionRight:
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
	switch weekViewActionForKey(msg.String(), m.context.Config.Keybinds.List.Quit) {
	case weekViewActionBack:
		m.ui.Current = stateList
		return m, nil
	case weekViewActionLeft:
		m.weekOffset--
		m.day.Viewport.GotoTop()
		m.day.Viewport.SetContent(m.renderWeekContent())
		return m, nil
	case weekViewActionRight:
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
