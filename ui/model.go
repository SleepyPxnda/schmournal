package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fgrohme/tui-journal/config"
	"github.com/fgrohme/tui-journal/journal"
)

// ── View states ───────────────────────────────────────────────────────────────

type viewState int

const (
	stateList viewState = iota
	stateDayView
	stateWorkForm
	stateTimeInput
	stateNotesEditor
	stateConfirmDelete
	stateDateInput
	stateWeekView
)

const (
	headerHeight = 2 // title line + separator
	footerHeight = 1
	statsHeight  = 3 // week bar + progress bar + separator
)

// ── Messages ──────────────────────────────────────────────────────────────────

type recordsLoadedMsg struct{ records []journal.DayRecord }
type daySavedMsg struct{ label string }
type dayDeletedMsg struct{}
type exportedMsg struct{ path string }
type clearStatusMsg struct{}
type errMsg struct{ err error }

// ── List item ─────────────────────────────────────────────────────────────────

type dayListItem struct{ rec journal.DayRecord }

func (d dayListItem) FilterValue() string {
	parts := []string{d.rec.Date}
	for _, e := range d.rec.Entries {
		if e.Project != "" {
			parts = append(parts, e.Project)
		}
		parts = append(parts, e.Task)
	}
	if d.rec.Notes != "" {
		parts = append(parts, d.rec.Notes)
	}
	return strings.Join(parts, " ")
}

func (d dayListItem) Title() string {
	t, err := d.rec.ParseDate()
	if err != nil {
		return d.rec.Date
	}
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "✦ " + t.Format("Monday, 02 January 2006")
	}
	return t.Format("Monday, 02 January 2006")
}

func (d dayListItem) Description() string { return d.rec.Summary() }

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	state  viewState
	width  int
	height int
	ready  bool

	cfg config.Config

	list    list.Model
	records []journal.DayRecord

	dayRecord     journal.DayRecord
	selectedEntry int // index into dayRecord.Entries; -1 = no selection
	dayViewTab    int // 0 = Work Log, 1 = Summary

	taskInput     textinput.Model
	projectInput  textinput.Model
	durationInput textinput.Model
	activeInput   int
	isBreakEntry  bool
	editEntryIdx  int // -1 = new, >=0 = editing existing entry

	textarea textarea.Model // notes editor

	timeInput      textinput.Model
	timeInputStart bool

	dateInput textinput.Model // for opening/creating any day

	deleteDay bool // true = confirm delete whole day, false = confirm delete entry
	deleteIdx int  // index in records (deleteDay) or entries (!deleteDay)
	prevState viewState

	viewport viewport.Model

	weekOffset int // 0 = current week, -1 = last week, etc.

	statusMsg string
	isError   bool
}

func (m Model) contentHeight() int {
	return m.height - headerHeight - footerHeight
}

func newDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.NormalTitle = listNormalTitle
	d.Styles.NormalDesc = listNormalDesc
	d.Styles.SelectedTitle = listSelectedTitle
	d.Styles.SelectedDesc = listSelectedDesc
	d.Styles.DimmedTitle = listDimmedTitle
	d.Styles.DimmedDesc = listDimmedDesc
	d.Styles.FilterMatch = listFilterMatch
	return d
}

// New constructs the initial model using the provided configuration.
func New(cfg config.Config) Model {
	l := list.New([]list.Item{}, newDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color(cBlue))
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))

	ta := textarea.New()
	ta.Placeholder = "Start writing…"
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(cSubtext0))

	taskIn := textinput.New()
	taskIn.Placeholder = "e.g. Feature development, meeting, code review…"
	taskIn.CharLimit = 120
	taskIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	taskIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	taskIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	durIn := textinput.New()
	durIn.Placeholder = "e.g. 1h 30m, 45m, 2h"
	durIn.CharLimit = 20
	durIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	durIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	durIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	projIn := textinput.New()
	projIn.Placeholder = "e.g. MyApp, Backend, Infra  (optional)"
	projIn.CharLimit = 80
	projIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	projIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	projIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	timeIn := textinput.New()
	timeIn.Placeholder = "09:00"
	timeIn.CharLimit = 5
	timeIn.Width = 10
	timeIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	timeIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	timeIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	dateIn := textinput.New()
	dateIn.Placeholder = "YYYY-MM-DD"
	dateIn.CharLimit = 10
	dateIn.Width = 14
	dateIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	dateIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	dateIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	return Model{
		cfg:           cfg,
		state:         stateList,
		list:          l,
		textarea:      ta,
		taskInput:     taskIn,
		projectInput:  projIn,
		durationInput: durIn,
		timeInput:     timeIn,
		dateInput:     dateIn,
		selectedEntry: -1,
		editEntryIdx:  -1,
	}
}

func (m Model) Init() tea.Cmd {
	return loadRecords
}

// ── Commands ──────────────────────────────────────────────────────────────────

