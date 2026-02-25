package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/fgrohme/tui-journal/journal"
)

// ── View states ───────────────────────────────────────────────────────────────

type viewState int

const (
	stateList viewState = iota
	stateViewer
	stateEditor
	stateConfirmDelete
	stateWorkLogForm
	stateTimeInput
)

const (
	headerHeight = 2 // title line + separator
	footerHeight = 1
)

// ── Messages ──────────────────────────────────────────────────────────────────

type entriesLoadedMsg struct {
	entries []journal.Entry
	err     error
}
type savedMsg struct{}
type deletedMsg struct{}
type stampedMsg struct{ text string }
type workEntrySavedMsg struct{ isBreak bool }
type exportedMsg struct{ path string }
type clearStatusMsg struct{}
type errMsg struct{ err error }

// ── List item ─────────────────────────────────────────────────────────────────

type entryItem struct{ entry journal.Entry }

func (i entryItem) FilterValue() string {
	return i.entry.Title + " " + i.entry.Content
}

func (i entryItem) Title() string {
	now := time.Now()
	isToday := i.entry.Date.Year() == now.Year() && i.entry.Date.YearDay() == now.YearDay()
	if isToday {
		return "✦ " + i.entry.Title
	}
	return i.entry.Title
}

func (i entryItem) Description() string {
	date := i.entry.Date.Format("Mon, 02 Jan 2006")
	for _, line := range strings.Split(i.entry.Content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if len(line) > 60 {
				line = line[:60] + "…"
			}
			return date + "  ·  " + line
		}
	}
	return date
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	state     viewState
	prevState viewState
	width     int
	height    int
	ready     bool

	// List view
	list    list.Model
	entries []journal.Entry

	// Viewer
	viewport    viewport.Model
	viewing     int
	viewingPath string // stable reference across reloads

	// Editor
	textarea textarea.Model
	editPath string
	editDate time.Time

	// Delete confirm
	deleteIdx int

	// Work log form
	taskInput          textinput.Model
	projectInput       textinput.Model
	durationInput      textinput.Model
	activeInput        int    // field index (work: 0=task 1=project 2=duration; break: 0=label 1=duration)
	workLogPath        string // entry being targeted by the form
	workLogReturnState viewState
	isBreakEntry       bool // true when the form is logging a break

	// Manual time input dialog
	timeInput      textinput.Model
	timeInputStart bool // true = setting start, false = setting finish

	// Status bar
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

// New constructs the initial model.
func New() Model {
	l := list.New([]list.Item{}, newDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color(cBlue))
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))

	ta := textarea.New()
	ta.Placeholder = "Start writing…"
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color(cSurface0))
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

	return Model{
		state:         stateList,
		list:          l,
		textarea:      ta,
		taskInput:     taskIn,
		projectInput:  projIn,
		durationInput: durIn,
		timeInput:     timeIn,
	}
}

func (m Model) Init() tea.Cmd {
	return loadEntries
}

// ── Commands ──────────────────────────────────────────────────────────────────

func loadEntries() tea.Msg {
	entries, err := journal.LoadAll()
	return entriesLoadedMsg{entries: entries, err: err}
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		ch := m.contentHeight()
		if !m.ready {
			m.viewport = viewport.New(m.width, ch)
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = ch
		}
		m.list.SetSize(m.width, ch)
		m.textarea.SetWidth(m.width - 4)
		m.textarea.SetHeight(ch - 2) // border takes 2 lines
		return m, nil

	case entriesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "✗ " + msg.err.Error()
			m.isError = true
			return m, nil
		}
		m.entries = msg.entries
		items := make([]list.Item, len(m.entries))
		for i, e := range m.entries {
			items[i] = entryItem{entry: e}
		}
		m.list.SetItems(items)
		// Restore viewer position if we're still in the viewer.
		if m.state == stateViewer && m.viewingPath != "" {
			for i, e := range m.entries {
				if e.Path == m.viewingPath {
					m.viewing = i
					m.viewport.SetContent(m.renderMarkdown(e.Content))
					break
				}
			}
		}
		return m, nil

	case savedMsg:
		m.statusMsg = "✓ Entry saved"
		m.isError = false
		m.state = stateList
		return m, tea.Batch(loadEntries, clearStatusCmd())

	case stampedMsg:
		m.statusMsg = "✓ " + msg.text
		m.isError = false
		// State stays as stateViewer; entriesLoadedMsg will refresh the viewport.
		return m, tea.Batch(loadEntries, clearStatusCmd())

	case workEntrySavedMsg:
		if msg.isBreak {
			m.statusMsg = "✓ Break logged"
		} else {
			m.statusMsg = "✓ Work entry logged"
		}
		m.isError = false
		m.state = m.workLogReturnState
		return m, tea.Batch(loadEntries, clearStatusCmd())

	case exportedMsg:
		// Abbreviate home dir in the displayed path.
		display := msg.path
		if home, err := os.UserHomeDir(); err == nil {
			display = strings.Replace(display, home, "~", 1)
		}
		m.statusMsg = "✓ Exported → " + display
		m.isError = false
		return m, clearStatusCmd()

	case deletedMsg:
		m.statusMsg = "✓ Entry deleted"
		m.isError = false
		m.state = stateList
		return m, tea.Batch(loadEntries, clearStatusCmd())

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
		case stateViewer:
			return m.handleViewerKey(msg)
		case stateEditor:
			return m.handleEditorKey(msg)
		case stateConfirmDelete:
			return m.handleConfirmDeleteKey(msg)
		case stateWorkLogForm:
			return m.handleWorkLogFormKey(msg)
		case stateTimeInput:
			return m.handleTimeInputKey(msg)
		}
	}

	// Forward non-key messages to the active sub-model.
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateViewer:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case stateEditor:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	case stateWorkLogForm:
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
	}
	return m, nil
}

