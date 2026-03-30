package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/internal/application/usecase"
)

func (m Model) loadRecordsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.context.UseCases == nil || m.context.UseCases.LoadAllDayRecords == nil {
			return errMsg{err: fmt.Errorf("load all day records use case is not configured")}
		}
		records, err := m.context.UseCases.LoadAllDayRecords.ExecuteDTO()
		if err != nil {
			return errMsg{err: err}
		}
		return recordsLoadedMsg{records: toUIDayRecords(records)}
	}
}

func (m Model) loadWorkspaceTodosCmd() tea.Cmd {
	return func() tea.Msg {
		if m.context.UseCases == nil || m.context.UseCases.LoadWorkspaceTodos == nil {
			return errMsg{err: fmt.Errorf("load workspace todos use case is not configured")}
		}
		workspace := m.context.ActiveWorkspace
		if workspace == "" {
			workspace = "default"
		}
		todos, err := m.context.UseCases.LoadWorkspaceTodos.ExecuteDTO(usecase.LoadWorkspaceTodosInput{
			Workspace: workspace,
		})
		if err != nil {
			return errMsg{err: err}
		}
		return workspaceTodosLoadedMsg{todos: toUIWorkspaceTodos(todos)}
	}
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// clockTickCmd returns a command that fires a clockTickMsg after one second.
// It is re-issued on every tick while the clock is running.
func clockTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return clockTickMsg{}
	})
}

func (m Model) saveDayCmd(label string) tea.Cmd {
	rec := m.day.Record
	return func() tea.Msg {
		if m.context.UseCases == nil || m.context.UseCases.SaveDayRecord == nil {
			return errMsg{err: fmt.Errorf("save day record use case is not configured")}
		}
		if err := m.context.UseCases.SaveDayRecord.ExecuteDTO(usecase.SaveDayRecordDTOInput{
			Record: toUseCaseDayRecord(rec),
		}); err != nil {
			return errMsg{err: err}
		}
		return daySavedMsg{label: label}
	}
}

func (m Model) saveWorkspaceTodosCmd(label string) tea.Cmd {
	todos := WorkspaceTodos{Todos: m.workspace.Todos}
	return func() tea.Msg {
		if m.context.UseCases == nil || m.context.UseCases.SaveWorkspaceTodos == nil {
			return errMsg{err: fmt.Errorf("save workspace todos use case is not configured")}
		}
		workspace := m.context.ActiveWorkspace
		if workspace == "" {
			workspace = "default"
		}
		if err := m.context.UseCases.SaveWorkspaceTodos.ExecuteDTO(usecase.SaveWorkspaceTodosDTOInput{
			Workspace: workspace,
			Todos:     toUseCaseWorkspaceTodos(todos),
		}); err != nil {
			return errMsg{err: err}
		}
		return daySavedMsg{label: label}
	}
}

func (m Model) collectCompletedTodosCmd(label string) tea.Cmd {
	return func() tea.Msg {
		if m.context.UseCases == nil || m.context.UseCases.ManageTodos == nil || m.context.UseCases.LoadWorkspaceTodos == nil {
			return errMsg{err: fmt.Errorf("todo management use cases are not configured")}
		}
		workspace := m.context.ActiveWorkspace
		if workspace == "" {
			workspace = "default"
		}

		output, err := m.context.UseCases.ManageTodos.ArchiveCompletedTodos(usecase.ArchiveCompletedTodosInput{
			Workspace: workspace,
		})
		if err != nil {
			return errMsg{err: err}
		}

		todos, err := m.context.UseCases.LoadWorkspaceTodos.ExecuteDTO(usecase.LoadWorkspaceTodosInput{
			Workspace: workspace,
		})
		if err != nil {
			return errMsg{err: err}
		}
		return workspaceTodosManagedMsg{
			todos:          toUIWorkspaceTodos(todos),
			completedToday: toUITodos(output.ArchivedTodos),
			label:          label,
		}
	}
}