func loadRecords() tea.Msg {
	records, err := journal.LoadAll()
	if err != nil {
		return errMsg{err: err}
	}
	return recordsLoadedMsg{records: records}
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

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		ch := m.contentHeight()
		vpH := ch - 2 // 2 lines for tab bar + separator
		if !m.ready {
			m.viewport = viewport.New(m.width, vpH)
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpH
		}
		listH := ch - statsHeight
		if m.renderEOMBanner() != "" {
			listH--
		}
		m.list.SetSize(m.width, listH)
		m.textarea.SetWidth(m.width - 4)
		m.textarea.SetHeight(ch - 2)
		return m, nil

	case recordsLoadedMsg:
		m.records = msg.records
		items := make([]list.Item, len(m.records))
		for i, r := range m.records {
			items[i] = dayListItem{rec: r}
		}
		m.list.SetItems(items)
		return m, nil

	case daySavedMsg:
		m.statusMsg = msg.label
		m.isError = false
		return m, tea.Batch(loadRecords, clearStatusCmd())

	case dayDeletedMsg:
		m.statusMsg = "✓ Day deleted"
		m.isError = false
		m.state = stateList
		return m, tea.Batch(loadRecords, clearStatusCmd())

	case exportedMsg:
		display := msg.path
		if home, err := os.UserHomeDir(); err == nil {
			display = strings.Replace(display, home, "~", 1)
		}
		m.statusMsg = "✓ Exported → " + display
		m.isError = false
		return m, clearStatusCmd()

	case errMsg:
		m.statusMsg = "✗ " + msg.err.Error()
		m.isError = true
		return m, clearStatusCmd()

	case clearStatusMsg:
		m.statusMsg = ""
		m.isError = false
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.state {
		case stateList:
			return m.handleListKey(msg)
		case stateDayView:
			return m.handleDayViewKey(msg)
		case stateWorkForm:
			return m.handleWorkFormKey(msg)
		case stateTimeInput:
			return m.handleTimeInputKey(msg)
		case stateNotesEditor:
			return m.handleNotesEditorKey(msg)
		case stateConfirmDelete:
			return m.handleConfirmDeleteKey(msg)
		case stateDateInput:
			return m.handleDateInputKey(msg)
		case stateWeekView:
			return m.handleWeekViewKey(msg)
		}
	}

	// Forward non-key messages to active sub-model.
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateDayView, stateWeekView:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case stateWorkForm:
		var cmd tea.Cmd
		switch {
		case m.activeInput == 0:
			m.taskInput, cmd = m.taskInput.Update(msg)
		case m.activeInput == 1 && !m.isBreakEntry:
			m.projectInput, cmd = m.projectInput.Update(msg)
		default:
			m.durationInput, cmd = m.durationInput.Update(msg)
		}
		return m, cmd
	case stateTimeInput:
		var cmd tea.Cmd
		m.timeInput, cmd = m.timeInput.Update(msg)
		return m, cmd
	case stateNotesEditor:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	case stateDateInput:
		var cmd tea.Cmd
		m.dateInput, cmd = m.dateInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// ── Key handlers ──────────────────────────────────────────────────────────────

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtering := m.list.FilterState() == list.Filtering
	if !filtering {
		kb := m.cfg.Keybinds.List
		switch msg.String() {
		case kb.Quit, "esc":
			return m, tea.Quit
		case kb.OpenToday:
			return m.openDayViewToday()
		case kb.OpenDate:
			return m.openDateInput()
		case "enter":
			if item, ok := m.list.SelectedItem().(dayListItem); ok {
				return m.openDayView(item.rec)
			}
		case kb.Delete:
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.records) {
				m.deleteDay = true
				m.deleteIdx = idx
				m.prevState = stateList
				m.state = stateConfirmDelete
				return m, nil
			}
		case kb.AddWork:
			return m.openWorkFormForToday(false)
		case kb.AddBreak:
			return m.openWorkFormForToday(true)
		case kb.Export:
			if item, ok := m.list.SelectedItem().(dayListItem); ok {
				rec := item.rec
				return m, func() tea.Msg {
					path, err := journal.SaveExport(rec)
					if err != nil {
						return errMsg{err: err}
					}
					return exportedMsg{path: path}
				}
			}
		case kb.WeekView:
			return m.openWeekView()
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleWeekViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := m.cfg.Keybinds
	switch msg.String() {
	case "esc", kb.List.Quit:
		m.state = stateList
		return m, nil
	case "left", kb.Week.PrevWeek:
		m.weekOffset--
		m.viewport.GotoTop()
		m.viewport.SetContent(m.renderWeekContent())
		return m, nil
	case "right", kb.Week.NextWeek:
		if m.weekOffset < 0 {
			m.weekOffset++
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderWeekContent())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleDayViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.dayRecord.Entries)
	kb := m.cfg.Keybinds.Day
	switch msg.String() {
	case "left":
		if m.dayViewTab > 0 {
			m.dayViewTab--
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "right":
		if m.dayViewTab < 1 {
			m.dayViewTab++
			m.viewport.GotoTop()
			m.viewport.SetContent(m.renderDayContent())
		}
		return m, nil
	case "j", "down":
		if m.dayViewTab == 0 && m.selectedEntry < n-1 {
			m.selectedEntry++
			m.viewport.SetContent(m.renderDayContent())
			m.scrollToSelected()
		}
		return m, nil
	case "k", "up":
		if m.dayViewTab == 0 && m.selectedEntry > 0 {
			m.selectedEntry--
			m.viewport.SetContent(m.renderDayContent())
			m.scrollToSelected()
		}
		return m, nil
	case kb.AddWork:
		return m.openWorkForm(false, -1)
	case kb.AddBreak:
		return m.openWorkForm(true, -1)
	case kb.Edit:
		if m.selectedEntry >= 0 && m.selectedEntry < n {
			return m.openWorkForm(m.dayRecord.Entries[m.selectedEntry].IsBreak, m.selectedEntry)
		}
		return m.openNotesEditor()
	case kb.Delete:
		if m.selectedEntry >= 0 && m.selectedEntry < n {
			m.deleteDay = false
			m.deleteIdx = m.selectedEntry
			m.prevState = stateDayView
			m.state = stateConfirmDelete
			return m, nil
		}
		m.deleteDay = true
		m.deleteIdx = -1 // current day
		m.prevState = stateDayView
		m.state = stateConfirmDelete
		return m, nil
	case kb.SetStartNow:
		m.dayRecord.StartTime = time.Now().Format("15:04")
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ Start time set to " + m.dayRecord.StartTime)
	case kb.SetStartManual:
		return m.openTimeInput(true)
	case kb.SetEndNow:
		m.dayRecord.EndTime = time.Now().Format("15:04")
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ End time set to " + m.dayRecord.EndTime)
	case kb.SetEndManual:
		return m.openTimeInput(false)
	case kb.Notes:
		return m.openNotesEditor()
	case kb.Export:
		rec := m.dayRecord
		return m, func() tea.Msg {
			path, err := journal.SaveExport(rec)
			if err != nil {
				return errMsg{err: err}
			}
			return exportedMsg{path: path}
		}
	case "esc":
		m.state = stateList
		return m, loadRecords
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleWorkFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		return m.focusField((m.activeInput + 1) % m.numFormFields())
	case tea.KeyShiftTab:
		return m.focusField((m.activeInput - 1 + m.numFormFields()) % m.numFormFields())

	case tea.KeyEnter:
		lastField := m.numFormFields() - 1
		if m.activeInput < lastField {
			return m.focusField(m.activeInput + 1)
		}
		// ── Submit ────────────────────────────────────────────────────────────
		task := strings.TrimSpace(m.taskInput.Value())
		durStr := strings.TrimSpace(m.durationInput.Value())
		if task == "" {
			m.statusMsg = "✗ Task name is required"
			m.isError = true
			return m, clearStatusCmd()
		}
		dur, err := journal.ParseDuration(durStr)
		if err != nil {
			m.statusMsg = "✗ " + err.Error()
			m.isError = true
			return m, clearStatusCmd()
		}
		projectRaw := strings.TrimSpace(m.projectInput.Value())
		isBreak := m.isBreakEntry
		editIdx := m.editEntryIdx
		wasSplit := false

		if editIdx >= 0 && editIdx < len(m.dayRecord.Entries) && isBreak {
			// Update existing break entry in-place.
			m.dayRecord.Entries[editIdx].Task = task
			m.dayRecord.Entries[editIdx].DurationMin = int(dur.Minutes())
			m.selectedEntry = editIdx
		} else if isBreak {
			// For new breaks: merge into an existing break with the same label (case-insensitive).
			taskLower := strings.ToLower(task)
			merged := false
			for i, e := range m.dayRecord.Entries {
				if e.IsBreak && strings.ToLower(e.Task) == taskLower {
					m.dayRecord.Entries[i].DurationMin += int(dur.Minutes())
					m.selectedEntry = i
					merged = true
					break
				}
			}
			if !merged {
				entry := journal.WorkEntry{
					ID:          journal.NewID(),
					Task:        task,
					DurationMin: int(dur.Minutes()),
					IsBreak:     true,
				}
				m.dayRecord.Entries = append(m.dayRecord.Entries, entry)
				m.selectedEntry = len(m.dayRecord.Entries) - 1
			}
		} else {
			// Split comma-separated projects and distribute duration evenly.
			// This applies both to new entries and edited entries.
			rawParts := strings.Split(projectRaw, ",")
			projects := make([]string, 0, len(rawParts))
			for _, p := range rawParts {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					projects = append(projects, trimmed)
				}
			}
			if len(projects) == 0 {
				projects = []string{""}
			}
			totalMin := int(dur.Minutes())
			base := totalMin / len(projects)
			remainder := totalMin % len(projects)
			newEntries := make([]journal.WorkEntry, 0, len(projects))
			for i, proj := range projects {
				mins := base
				if i < remainder {
					mins++ // distribute remainder evenly: one extra minute to first N projects
				}
				if mins == 0 {
					continue // skip zero-duration entries when duration < number of projects
				}
				newEntries = append(newEntries, journal.WorkEntry{
					ID:          journal.NewID(),
					Task:        task,
					Project:     proj,
					DurationMin: mins,
					IsBreak:     false,
				})
			}
			if len(newEntries) == 0 {
				m.statusMsg = "✗ Duration too short to distribute across projects"
				m.isError = true
				return m, clearStatusCmd()
			}
			wasSplit = len(projects) > 1
			if editIdx >= 0 && editIdx < len(m.dayRecord.Entries) {
				// Replace the edited entry with the split entries.
				updated := make([]journal.WorkEntry, 0, len(m.dayRecord.Entries)-1+len(newEntries))
				updated = append(updated, m.dayRecord.Entries[:editIdx]...)
				updated = append(updated, newEntries...)
				updated = append(updated, m.dayRecord.Entries[editIdx+1:]...)
				m.dayRecord.Entries = updated
				m.selectedEntry = editIdx + len(newEntries) - 1
			} else {
				m.dayRecord.Entries = append(m.dayRecord.Entries, newEntries...)
				m.selectedEntry = len(m.dayRecord.Entries) - 1
			}
		}

		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		m.scrollToSelected()

		label := "✓ Work entry logged"
		if editIdx >= 0 && !wasSplit {
			label = "✓ Entry updated"
		} else if isBreak {
			label = "✓ Break logged"
		} else if wasSplit {
			label = "✓ Work entries split across projects"
		}
		return m, m.saveDayCmd(label)

	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}

	var cmd tea.Cmd
	switch {
	case m.activeInput == 0:
		m.taskInput, cmd = m.taskInput.Update(msg)
	case m.activeInput == 1 && !m.isBreakEntry:
		m.projectInput, cmd = m.projectInput.Update(msg)
	default:
		m.durationInput, cmd = m.durationInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleTimeInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.timeInput.Value())
		if !isValidHHMM(val) {
			m.statusMsg = "✗ Invalid time — use HH:MM (e.g. 09:30)"
			m.isError = true
			m.state = stateDayView
			return m, clearStatusCmd()
		}
		if m.timeInputStart {
			m.dayRecord.StartTime = val
		} else {
			m.dayRecord.EndTime = val
		}
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		label := "✓ End time set to " + val
		if m.timeInputStart {
			label = "✓ Start time set to " + val
		}
		return m, m.saveDayCmd(label)
	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.timeInput, cmd = m.timeInput.Update(msg)
	return m, cmd
}

