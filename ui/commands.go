package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sleepypxnda/schmournal/journal"
)

func loadRecords() tea.Msg {
	records, err := journal.LoadAll()
	if err != nil {
		return errMsg{err: err}
	}
	return recordsLoadedMsg{records: records}
}

func loadWorkspaceTodos() tea.Msg {
	todos, err := journal.LoadWorkspaceTodos()
	if err != nil {
		return errMsg{err: err}
	}
	return workspaceTodosLoadedMsg{todos: todos}
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
	rec := m.dayRecord
	return func() tea.Msg {
		if err := journal.Save(rec); err != nil {
			return errMsg{err: err}
		}
		return daySavedMsg{label: label}
	}
}

func (m Model) saveWorkspaceTodosCmd(label string) tea.Cmd {
	todos := journal.WorkspaceTodos{Todos: m.workspaceTodos}
	return func() tea.Msg {
		if err := journal.SaveWorkspaceTodos(todos); err != nil {
			return errMsg{err: err}
		}
		return daySavedMsg{label: label}
	}
}
