package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fgrohme/tui-journal/journal"
)

func loadRecords() tea.Msg {
	records, err := journal.LoadAll()
	if err != nil {
		return errMsg{err: err}
	}
	return recordsLoadedMsg{records: records}
}

func loadWeeklyGoals() tea.Msg {
	goals, err := journal.LoadWeeklyGoals()
	if err != nil {
		return errMsg{err: err}
	}
	return weekGoalsLoadedMsg{goals: goals}
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
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