func (m Model) handleNotesEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		m.dayRecord.Notes = m.textarea.Value()
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ Notes saved")
	case tea.KeyEsc:
		m.state = stateDayView
		return m, nil
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteDay {
			// Delete the whole day file.
			var path string
			if m.prevState == stateDayView {
				path = m.dayRecord.Path
			} else if m.deleteIdx >= 0 && m.deleteIdx < len(m.records) {
				path = m.records[m.deleteIdx].Path
			}
			if path == "" {
				m.state = m.prevState
				return m, nil
			}
			return m, func() tea.Msg {
				if err := journal.Delete(path); err != nil {
					return errMsg{err: err}
				}
				return dayDeletedMsg{}
			}
		}
		// Delete a single entry from the current day.
		if m.deleteIdx >= 0 && m.deleteIdx < len(m.dayRecord.Entries) {
			m.dayRecord.Entries = append(
				m.dayRecord.Entries[:m.deleteIdx],
				m.dayRecord.Entries[m.deleteIdx+1:]...,
			)
			if m.selectedEntry >= len(m.dayRecord.Entries) {
				m.selectedEntry = len(m.dayRecord.Entries) - 1
			}
		}
		m.state = stateDayView
		m.viewport.SetContent(m.renderDayContent())
		return m, m.saveDayCmd("✓ Entry deleted")
	case "n", "N", "esc":
		m.state = m.prevState
	}
	return m, nil
}