// ── Key handlers ──────────────────────────────────────────────────────────────

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtering := m.list.FilterState() == list.Filtering
	switch msg.String() {
	case "q":
		if !filtering {
			return m, tea.Quit
		}
	case "n":
		if !filtering {
			return m.openNewEntry()
		}
	case "enter":
		if !filtering {
			if item, ok := m.list.SelectedItem().(entryItem); ok {
				return m.openViewer(item.entry)
			}
		}
	case "e":
		if !filtering {
			if item, ok := m.list.SelectedItem().(entryItem); ok {
				return m.openEditor(item.entry)
			}
		}
	case "d":
		if !filtering {
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.entries) {
				m.deleteIdx = idx
				m.prevState = stateList
				m.state = stateConfirmDelete
				return m, nil
			}
		}
	case "w":
		// Log work entry to today's journal (creates the entry if needed).
		if !filtering {
			return m.openWorkLogFormForToday(false)
		}
	case "b":
		// Log a break to today's journal (creates the entry if needed).
		if !filtering {
			return m.openWorkLogFormForToday(true)
		}
	case "x":
		// Export selected entry.
		if !filtering {
			if item, ok := m.list.SelectedItem().(entryItem); ok {
				entry := item.entry
				return m, func() tea.Msg {
					path, err := journal.SaveExport(entry)
					if err != nil {
						return errMsg{err: err}
					}
					return exportedMsg{path: path}
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleViewerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = stateList
		return m, nil
	case "e":
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.openEditor(m.entries[m.viewing])
		}
	case "d":
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			m.deleteIdx = m.viewing
			m.prevState = stateViewer
			m.state = stateConfirmDelete
			return m, nil
		}
	case "w":
		// Log work entry to the currently viewed entry.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.openWorkLogForm(m.entries[m.viewing].Path, stateViewer, false)
		}
	case "b":
		// Log a break to the currently viewed entry.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.openWorkLogForm(m.entries[m.viewing].Path, stateViewer, true)
		}
	case "x":
		// Export current entry.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			entry := m.entries[m.viewing]
			return m, func() tea.Msg {
				path, err := journal.SaveExport(entry)
				if err != nil {
					return errMsg{err: err}
				}
				return exportedMsg{path: path}
			}
		}
	case "s":
		// Stamp work-day start time.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.stampTime(m.entries[m.viewing], true)
		}
	case "S":
		// Open dialog to manually set start time.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.openTimeInput(true)
		}
	case "f":
		// Stamp work-day finish time.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.stampTime(m.entries[m.viewing], false)
		}
	case "F":
		// Open dialog to manually set finish time.
		if m.viewing >= 0 && m.viewing < len(m.entries) {
			return m.openTimeInput(false)
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		content := m.textarea.Value()
		path := m.editPath
		return m, func() tea.Msg {
			if err := journal.Save(path, content); err != nil {
				return errMsg{err: err}
			}
			return savedMsg{}
		}
	case tea.KeyEsc:
		m.state = stateList
		return m, nil
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteIdx >= 0 && m.deleteIdx < len(m.entries) {
			path := m.entries[m.deleteIdx].Path
			return m, func() tea.Msg {
				if err := journal.Delete(path); err != nil {
					return errMsg{err: err}
				}
				return deletedMsg{}
			}
		}
	case "n", "N", "esc":
		m.state = m.prevState
	}
	return m, nil
}

