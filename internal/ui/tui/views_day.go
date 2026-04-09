package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// viewDayView renders the detailed day view with work log and summary tabs.
func (m Model) viewDayView() string {
	subtitle := m.day.Record.Date
	subtitleStyle := headerSubtitleStyle
	if t, err := m.day.Record.ParseDate(); err == nil {
		subtitle = t.Format("Monday, 02 January 2006")
		if !m.effectiveIsWorkDay(t) {
			subtitleStyle = headerNonWorkDaySubtitleStyle
		}
	}
	header := m.renderHeaderStyled(m.appTitle(), subtitle, subtitleStyle)
	tabBar := m.renderDayTabBar()

	var footerKeys [][2]string
	kb := m.context.Config.Keybinds.Day
	clockEnabled := m.context.Config.Modules.ClockEnabled
	todoEnabled := m.context.Config.Modules.TodoEnabled
	if m.day.Selection.DayTab == 0 {
		editLabel := "edit"
		deleteLabel := "del"
		if todoEnabled && m.day.Selection.Pane == 1 {
			editLabel = "edit todo"
			deleteLabel = "del todo"
		}
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{"j/k", "select"},
		}
		if todoEnabled {
			footerKeys = append(footerKeys,
				[2]string{"tab", "pane/indent"},
				[2]string{"S-tab", "outdent"},
				[2]string{"S-↑/↓", "reorder"},
			)
		}
		footerKeys = append(footerKeys,
			[2]string{kb.AddWork, "work"},
			[2]string{kb.AddBreak, "break"},
			[2]string{kb.Edit, editLabel},
			[2]string{kb.Delete, deleteLabel},
		)
		if todoEnabled {
			footerKeys = append(footerKeys,
				[2]string{"backspace", "del todo"},
				[2]string{"space", "toggle todo"},
			)
		}
		footerKeys = append(footerKeys,
			[2]string{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			[2]string{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			[2]string{kb.Notes, "notes"},
		)
		if todoEnabled {
			footerKeys = append(footerKeys, [2]string{kb.TodoOverview, "todo pane"})
		}
		if clockEnabled {
			clockKey := kb.ClockStart
			clockLabel := "start clock"
			if m.clock.Running {
				clockKey = kb.ClockStop
				clockLabel = "stop clock"
			}
			footerKeys = append(footerKeys, [2]string{clockKey, clockLabel})
		}
		footerKeys = append(footerKeys,
			[2]string{"esc", "back"},
		)
	} else {
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{"esc", "back"},
		}
	}
	footer := m.renderFooter(footerKeys)
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, m.day.Viewport.View(), footer)
}