// ── Navigation helpers ────────────────────────────────────────────────────────

func (m Model) openDayView(rec journal.DayRecord) (tea.Model, tea.Cmd) {
	// Reload from disk to get freshest data.
	fresh, err := journal.Load(rec.Path)
	if err != nil {
		fresh = rec
	}
	m.dayRecord = fresh
	if m.dayRecord.Path == "" {
		m.dayRecord.Path = rec.Path
	}
	m.dayViewTab = 0
	m.selectedEntry = -1
	if len(m.dayRecord.Entries) > 0 {
		m.selectedEntry = 0
	}
	m.state = stateDayView
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderDayContent())
	return m, nil
}

func (m Model) openWeekView() (tea.Model, tea.Cmd) {
	m.weekOffset = 0
	m.state = stateWeekView
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderWeekContent())
	return m, nil
}

func (m Model) openDayViewToday() (tea.Model, tea.Cmd) {
	path, err := journal.TodayPath()
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	rec, err := journal.Load(path)
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	if rec.Date == "" {
		rec.Date = time.Now().Format("2006-01-02")
	}
	rec.Path = path
	return m.openDayView(rec)
}

func (m Model) openWorkForm(isBreak bool, editIdx int) (tea.Model, tea.Cmd) {
	m.isBreakEntry = isBreak
	m.editEntryIdx = editIdx
	m.taskInput.SetValue("")
	m.projectInput.SetValue("")
	m.durationInput.SetValue("")
	if editIdx >= 0 && editIdx < len(m.dayRecord.Entries) {
		e := m.dayRecord.Entries[editIdx]
		m.taskInput.SetValue(e.Task)
		m.projectInput.SetValue(e.Project)
		m.durationInput.SetValue(journal.FormatDuration(e.Duration()))
	}
	if isBreak {
		m.taskInput.Placeholder = "e.g. Lunch, coffee break, walk…"
	} else {
		m.taskInput.Placeholder = "e.g. Feature development, meeting, code review…"
	}
	m.state = stateWorkForm
	return m.focusField(0)
}

func (m Model) openWorkFormForToday(isBreak bool) (tea.Model, tea.Cmd) {
	path, err := journal.TodayPath()
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	rec, err := journal.Load(path)
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	if rec.Date == "" {
		rec.Date = time.Now().Format("2006-01-02")
	}
	rec.Path = path
	m.dayRecord = rec
	if m.selectedEntry < 0 || m.selectedEntry >= len(rec.Entries) {
		m.selectedEntry = len(rec.Entries) - 1
	}
	// After form submission we'll land in stateDayView.
	return m.openWorkForm(isBreak, -1)
}

func (m Model) openNotesEditor() (tea.Model, tea.Cmd) {
	m.textarea.SetValue(m.dayRecord.Notes)
	blinkCmd := m.textarea.Focus()
	m.state = stateNotesEditor
	return m, blinkCmd
}

func (m Model) openTimeInput(isStart bool) (tea.Model, tea.Cmd) {
	m.timeInputStart = isStart
	m.timeInput.SetValue(time.Now().Format("15:04"))
	m.timeInput.CursorEnd()
	m.state = stateTimeInput
	return m, m.timeInput.Focus()
}

// numFormFields returns how many fields the current work-log form has.
func (m Model) numFormFields() int {
	if m.isBreakEntry {
		return 2
	}
	return 3
}

// focusField blurs all inputs and focuses the one at index n.
func (m Model) focusField(n int) (Model, tea.Cmd) {
	m.taskInput.Blur()
	m.projectInput.Blur()
	m.durationInput.Blur()
	m.activeInput = n
	switch {
	case n == 0:
		return m, m.taskInput.Focus()
	case n == 1 && !m.isBreakEntry:
		return m, m.projectInput.Focus()
	default:
		return m, m.durationInput.Focus()
	}
}

func isValidHHMM(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}

// scrollToSelected adjusts the viewport so the selected entry is visible.
func (m *Model) scrollToSelected() {
	const entryStartLine = 7
	if m.selectedEntry < 0 {
		return
	}
	targetLine := entryStartLine + m.selectedEntry
	if targetLine < m.viewport.YOffset {
		m.viewport.YOffset = targetLine
	} else if targetLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = targetLine - m.viewport.Height + 1
	}
}

// ── Tab bar ───────────────────────────────────────────────────────────────────

func (m Model) renderTabBar() string {
	tabs := []string{"📋  Work Log", "📊  Summary"}
	var parts []string
	for i, label := range tabs {
		if i == m.dayViewTab {
			parts = append(parts, activeTabStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+label+" "))
		}
	}
	bar := strings.Join(parts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.width))
	return bar + "\n" + sep
}

// ── Day view content renderer ─────────────────────────────────────────────────

func (m Model) renderDayContent() string {
	if m.dayViewTab == 1 {
		return m.renderSummaryContent()
	}
	return m.renderWorkLogContent()
}

