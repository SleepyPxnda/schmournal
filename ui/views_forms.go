package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// viewWorkLogForm renders the form for adding or editing work/break entries.
func (m Model) viewWorkLogForm() string {
	var badge, taskLabel string
	dateStr := m.day.Record.Date
	if m.workForm.IsBreakEntry {
		badge = breakLogBadgeStyle.Render(" Log Break ")
		taskLabel = "Break label"
	} else {
		badge = workLogBadgeStyle.Render(" Log Work ")
		taskLabel = "What did you work on?"
	}
	header := m.renderHeader(m.appTitle(), badge+
		headerSubtitleStyle.Render("  "+dateStr))
	footer := m.renderFooter([][2]string{
		{"tab", "next field"},
		{"enter", "save"},
		{"esc", "cancel"},
	})

	formWidth := m.window.Width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8

	m.workForm.TaskInput.Width = inputWidth
	m.workForm.ProjectInput.Width = inputWidth
	m.workForm.DurationInput.Width = inputWidth

	renderBox := func(input textinput.Model, active bool) string {
		if active {
			return formActiveInputStyle.Width(inputWidth).Render(input.View())
		}
		return formInactiveInputStyle.Width(inputWidth).Render(input.View())
	}

	taskBox := renderBox(m.workForm.TaskInput, m.workForm.ActiveInput == 0)
	durBox := renderBox(m.workForm.DurationInput, m.workForm.ActiveInput == m.numFormFields()-1)

	var body string
	if m.workForm.IsBreakEntry {
		body = formLabelStyle.Render(taskLabel) + "\n" +
			taskBox + "\n\n" +
			formLabelStyle.Render("Duration") +
			formHintStyle.Render("  e.g. 1h 30m · 45m · 2h") + "\n" +
			durBox
	} else {
		projBox := renderBox(m.workForm.ProjectInput, m.workForm.ActiveInput == 1)
		body = formLabelStyle.Render(taskLabel) + "\n" +
			taskBox + "\n\n" +
			formLabelStyle.Render("Project") +
			formHintStyle.Render("  optional") + "\n" +
			projBox + "\n\n" +
			formLabelStyle.Render("Duration") +
			formHintStyle.Render("  e.g. 1h 30m · 45m · 2h") + "\n" +
			durBox
	}

	form := formBoxStyle.Width(formWidth).Render(body)

	fh := lipgloss.Height(form)
	topPad := (m.contentHeight() - fh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.window.Width).Align(lipgloss.Center).Render(form)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

// viewClockForm renders the form for starting a clock timer.
func (m Model) viewClockForm() string {
	badge := workLogBadgeStyle.Render(" Start Clock ")
	dateStr := m.day.Record.Date
	header := m.renderHeader(m.appTitle(), badge+
		headerSubtitleStyle.Render("  "+dateStr))
	footer := m.renderFooter([][2]string{
		{"tab", "next field"},
		{"enter", "start"},
		{"esc", "cancel"},
	})

	formWidth := m.window.Width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8

	m.workForm.TaskInput.Width = inputWidth
	m.workForm.ProjectInput.Width = inputWidth

	renderBox := func(input textinput.Model, active bool) string {
		if active {
			return formActiveInputStyle.Width(inputWidth).Render(input.View())
		}
		return formInactiveInputStyle.Width(inputWidth).Render(input.View())
	}

	body := formLabelStyle.Render("What are you working on?") + "\n" +
		renderBox(m.workForm.TaskInput, m.workForm.ActiveInput == 0) + "\n\n" +
		formLabelStyle.Render("Project") +
		formHintStyle.Render("  optional · comma-separate for multiple projects") + "\n" +
		renderBox(m.workForm.ProjectInput, m.workForm.ActiveInput == 1)

	form := formBoxStyle.Width(formWidth).Render(body)

	fh := lipgloss.Height(form)
	topPad := (m.contentHeight() - fh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.window.Width).Align(lipgloss.Center).Render(form)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

// viewTimeInput renders the form for setting start or end time.
func (m Model) viewTimeInput() string {
	label := "Set Start Time"
	badge := workLogBadgeStyle.Render(" Start ")
	if !m.timeForm.IsStart {
		label = "Set Finish Time"
		badge = breakLogBadgeStyle.Render(" Finish ")
	}
	header := m.renderHeader(m.appTitle(), badge)
	footer := m.renderFooter([][2]string{
		{"enter", "confirm"},
		{"r", "reset"},
		{"esc", "cancel"},
	})

	m.timeForm.Input.Width = 12
	inputBox := formActiveInputStyle.Width(14).Render(m.timeForm.Input.View())

	dialog := formBoxStyle.Render(
		formLabelStyle.Render(label) + "\n" +
			formHintStyle.Render("24-hour format  ·  e.g. 09:30, 14:00") + "\n\n" +
			inputBox,
	)

	dh := lipgloss.Height(dialog)
	topPad := (m.contentHeight() - dh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.window.Width).Align(lipgloss.Center).Render(dialog)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

// viewTodoForm renders the form for creating a new todo.
func (m Model) viewTodoForm() string {
	header := m.renderHeader(m.appTitle(), "Todo")
	footer := m.renderFooter([][2]string{
		{"enter", "save"},
		{"esc", "cancel"},
	})
	formWidth := m.window.Width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8
	m.todoEditor.Input.Width = inputWidth
	box := formBoxStyle.Width(formWidth).Render(
		formLabelStyle.Render("Todo title") + "\n" +
			formActiveInputStyle.Width(inputWidth).Render(m.todoEditor.Input.View()) + "\n\n" +
			dayViewMutedStyle.Render("Tip: press Shift+a from a parent todo to add a subtodo"),
	)
	fh := lipgloss.Height(box)
	topPad := (m.contentHeight() - fh) / 2
	if topPad < 0 {
		topPad = 0
	}
	centered := lipgloss.NewStyle().Width(m.window.Width).Align(lipgloss.Center).Render(box)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

// viewDateInput renders the form for opening a specific date.
func (m Model) viewDateInput() string {
	header := m.renderHeader(m.appTitle(), "Open Day")
	prompt := dayViewLabelStyle.Render("Enter date:") + "  " + m.dateForm.Input.View()
	box := formBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			formLabelStyle.Render("Open or create a journal entry for any day"),
			"",
			prompt,
			"",
			dayViewMutedStyle.Render("enter  confirm  ·  esc  cancel"),
		),
	)
	bh := lipgloss.Height(box)
	ch := m.contentHeight()
	topPad := (ch - bh) / 2
	if topPad < 0 {
		topPad = 0
	}
	pad := strings.Repeat("\n", topPad)
	footer := m.renderFooter([][2]string{{"enter", "open"}, {"esc", "cancel"}})
	return lipgloss.JoinVertical(lipgloss.Left, header, pad+box, footer)
}

// viewConfirmDelete renders the confirmation dialog for deleting items.
func (m Model) viewConfirmDelete() string {
	var subject string
	if m.delete.Day {
		if m.delete.PrevState == stateDayView {
			subject = m.day.Record.Date
		} else if m.delete.Idx >= 0 && m.delete.Idx < len(m.listState.Records) {
			subject = m.listState.Records[m.delete.Idx].Date
		}
		subject = "the day " + subject
	} else if m.delete.Idx == deleteTodoIdx {
		if m.todoSelection.Top >= 0 && m.todoSelection.Top < len(m.workspace.Todos) {
			if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 >= 0 &&
				m.todoSelection.Sub < len(m.workspace.Todos[m.todoSelection.Top].Subtodos) &&
				m.todoSelection.Sub2 < len(m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos) {
				subject = `todo "` + m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos[m.todoSelection.Sub2].Title + `"`
			} else if m.todoSelection.Sub >= 0 && m.todoSelection.Sub < len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
				subject = `todo "` + m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Title + `"`
			} else {
				subject = `todo "` + m.workspace.Todos[m.todoSelection.Top].Title + `"`
			}
		} else {
			subject = "this todo"
		}
	} else {
		if m.delete.Idx >= 0 && m.delete.Idx < len(m.day.Record.Entries) {
			subject = `entry "` + m.day.Record.Entries[m.delete.Idx].Task + `"`
		} else {
			subject = "this entry"
		}
	}
	header := m.renderHeader(m.appTitle(), "Delete")

	dialog := confirmBoxStyle.Render(
		confirmTitleStyle.Render(fmt.Sprintf("Delete %s?", subject)) +
			"\n\n  " +
			confirmYesStyle.Render("[y]") + helpStyle.Render(" yes") +
			"    " +
			confirmNoStyle.Render("[n]") + helpStyle.Render(" no / esc"),
	)

	dh := lipgloss.Height(dialog)
	topPad := (m.contentHeight() - dh) / 2
	if topPad < 0 {
		topPad = 0
	}

	centered := lipgloss.NewStyle().Width(m.window.Width).Align(lipgloss.Center).Render(dialog)
	return header + "\n" + strings.Repeat("\n", topPad) + centered
}

// viewWorkspacePicker renders the workspace selection dialog.
func (m Model) viewWorkspacePicker() string {
	header := m.renderHeader(m.appTitle(), "Switch Workspace")
	innerW := 36
	if m.window.Width-8 > innerW {
		innerW = m.window.Width / 2
	}
	if innerW > 60 {
		innerW = 60
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))
	var rows []string
	rows = append(rows, formLabelStyle.Render("Select a workspace:"))
	rows = append(rows, div)
	for i, ws := range m.context.Config.Workspaces {
		cursor := "  "
		if i == m.workspacePicker.Index {
			cursor = "▶ "
		}
		label := ws.Name
		if ws.Name == m.context.ActiveWorkspace {
			label += "  " + statusSuccessStyle.Render("✓")
		}
		line := cursor + label
		if i == m.workspacePicker.Index {
			line = selectedEntryStyle.Width(innerW).Render(line)
		} else {
			line = normalEntryStyle.Render(line)
		}
		rows = append(rows, line)
	}
	rows = append(rows, div)
	rows = append(rows, dayViewMutedStyle.Render("j/k  navigate  ·  enter  switch  ·  esc  cancel"))
	box := formBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	bh := lipgloss.Height(box)
	ch := m.contentHeight()
	topPad := (ch - bh) / 2
	if topPad < 0 {
		topPad = 0
	}
	centered := lipgloss.NewStyle().Width(m.window.Width).Align(lipgloss.Center).Render(box)
	footer := m.renderFooter([][2]string{{"j/k", "navigate"}, {"enter", "switch"}, {"esc", "cancel"}})
	return lipgloss.JoinVertical(lipgloss.Left, header, strings.Repeat("\n", topPad)+centered, footer)
}






