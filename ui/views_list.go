package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// viewList renders the main list view showing all journal days.
func (m Model) viewList() string {
	subtitle := time.Now().Format("Mon, 02 Jan 2006")
	if m.context.ActiveWorkspace != "" {
		subtitle = m.context.ActiveWorkspace + "  ·  " + subtitle
	}
	header := m.renderHeader(m.appTitle(), subtitle)
	kb := m.context.Config.Keybinds.List
	var footerKeys [][2]string
	footerKeys = append(footerKeys,
		[2]string{kb.OpenToday, "open today"},
		[2]string{kb.OpenDate, "open date"},
		[2]string{"enter", "view"},
		[2]string{kb.WeekView, "week"},
		[2]string{kb.StatsView, "stats"},
		[2]string{kb.Delete, "delete"},
		[2]string{kb.Export, "export"},
		[2]string{"/", "filter"},
	)
	if len(m.context.Config.Workspaces) > 0 {
		footerKeys = append(footerKeys, [2]string{kb.SwitchWorkspace, "workspace"})
	}
	footerKeys = append(footerKeys, [2]string{"esc", "quit"})
	footer := m.renderFooter(footerKeys)
	sections := []string{header, m.renderStats()}
	if eom := m.renderEOMBanner(); eom != "" {
		sections = append(sections, eom)
	}
	sections = append(sections, m.listState.Model.View(), footer)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