func (m Model) renderWorkLogContent() string {
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder

	b.WriteString("\n")

	// ── Work Day section ──────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("🕐  Work Day") + "\n")
	b.WriteString(div + "\n")

	start := m.dayRecord.StartTime
	end := m.dayRecord.EndTime
	startDisplay := start
	if startDisplay == "" {
		startDisplay = dayViewMutedStyle.Render("—")
	} else {
		startDisplay = dayViewValueStyle.Render(start)
	}
	endDisplay := end
	if endDisplay == "" {
		endDisplay = dayViewMutedStyle.Render("—")
	} else {
		endDisplay = dayViewValueStyle.Render(end)
	}
	timeLine := "  " + dayViewLabelStyle.Render("Start:") + " " + startDisplay +
		"   " + dayViewLabelStyle.Render("End:") + " " + endDisplay
	if dur, ok := m.dayRecord.DayDuration(); ok {
		timeLine += "   " + dayViewMutedStyle.Render("("+journal.FormatDuration(dur)+")")
	}
	b.WriteString(timeLine + "\n\n")

	// ── Work Log section ──────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("📋  Work Log") + "\n")
	b.WriteString(div + "\n")

	entries := m.dayRecord.Entries
	if len(entries) == 0 {
		b.WriteString(dayViewMutedStyle.Render("  No entries yet") + "\n")
	} else {
		// column widths: selector(2) + project(14) + task(dynamic) + duration(8)
		taskW := innerW - 2 - 14 - 8
		if taskW < 10 {
			taskW = 10
		}
		for i, e := range entries {
			selector := "  "
			if i == m.selectedEntry {
				selector = "▶ "
			}

			proj := fmt.Sprintf("%-14s", e.Project)
			taskStr := e.Task
			if e.IsBreak {
				taskStr = "☕  " + taskStr
			}
			if len(taskStr) > taskW {
				taskStr = taskStr[:taskW-1] + "…"
			}
			taskStr = fmt.Sprintf("%-*s", taskW, taskStr)
			durStr := fmt.Sprintf("%8s", journal.FormatDuration(e.Duration()))

			line := selector + proj + taskStr + durStr

			if i == m.selectedEntry {
				line = selectedEntryStyle.Render(line)
			} else if e.IsBreak {
				line = breakEntryStyle.Render(line)
			} else {
				line = normalEntryStyle.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	work, breaks, _ := m.dayRecord.WorkTotals()
	totals := ""
	if work > 0 || breaks > 0 {
		totals = "  Work: " + journal.FormatDuration(work)
		if breaks > 0 {
			totals += "  ·  Breaks: " + journal.FormatDuration(breaks)
			totals += "  ·  Total: " + journal.FormatDuration(work+breaks)
		}
	}
	if totals != "" {
		b.WriteString("\n" + dayViewTotalsStyle.Render(totals) + "\n")
	}
	if dayDur, ok := m.dayRecord.DayDuration(); ok {
		logged := work + breaks
		if logged != dayDur {
			diff := dayDur - logged
			sign := "+"
			if diff < 0 {
				diff = -diff
				sign = "-"
			}
			warn := "  ⚠  Logged time (" + journal.FormatDuration(logged) + ") differs from day span (" +
				journal.FormatDuration(dayDur) + ") by " + sign + journal.FormatDuration(diff)
			b.WriteString(dayViewWarnStyle.Render(warn) + "\n")
		}
	}

	b.WriteString("\n")

	// ── Notes section ─────────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("📝  Notes") + "\n")
	b.WriteString(div + "\n")
	if m.dayRecord.Notes == "" {
		b.WriteString(dayViewMutedStyle.Render("  No notes") + "\n")
	} else {
		for _, line := range strings.Split(m.dayRecord.Notes, "\n") {
			b.WriteString(dayViewNotesStyle.Render("  "+line) + "\n")
		}
	}

	return b.String()
}

func (m Model) renderSummaryContent() string {
	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	// ── Work Day times ────────────────────────────────────────────────────────
	b.WriteString(dayViewSectionStyle.Render("🕐  Work Day") + "\n")
	b.WriteString(div + "\n")
	startD, endD := m.dayRecord.StartTime, m.dayRecord.EndTime
	if startD == "" {
		startD = dayViewMutedStyle.Render("—")
	} else {
		startD = dayViewValueStyle.Render(startD)
	}
	if endD == "" {
		endD = dayViewMutedStyle.Render("—")
	} else {
		endD = dayViewValueStyle.Render(endD)
	}
	timeLine := "  " + dayViewLabelStyle.Render("Start:") + " " + startD +
		"   " + dayViewLabelStyle.Render("End:") + " " + endD
	if dur, ok := m.dayRecord.DayDuration(); ok {
		timeLine += "   " + dayViewMutedStyle.Render("("+journal.FormatDuration(dur)+")")
	}
	b.WriteString(timeLine + "\n\n")

	// ── Projects ──────────────────────────────────────────────────────────────
	type projGroup struct {
		name    string
		entries []journal.WorkEntry
	}
	seen := make(map[string]int)
	var groups []projGroup
	var breakEntries []journal.WorkEntry

	for _, e := range m.dayRecord.Entries {
		if e.IsBreak {
			breakEntries = append(breakEntries, e)
			continue
		}
		proj := e.Project
		if proj == "" {
			proj = "—"
		}
		if idx, ok := seen[proj]; ok {
			groups[idx].entries = append(groups[idx].entries, e)
		} else {
			seen[proj] = len(groups)
			groups = append(groups, projGroup{name: proj, entries: []journal.WorkEntry{e}})
		}
	}

	// consolidateByName merges entries with the same task name (case-insensitive).
	consolidateByName := func(entries []journal.WorkEntry) []journal.WorkEntry {
		seen := make(map[string]int)
		var out []journal.WorkEntry
		for _, e := range entries {
			key := strings.ToLower(e.Task)
			if idx, ok := seen[key]; ok {
				out[idx].DurationMin += e.DurationMin
			} else {
				seen[key] = len(out)
				out = append(out, e)
			}
		}
		return out
	}

	if len(groups) == 0 && len(breakEntries) == 0 {
		b.WriteString("  " + dayViewMutedStyle.Render("No entries yet") + "\n")
	} else {
		b.WriteString(dayViewSectionStyle.Render("🗂  By Project") + "\n")
		b.WriteString(div + "\n")

		for _, g := range groups {
			tasks := consolidateByName(g.entries)
			var projTotal time.Duration
			var names []string
			for _, t := range tasks {
				projTotal += t.Duration()
				names = append(names, t.Task)
			}

			projLabel := g.name
			if projLabel == "—" {
				projLabel = "Other"
			}

			durStr := fmt.Sprintf("%-8s", journal.FormatDuration(projTotal))
			taskList := "\"" + strings.Join(names, ", ") + "\""
			b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
				"  " + dayViewSectionStyle.Render(projLabel) +
				"  " + dayViewMutedStyle.Render(taskList) + "\n")
		}

		// ── Breaks block ──────────────────────────────────────────────────────
		if len(breakEntries) > 0 {
			bkList := consolidateByName(breakEntries)
			var breakTotal time.Duration
			var names []string
			for _, e := range bkList {
				breakTotal += e.Duration()
				names = append(names, e.Task)
			}
			durStr := fmt.Sprintf("%-8s", journal.FormatDuration(breakTotal))
			taskList := "\"" + strings.Join(names, ", ") + "\""
			b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
				"  " + breakEntryStyle.Render("☕  Breaks") +
				"  " + dayViewMutedStyle.Render(taskList) + "\n")
		}

		// ── Totals ────────────────────────────────────────────────────────────
		work, breaks, total := m.dayRecord.WorkTotals()
		b.WriteString(div + "\n")
		if breaks > 0 {
			b.WriteString(dayViewTotalsStyle.Render(fmt.Sprintf("  Work: %s  ·  Breaks: %s  ·  Total: %s",
				journal.FormatDuration(work), journal.FormatDuration(breaks), journal.FormatDuration(total))) + "\n")
		} else if work > 0 {
			b.WriteString(dayViewTotalsStyle.Render("  Total work: "+journal.FormatDuration(work)) + "\n")
		}
	}

	return b.String()
}

func (m Model) View() string {
	if !m.ready {
		return "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve)).Render("Loading…")
	}
	switch m.state {
	case stateList:
		return m.viewList()
	case stateDayView:
		return m.viewDayView()
	case stateWorkForm:
		return m.viewWorkLogForm()
	case stateTimeInput:
		return m.viewTimeInput()
	case stateNotesEditor:
		return m.viewNotesEditor()
	case stateConfirmDelete:
		return m.viewConfirmDelete()
	case stateDateInput:
		return m.viewDateInput()
	case stateWeekView:
		return m.viewWeekView()
	}
	return ""
}

func (m Model) renderHeader(title, subtitle string) string {
	left := headerTitleStyle.Render(title)
	right := headerSubtitleStyle.Render(subtitle)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	top := left + strings.Repeat(" ", gap) + right
	sep := separatorStyle.Render(strings.Repeat("─", m.width))
	return top + "\n" + sep
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

	if m.statusMsg != "" {
		s := m.statusMsg
		if m.isError {
			s = statusErrorStyle.Render(s)
		} else {
			s = statusSuccessStyle.Render(s)
		}
		pad := m.width - lipgloss.Width(help) - lipgloss.Width(s) - 2
		if pad < 1 {
			pad = 1
		}
		return help + strings.Repeat(" ", pad) + s
	}
	return help
}

// ── Date input (open/create any day) ─────────────────────────────────────────

func (m Model) openDateInput() (tea.Model, tea.Cmd) {
	m.dateInput.SetValue("")
	m.dateInput.Placeholder = time.Now().Format("2006-01-02")
	cmd := m.dateInput.Focus()
	m.state = stateDateInput
	return m, cmd
}

func (m Model) handleDateInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.dateInput.Blur()
		m.state = stateList
		return m, nil
	case "enter":
		raw := strings.TrimSpace(m.dateInput.Value())
		if raw == "" {
			raw = time.Now().Format("2006-01-02")
		}
		// Accept YYYY-MM-DD
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			m.statusMsg = "✗ Invalid date — use YYYY-MM-DD"
			m.isError = true
			m.state = stateList
			return m, clearStatusCmd()
		}
		m.dateInput.Blur()
		dateStr := date.Format("2006-01-02")
		path, err := journal.PathForDate(dateStr)
		if err != nil {
			m.statusMsg = "✗ " + err.Error()
			m.isError = true
			m.state = stateList
			return m, clearStatusCmd()
		}
		rec, _ := journal.Load(path)
		if rec.Date == "" {
			rec.Date = dateStr
		}
		rec.Path = path
		return m.openDayView(rec)
	}
	var cmd tea.Cmd
	m.dateInput, cmd = m.dateInput.Update(msg)
	return m, cmd
}

