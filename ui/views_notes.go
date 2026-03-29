package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// viewNotesEditor renders the full-screen notes editor.
func (m Model) viewNotesEditor() string {
	subtitle := m.day.Record.Date
	if t, err := m.day.Record.ParseDate(); err == nil {
		subtitle = t.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader(m.appTitle(), subtitle)
	footer := m.renderFooter([][2]string{
		{"ctrl+s", "save"},
		{"esc", "cancel"},
	})
	editor := editorBorderStyle.
		Width(m.window.Width - 4).
		Render(m.day.Notes.View())
	return lipgloss.JoinVertical(lipgloss.Left, header, editor, footer)
}

