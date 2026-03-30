package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHeader(title, subtitle string) string {
	return m.renderHeaderStyled(title, subtitle, headerSubtitleStyle)
}

func (m Model) renderHeaderStyled(title, subtitle string, subtitleStyle lipgloss.Style) string {
	left := headerTitleStyle.Render(title)
	right := subtitleStyle.Render(subtitle)
	gap := m.window.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	top := left + strings.Repeat(" ", gap) + right
	sep := separatorStyle.Render(strings.Repeat("─", m.window.Width))
	return top + "\n" + sep
}

// appTitle returns the application title with the version number appended.
func (m Model) appTitle() string {
	if m.ui.Version != "" && m.ui.Version != "dev" {
		ver := m.ui.Version
		if strings.HasPrefix(ver, "v") {
			ver = ver[1:]
		}
		return "📔  Schmournal  v" + ver
	}
	return "📔  Schmournal"
}

// joinKeyLabels joins two key labels with "/" for footer display.
// If either label is empty, only the non-empty one is shown; if both are empty "?" is returned.
func joinKeyLabels(a, b string) string {
	if a == "" && b == "" {
		return "?"
	}
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "/" + b
}

func (m Model) renderFooter(keys [][2]string) string {
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k[0])+" "+helpStyle.Render(k[1]))
	}
	help := strings.Join(parts, helpStyle.Render("  ·  "))

	if m.status.Message != "" {
		s := m.status.Message
		if m.status.IsError {
			s = statusErrorStyle.Render(s)
		} else {
			s = statusSuccessStyle.Render(s)
		}
		pad := m.window.Width - lipgloss.Width(help) - lipgloss.Width(s) - 2
		if pad < 1 {
			pad = 1
		}
		return help + strings.Repeat(" ", pad) + s
	}
	return help
}

func (m Model) renderDayTabBar() string {
	tabs := []string{"📋  Work Log", "📊  Summary"}
	var parts []string
	for i, label := range tabs {
		if i == m.day.Selection.DayTab {
			parts = append(parts, activeTabStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+label+" "))
		}
	}
	bar := strings.Join(parts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.window.Width))
	return bar + "\n" + sep
}

func (m Model) renderStatsTabBar() string {
	tabs := []string{"🔥  Overview", "📆  Monthly", "📈  Yearly", "🏆  All-time"}
	var parts []string
	for i, label := range tabs {
		if i == m.stats.Tab {
			parts = append(parts, activeTabStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+label+" "))
		}
	}
	bar := strings.Join(parts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.window.Width))
	return bar + "\n" + sep
}