func (m Model) viewDateInput() string {
	header := m.renderHeader("📔  Schmournal", "Open Day")
	prompt := dayViewLabelStyle.Render("Enter date:") + "  " + m.dateInput.View()
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

func (m Model) viewList() string {
	header := m.renderHeader("📔  Schmournal", time.Now().Format("Mon, 02 Jan 2006"))
	kb := m.cfg.Keybinds.List
	footer := m.renderFooter([][2]string{
		{kb.OpenToday, "open today"},
		{kb.OpenDate, "open date"},
		{kb.AddWork, "log work"},
		{kb.AddBreak, "log break"},
		{"enter", "view"},
		{kb.WeekView, "week"},
		{kb.Delete, "delete"},
		{kb.Export, "export"},
		{"/", "filter"},
		{"esc", "quit"},
	})
	sections := []string{header, m.renderStats()}
	if eom := m.renderEOMBanner(); eom != "" {
		sections = append(sections, eom)
	}
	sections = append(sections, m.list.View(), footer)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderEOMBanner() string {
	now := time.Now()
	// last day of the month: first day of next month minus one day
	firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if now.Before(firstOfNext.AddDate(0, 0, -1)) {
		return ""
	}
	return eomBannerStyle.Width(m.width).Render("⚠  Last day of the month — don't forget to submit your times!")
}

func (m Model) renderStats() string {
	now := time.Now()

	// Build a set of dates that have records.
	dated := make(map[string]bool, len(m.records))
	for _, r := range m.records {
		dated[r.Date] = true
	}

	// Week bar: ISO week starting Monday.
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	dayInitials := [7]string{"M", "T", "W", "T", "F", "S", "S"}
	const filled = "▓"
	const empty = "░"
	var weekParts []string
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		lbl := statsLabelStyle.Render(dayInitials[i])
		var block string
		switch {
		case dated[dateStr]:
			block = statsBlockFilledStyle.Render(filled)
		case d.After(now):
			block = statsBlockFutureStyle.Render(empty)
		default:
			block = statsBlockEmptyStyle.Render(empty)
		}
		weekParts = append(weekParts, lbl+" "+block)
	}
	weekStr := strings.Join(weekParts, "  ")

	// Month count.
	monthKey := now.Format("2006-01")
	monthCount := 0
	for _, r := range m.records {
		if strings.HasPrefix(r.Date, monthKey) {
			monthCount++
		}
	}
	monthStr := statsLabelStyle.Render(now.Format("Jan")+": ") +
		statsValueStyle.Render(fmt.Sprintf("%d", monthCount))

	// Streak: consecutive days going back from today.
	streak := 0
	for check := now; ; check = check.AddDate(0, 0, -1) {
		if !dated[check.Format("2006-01-02")] {
			break
		}
		streak++
	}
	var streakStr string
	if streak > 0 {
		streakStr = "🔥 " + statsStreakStyle.Render(fmt.Sprintf("%d", streak)) +
			statsLabelStyle.Render(" day streak")
	} else {
		streakStr = statsLabelStyle.Render("No active streak")
	}

	dot := helpStyle.Render("  ·  ")
	line := "  " + weekStr + dot + monthStr + dot + streakStr
	sep := separatorStyle.Render(strings.Repeat("─", m.width))

	// Weekly work hours progress bar.
	var weekWork time.Duration
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		for _, r := range m.records {
			if r.Date == dateStr {
				work, _, _ := r.WorkTotals()
				weekWork += work
				break
			}
		}
	}
	const weeklyGoal = 40 * time.Hour
	pct := float64(weekWork) / float64(weeklyGoal)
	if pct > 1 {
		pct = 1
	}
	const progressBarWidth = 20
	filledCount := int(pct * progressBarWidth)
	progressBar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
		statsBlockEmptyStyle.Render(strings.Repeat("░", progressBarWidth-filledCount))
	weekHoursStr := "  " + statsLabelStyle.Render("Week: ") +
		statsValueStyle.Render(journal.FormatDuration(weekWork)) +
		statsLabelStyle.Render(" / 40h  ") +
		progressBar +
		statsLabelStyle.Render(fmt.Sprintf("  %.0f%%", pct*100))

	return line + "\n" + weekHoursStr + "\n" + sep
}