func (m Model) handleWorkLogFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		project := strings.TrimSpace(m.projectInput.Value())
		targetPath := m.workLogPath
		retState := m.workLogReturnState
		isBreak := m.isBreakEntry
		m.state = retState
		return m, func() tea.Msg {
			content := loadOrCreateContent(targetPath)
			var updated string
			if isBreak {
				updated = journal.AddBreakEntry(content, task, dur)
			} else {
				updated = journal.AddWorkEntry(content, task, project, dur)
			}
			if err := journal.Save(targetPath, updated); err != nil {
				return errMsg{err: err}
			}
			return workEntrySavedMsg{isBreak: isBreak}
		}

	case tea.KeyEsc:
		m.state = m.workLogReturnState
		return m, nil
	}

	// Forward to the active sub-input.
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

// ── Navigation helpers ────────────────────────────────────────────────────────

func (m Model) openViewer(entry journal.Entry) (tea.Model, tea.Cmd) {
	for i, e := range m.entries {
		if e.Path == entry.Path {
			m.viewing = i
			break
		}
	}
	m.viewingPath = entry.Path
	m.viewport.SetContent(m.renderMarkdown(entry.Content))
	m.viewport.GotoTop()
	m.state = stateViewer
	return m, nil
}

func (m Model) openEditor(entry journal.Entry) (tea.Model, tea.Cmd) {
	m.editPath = entry.Path
	m.editDate = entry.Date
	m.textarea.SetValue(entry.Content)
	blinkCmd := m.textarea.Focus()
	m.state = stateEditor
	return m, blinkCmd
}

func (m Model) openNewEntry() (tea.Model, tea.Cmd) {
	path, err := journal.TodayPath()
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	now := time.Now()
	content := journal.NewEntryContent(now)
	if existing, err := journal.Load(path); err == nil {
		content = existing.Content
	}
	m.editPath = path
	m.editDate = now
	m.textarea.SetValue(content)
	blinkCmd := m.textarea.Focus()
	m.state = stateEditor
	return m, blinkCmd
}

// numFormFields returns how many fields the current work-log form has.
// Work entries: 3 (task, project, duration). Breaks: 2 (label, duration).
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

// openWorkLogForm opens the work/break log entry form targeting the entry at path.
func (m Model) openWorkLogForm(path string, returnTo viewState, isBreak bool) (tea.Model, tea.Cmd) {
	m.workLogPath = path
	m.workLogReturnState = returnTo
	m.isBreakEntry = isBreak
	m.taskInput.SetValue("")
	m.projectInput.SetValue("")
	m.durationInput.SetValue("")
	if isBreak {
		m.taskInput.Placeholder = "e.g. Lunch, coffee break, walk…"
	} else {
		m.taskInput.Placeholder = "e.g. Feature development, meeting, code review…"
	}
	m.state = stateWorkLogForm
	return m.focusField(0)
}

// openWorkLogFormForToday opens the work/break form pointing at today's entry.
func (m Model) openWorkLogFormForToday(isBreak bool) (tea.Model, tea.Cmd) {
	path, err := journal.TodayPath()
	if err != nil {
		m.statusMsg = "✗ " + err.Error()
		m.isError = true
		return m, nil
	}
	return m.openWorkLogForm(path, stateList, isBreak)
}

// stampTime writes the current time as the start or finish marker in an entry.
func (m Model) stampTime(entry journal.Entry, isStart bool) (tea.Model, tea.Cmd) {
	timeStr := time.Now().Format("15:04")
	path := entry.Path
	content := entry.Content
	var (
		updated string
		found   bool
		label   string
	)
	if isStart {
		updated, found = journal.StampStartTime(content, timeStr)
		label = "Start time set to " + timeStr
	} else {
		updated, found = journal.StampEndTime(content, timeStr)
		label = "Finish time set to " + timeStr
	}
	if !found {
		m.statusMsg = "✗ No time marker found — entry needs the daily template"
		m.isError = true
		return m, clearStatusCmd()
	}
	return m, func() tea.Msg {
		if err := journal.Save(path, updated); err != nil {
			return errMsg{err: err}
		}
		return stampedMsg{text: label}
	}
}

// openTimeInput opens the manual time-entry dialog.
func (m Model) openTimeInput(isStart bool) (tea.Model, tea.Cmd) {
	m.timeInputStart = isStart
	// Pre-fill with the current time as a sensible default.
	m.timeInput.SetValue(time.Now().Format("15:04"))
	m.timeInput.CursorEnd()
	m.state = stateTimeInput
	return m, m.timeInput.Focus()
}

