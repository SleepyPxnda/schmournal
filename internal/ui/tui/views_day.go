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
	if m.day.Selection.DayTab == 0 {
		clockKey := kb.ClockStart
		clockLabel := "start clock"
		if m.clock.Running {
			clockKey = kb.ClockStop
			clockLabel = "stop clock"
		}
		editLabel := "edit"
		deleteLabel := "del"
		if m.day.Selection.Pane == 1 {
			editLabel = "edit todo"
			deleteLabel = "del todo"
		}
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{"j/k", "select"},
			{"tab", "pane/indent"},
			{"S-tab", "outdent"},
			{"S-↑/↓", "reorder"},
			{kb.AddWork, "work"},
			{kb.AddBreak, "break"},
			{kb.Edit, editLabel},
			{kb.Delete, deleteLabel},
			{"backspace", "del todo"},
			{"space", "toggle todo"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{kb.Notes, "notes"},
			{kb.TodoOverview, "todo pane"},
			{clockKey, clockLabel},
			{kb.Export, "export"},
			{"esc", "back"},
		}
	} else {
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{kb.Export, "export"},
			{"esc", "back"},
		}
	}
	footer := m.renderFooter(footerKeys)
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, m.day.Viewport.View(), footer)
}