func (m Model) viewDayView() string {
	subtitle := m.dayRecord.Date
	if t, err := m.dayRecord.ParseDate(); err == nil {
		subtitle = t.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader("📔  Schmournal", subtitle)
	tabBar := m.renderTabBar()

	var footerKeys [][2]string
	kb := m.cfg.Keybinds.Day
	if m.dayViewTab == 0 {
		footerKeys = [][2]string{
			{"←/→", "switch tab"},
			{"j/k", "select"},
			{kb.AddWork, "work"},
			{kb.AddBreak, "break"},
			{kb.Edit, "edit"},
			{kb.Delete, "del"},
			{joinKeyLabels(kb.SetStartNow, kb.SetStartManual), "start"},
			{joinKeyLabels(kb.SetEndNow, kb.SetEndManual), "end"},
			{kb.Notes, "notes"},
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
	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, m.viewport.View(), footer)
}

func (m Model) viewWorkLogForm() string {
	var badge, taskLabel string
	dateStr := m.dayRecord.Date
	if m.isBreakEntry {
		badge = breakLogBadgeStyle.Render(" Log Break ")
		taskLabel = "Break label"
	} else {
		badge = workLogBadgeStyle.Render(" Log Work ")
		taskLabel = "What did you work on?"
	}
	header := m.renderHeader("📔  Schmournal", badge+
		headerSubtitleStyle.Render("  "+dateStr))
	footer := m.renderFooter([][2]string{
		{"tab", "next field"},
		{"enter", "save"},
		{"esc", "cancel"},
	})

	formWidth := m.width - 8
	if formWidth < 40 {
		formWidth = 40
	}
	inputWidth := formWidth - 8

	m.taskInput.Width = inputWidth
	m.projectInput.Width = inputWidth
	m.durationInput.Width = inputWidth

	renderBox := func(input textinput.Model, active bool) string {
		if active {
			return formActiveInputStyle.Width(inputWidth).Render(input.View())
		}
		return formInactiveInputStyle.Width(inputWidth).Render(input.View())
	}

	taskBox := renderBox(m.taskInput, m.activeInput == 0)
	durBox := renderBox(m.durationInput, m.activeInput == m.numFormFields()-1)

	var body string
	if m.isBreakEntry {
		body = formLabelStyle.Render(taskLabel) + "\n" +
			taskBox + "\n\n" +
			formLabelStyle.Render("Duration") +
			formHintStyle.Render("  e.g. 1h 30m · 45m · 2h") + "\n" +
			durBox
	} else {
		projBox := renderBox(m.projectInput, m.activeInput == 1)
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

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(form)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

func (m Model) viewTimeInput() string {
	label := "Set Start Time"
	badge := workLogBadgeStyle.Render(" Start ")
	if !m.timeInputStart {
		label = "Set Finish Time"
		badge = breakLogBadgeStyle.Render(" Finish ")
	}
	header := m.renderHeader("📔  Schmournal", badge)
	footer := m.renderFooter([][2]string{
		{"enter", "confirm"},
		{"esc", "cancel"},
	})

	m.timeInput.Width = 12
	inputBox := formActiveInputStyle.Width(14).Render(m.timeInput.View())

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

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(dialog)
	return header + "\n" + strings.Repeat("\n", topPad) + centered + "\n" + footer
}

func (m Model) viewNotesEditor() string {
	subtitle := m.dayRecord.Date
	if t, err := m.dayRecord.ParseDate(); err == nil {
		subtitle = t.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader("📔  Schmournal", subtitle)
	footer := m.renderFooter([][2]string{
		{"ctrl+s", "save"},
		{"esc", "cancel"},
	})
	editor := editorBorderStyle.
		Width(m.width - 4).
		Render(m.textarea.View())
	return lipgloss.JoinVertical(lipgloss.Left, header, editor, footer)
}

func (m Model) viewConfirmDelete() string {
	var subject string
	if m.deleteDay {
		if m.prevState == stateDayView {
			subject = m.dayRecord.Date
		} else if m.deleteIdx >= 0 && m.deleteIdx < len(m.records) {
			subject = m.records[m.deleteIdx].Date
		}
		subject = "the day " + subject
	} else {
		if m.deleteIdx >= 0 && m.deleteIdx < len(m.dayRecord.Entries) {
			subject = `entry "` + m.dayRecord.Entries[m.deleteIdx].Task + `"`
		} else {
			subject = "this entry"
		}
	}
	header := m.renderHeader("📔  Schmournal", "Delete")

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

	centered := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(dialog)
	return header + "\n" + strings.Repeat("\n", topPad) + centered
}

// ── Week view ─────────────────────────────────────────────────────────────────

func (m Model) viewWeekView() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)
	sunday := monday.AddDate(0, 0, 6)

	weekRange := monday.Format("02 Jan") + " – " + sunday.Format("02 Jan 2006")
	header := m.renderHeader("📅  This Week", weekRange)

	// Navigation hint bar (matches day-view tab bar height: 2 lines).
	var navParts []string
	navParts = append(navParts, inactiveTabStyle.Render("← prev week"))
	if m.weekOffset < 0 {
		navParts = append(navParts, inactiveTabStyle.Render("→ next week"))
	}
	navBar := strings.Join(navParts, inactiveTabStyle.Render("  "))
	sep := dayViewDividerStyle.Render(strings.Repeat("─", m.width))
	subHeader := navBar + "\n" + sep

	footer := m.renderFooter([][2]string{
		{"j/k", "scroll"},
		{"←/→", "prev/next week"},
		{"esc", "back"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, subHeader, m.viewport.View(), footer)
}

func (m Model) renderWeekContent() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+m.weekOffset*7)

	// Build date→record map for quick lookup.
	recByDate := make(map[string]journal.DayRecord, len(m.records))
	for _, r := range m.records {
		recByDate[r.Date] = r
	}

	innerW := m.width - 2
	if innerW < 40 {
		innerW = 40
	}
	div := dayViewDividerStyle.Render(strings.Repeat("─", innerW))

	var b strings.Builder
	b.WriteString("\n")

	var weekWork time.Duration

	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		dateStr := d.Format("2006-01-02")
		rec, hasRec := recByDate[dateStr]

		var work, breaks time.Duration
		if hasRec {
			work, breaks, _ = rec.WorkTotals()
		}
		weekWork += work

		// Day header line.
		dayLabel := d.Format("Mon  02 Jan 2006")
		var headerLine string
		if hasRec && (work+breaks) > 0 {
			headerLine = dayViewSectionStyle.Render(dayLabel) +
				dayViewMutedStyle.Render("  ·  ") +
				dayViewValueStyle.Render(journal.FormatDuration(work)+" work")
			if breaks > 0 {
				headerLine += dayViewMutedStyle.Render("  ·  " + journal.FormatDuration(breaks) + " breaks")
			}
		} else {
			headerLine = dayViewSectionStyle.Render(dayLabel) +
				dayViewMutedStyle.Render("  ·  no entries")
		}
		b.WriteString(headerLine + "\n")

		// Start / end time sub-line (only shown when at least one is set).
		if hasRec && (rec.StartTime != "" || rec.EndTime != "") {
			start := rec.StartTime
			end := rec.EndTime
			startStr := dayViewValueStyle.Render(start)
			if start == "" {
				startStr = dayViewMutedStyle.Render("—")
			}
			endStr := dayViewValueStyle.Render(end)
			if end == "" {
				endStr = dayViewMutedStyle.Render("—")
			}
			timeLine := "  " + dayViewLabelStyle.Render("Start:") + " " + startStr +
				"   " + dayViewLabelStyle.Render("End:") + " " + endStr
			if dur, ok := rec.DayDuration(); ok {
				timeLine += "   " + dayViewMutedStyle.Render("("+journal.FormatDuration(dur)+")")
			}
			b.WriteString(timeLine + "\n")
		}
		if hasRec && len(rec.Entries) > 0 {
			type projGroup struct {
				name   string
				dur    time.Duration
				tasks  []string
				isBreak bool
			}
			seenProj := make(map[string]int)
			var groups []projGroup

			for _, e := range rec.Entries {
				if e.IsBreak {
					// Collect breaks under a single "Breaks" group.
					found := false
					for gi, g := range groups {
						if g.isBreak {
							groups[gi].dur += e.Duration()
							groups[gi].tasks = uniqueAppend(groups[gi].tasks, e.Task)
							found = true
							break
						}
					}
					if !found {
						groups = append(groups, projGroup{
							name:    "Breaks",
							dur:     e.Duration(),
							tasks:   []string{e.Task},
							isBreak: true,
						})
					}
					continue
				}
				proj := e.Project
				if proj == "" {
					proj = "Other"
				}
				if idx, ok := seenProj[proj]; ok {
					groups[idx].dur += e.Duration()
					groups[idx].tasks = uniqueAppend(groups[idx].tasks, e.Task)
				} else {
					seenProj[proj] = len(groups)
					groups = append(groups, projGroup{
						name:  proj,
						dur:   e.Duration(),
						tasks: []string{e.Task},
					})
				}
			}

			for _, g := range groups {
				durStr := fmt.Sprintf("%-8s", journal.FormatDuration(g.dur))
				taskStr := `"` + strings.Join(g.tasks, ", ") + `"`
				if g.isBreak {
					b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
						"  " + breakEntryStyle.Render("☕  "+g.name) +
						"  " + dayViewMutedStyle.Render(taskStr) + "\n")
				} else {
					b.WriteString("  " + dayViewTotalsStyle.Render(durStr) +
						"  " + dayViewSectionStyle.Render(g.name) +
						"  " + dayViewMutedStyle.Render(taskStr) + "\n")
				}
			}
		}

		if i < 6 {
			b.WriteString("\n")
		}
	}

	// Week total + progress bar.
	b.WriteString("\n" + div + "\n")
	const weeklyGoal = 40 * time.Hour
	pct := float64(weekWork) / float64(weeklyGoal)
	if pct > 1 {
		pct = 1
	}
	const barW = 24
	filledCount := int(pct * barW)
	bar := statsBlockFilledStyle.Render(strings.Repeat("█", filledCount)) +
		statsBlockEmptyStyle.Render(strings.Repeat("░", barW-filledCount))
	totalLine := "  " + dayViewLabelStyle.Render("Week total: ") +
		dayViewValueStyle.Render(journal.FormatDuration(weekWork)) +
		dayViewMutedStyle.Render(" / 40h  ") +
		bar +
		dayViewMutedStyle.Render(fmt.Sprintf("  %.0f%%", pct*100))
	b.WriteString(totalLine + "\n")

	return b.String()
}

// uniqueAppend appends s to slice only if not already present (case-sensitive).
func uniqueAppend(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
