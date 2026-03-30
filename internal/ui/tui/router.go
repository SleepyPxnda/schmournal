package tui

import tea "github.com/charmbracelet/bubbletea"

// routeKeyMsg dispatches keyboard input to the active screen handler.
func (m Model) routeKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.ui.Current {
	case stateList:
		return m.handleListKey(msg)
	case stateDayView:
		return m.handleDayViewKey(msg)
	case stateWorkForm:
		return m.handleWorkFormKey(msg)
	case stateClockForm:
		return m.handleClockFormKey(msg)
	case stateTimeInput:
		return m.handleTimeInputKey(msg)
	case stateNotesEditor:
		return m.handleNotesEditorKey(msg)
	case stateTodoForm:
		return m.handleTodoFormKey(msg)
	case stateConfirmDelete:
		return m.handleConfirmDeleteKey(msg)
	case stateDateInput:
		return m.handleDateInputKey(msg)
	case stateWeekView:
		return m.handleWeekViewKey(msg)
	case stateWorkspacePicker:
		return m.handleWorkspacePickerKey(msg)
	case stateStats:
		return m.handleStatsKey(msg)
	}
	return m, nil
}

// routeSubModelMsg forwards non-key messages to the active Bubble component.
func (m Model) routeSubModelMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.ui.Current {
	case stateList:
		var cmd tea.Cmd
		m.listState.Model, cmd = m.listState.Model.Update(msg)
		return m, cmd
	case stateDayView, stateWeekView, stateStats:
		var cmd tea.Cmd
		m.day.Viewport, cmd = m.day.Viewport.Update(msg)
		return m, cmd
	case stateWorkForm:
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
	case stateClockForm:
		var cmd tea.Cmd
		if m.workForm.ActiveInput == 0 {
			m.workForm.TaskInput, cmd = m.workForm.TaskInput.Update(msg)
		} else {
			m.workForm.ProjectInput, cmd = m.workForm.ProjectInput.Update(msg)
		}
		return m, cmd
	case stateTimeInput:
		var cmd tea.Cmd
		m.timeForm.Input, cmd = m.timeForm.Input.Update(msg)
		return m, cmd
	case stateNotesEditor:
		var cmd tea.Cmd
		m.day.Notes, cmd = m.day.Notes.Update(msg)
		return m, cmd
	case stateTodoForm:
		var cmd tea.Cmd
		m.todoEditor.Input, cmd = m.todoEditor.Input.Update(msg)
		return m, cmd
	case stateDateInput:
		var cmd tea.Cmd
		m.dateForm.Input, cmd = m.dateForm.Input.Update(msg)
		return m, cmd
	}
	return m, nil
}