func (m Model) handleTimeInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.timeInput.Value())
		if !isValidHHMM(val) {
			m.statusMsg = "✗ Invalid time — use HH:MM (e.g. 09:30)"
			m.isError = true
			m.state = stateViewer
			return m, clearStatusCmd()
		}
		if m.viewing < 0 || m.viewing >= len(m.entries) {
			m.state = stateViewer
			return m, nil
		}
		entry := m.entries[m.viewing]
		isStart := m.timeInputStart
		m.state = stateViewer
		return m, func() tea.Msg {
			var (
				updated string
				found   bool
				label   string
			)
			if isStart {
				updated, found = journal.StampStartTime(entry.Content, val)
				label = "Start time set to " + val
			} else {
				updated, found = journal.StampEndTime(entry.Content, val)
				label = "Finish time set to " + val
			}
			if !found {
				return errMsg{err: fmt.Errorf("no time marker found — entry needs the daily template")}
			}
			if err := journal.Save(entry.Path, updated); err != nil {
				return errMsg{err: err}
			}
			return stampedMsg{text: label}
		}

	case tea.KeyEsc:
		m.state = stateViewer
		return m, nil
	}

	var cmd tea.Cmd
	m.timeInput, cmd = m.timeInput.Update(msg)
	return m, cmd
}

// isValidHHMM reports whether s is a valid 24-hour time in HH:MM format.
func isValidHHMM(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}
func loadOrCreateContent(path string) string {
	if e, err := journal.Load(path); err == nil {
		return e.Content
	}
	base := filepath.Base(path)
	dateStr := strings.TrimSuffix(base, ".md")
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t = time.Now()
	}
	return journal.NewEntryContent(t)
}

func (m Model) renderMarkdown(content string) string {
	w := m.width - 4
	if w < 40 {
		w = 40
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	return rendered
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if !m.ready {
		return "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve)).Render("Loading…")
	}
	switch m.state {
	case stateList:
		return m.viewList()
	case stateViewer:
		return m.viewViewer()
	case stateEditor:
		return m.viewEditor()
	case stateConfirmDelete:
		return m.viewConfirmDelete()
	case stateWorkLogForm:
		return m.viewWorkLogForm()
	case stateTimeInput:
		return m.viewTimeInput()
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

func (m Model) viewList() string {
	header := m.renderHeader("📔  Schmournal", time.Now().Format("Mon, 02 Jan 2006"))
	footer := m.renderFooter([][2]string{
		{"n", "new"},
		{"w", "log work"},
		{"b", "log break"},
		{"enter", "view"},
		{"e", "edit"},
		{"d", "delete"},
		{"x", "export"},
		{"/", "filter"},
		{"q", "quit"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, m.list.View(), footer)
}

func (m Model) viewViewer() string {
	subtitle := ""
	if m.viewing >= 0 && m.viewing < len(m.entries) {
		subtitle = m.entries[m.viewing].Date.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader("📔  Schmournal", subtitle)
	footer := m.renderFooter([][2]string{
		{"w", "log work"},
		{"b", "log break"},
		{"s", "start now"},
		{"S", "set start"},
		{"f", "finish now"},
		{"F", "set finish"},
		{"x", "export"},
		{"e", "edit"},
		{"d", "delete"},
		{"↑↓", "scroll"},
		{"esc", "back"},
	})
	return lipgloss.JoinVertical(lipgloss.Left, header, m.viewport.View(), footer)
}

func (m Model) viewEditor() string {
	subtitle := "New Entry"
	if !m.editDate.IsZero() {
		subtitle = "Editing · " + m.editDate.Format("Mon, 02 Jan 2006")
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
	entryDate := ""
	if m.deleteIdx >= 0 && m.deleteIdx < len(m.entries) {
		entryDate = m.entries[m.deleteIdx].Date.Format("Monday, 02 January 2006")
	}
	header := m.renderHeader("📔  Schmournal", "Delete Entry")

	dialog := confirmBoxStyle.Render(
		confirmTitleStyle.Render(fmt.Sprintf("Delete the entry for %s?", entryDate)) +
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

func (m Model) viewWorkLogForm() string {
	var badge, taskLabel string
	if m.isBreakEntry {
		badge = breakLogBadgeStyle.Render(" Log Break ")
		taskLabel = "Break label"
	} else {
		badge = workLogBadgeStyle.Render(" Log Work ")
		taskLabel = "What did you work on?"
	}
	header := m.renderHeader("📔  Schmournal", badge+
		headerSubtitleStyle.Render("  "+filepath.Base(m.workLogPath)))
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
