package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sleepypxnda/schmournal/config"
	"github.com/sleepypxnda/schmournal/journal"
)

// ── View states ───────────────────────────────────────────────────────────────

type viewState int

const (
	stateList viewState = iota
	stateDayView
	stateWorkForm
	stateClockForm
	stateTimeInput
	stateNotesEditor
	stateConfirmDelete
	stateDateInput
	stateWeekView
	stateWeekHoursInput
	stateWorkspacePicker
	stateStats
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
type weekGoalsLoadedMsg struct{ goals journal.WeeklyGoals }
type clockTickMsg struct{}

// ── List item ─────────────────────────────────────────────────────────────────

type dayListItem struct {
	rec       journal.DayRecord
	isWorkDay bool
}

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

	cfg             config.Config
	activeWorkspace string // name of the currently active workspace (empty = no workspaces)

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

	weekOffset     int // 0 = current week, -1 = last week, etc.
	weekGoals      journal.WeeklyGoals
	weekHoursInput textinput.Model

	statsTab int // 0=Overview 1=Monthly 2=Yearly 3=All-time

	workspaceIdx int // currently highlighted row in the workspace picker

	// ── Clock (Clocking tab) ──────────────────────────────────────────────────
	clockRunning bool
	clockStart   time.Time
	clockTask    string
	clockProject string
	clockFrame   int // animation frame index (incremented each tick)

	statusMsg string
	isError   bool

	version string
}

func (m Model) contentHeight() int {
	return m.height - headerHeight - footerHeight
}

// workDayDelegate wraps list.DefaultDelegate and renders non-working-day
// entries with a distinct colour so they stand out in the list view.
type workDayDelegate struct {
	list.DefaultDelegate
}

// Render overrides the default item rendering to apply the non-working-day
// colour scheme to items that were logged on an off-day.
func (d workDayDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	// Take an explicit local copy of the inner delegate so that per-item style
	// changes are confined to this call and never leak to subsequent items.
	inner := d.DefaultDelegate
	if di, ok := item.(dayListItem); ok && !di.isWorkDay && m.FilterState() == list.Unfiltered {
		inner.Styles.NormalTitle = listNonWorkDayTitle
		inner.Styles.NormalDesc = listNonWorkDayDesc
		inner.Styles.SelectedTitle = listNonWorkDaySelectedTitle
		inner.Styles.SelectedDesc = listNonWorkDaySelectedDesc
	}
	inner.Render(w, m, index, item)
}

func newDelegate() workDayDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.NormalTitle = listNormalTitle
	d.Styles.NormalDesc = listNormalDesc
	d.Styles.SelectedTitle = listSelectedTitle
	d.Styles.SelectedDesc = listSelectedDesc
	d.Styles.DimmedTitle = listDimmedTitle
	d.Styles.DimmedDesc = listDimmedDesc
	d.Styles.FilterMatch = listFilterMatch
	return workDayDelegate{DefaultDelegate: d}
}

// New constructs the initial model using the provided configuration.
func New(cfg config.Config, activeWorkspace string, version string) Model {
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

	weekHoursIn := textinput.New()
	weekHoursIn.Placeholder = fmt.Sprintf("%.0f", cfg.WeeklyHoursGoal)
	weekHoursIn.CharLimit = 8
	weekHoursIn.Width = 10
	weekHoursIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	weekHoursIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	weekHoursIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	return Model{
		cfg:             cfg,
		activeWorkspace: activeWorkspace,
		state:           stateList,
		list:            l,
		textarea:        ta,
		taskInput:       taskIn,
		projectInput:    projIn,
		durationInput:   durIn,
		timeInput:       timeIn,
		dateInput:       dateIn,
		weekHoursInput:  weekHoursIn,
		weekGoals:       journal.WeeklyGoals{},
		selectedEntry:   -1,
		editEntryIdx:    -1,
		version:         version,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadRecords, loadWeeklyGoals)
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
			isWork := true
			if t, err := r.ParseDate(); err == nil {
				isWork = m.effectiveIsWorkDay(t)
			}
			items[i] = dayListItem{rec: r, isWorkDay: isWork}
		}
		m.list.SetItems(items)
		return m, nil

	case weekGoalsLoadedMsg:
		m.weekGoals = msg.goals
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

	case clockTickMsg:
		if m.clockRunning {
			m.clockFrame++
			if m.state == stateDayView && m.dayViewTab == 0 {
				m.viewport.SetContent(m.renderDayContent())
			}
			return m, clockTickCmd()
		}
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
		case stateClockForm:
			return m.handleClockFormKey(msg)
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
		case stateWeekHoursInput:
			return m.handleWeekHoursInputKey(msg)
		case stateWorkspacePicker:
			return m.handleWorkspacePickerKey(msg)
		case stateStats:
			return m.handleStatsKey(msg)
		}
	}

	// Forward non-key messages to active sub-model.
	switch m.state {
	case stateList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case stateDayView, stateWeekView, stateStats:
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
	case stateClockForm:
		var cmd tea.Cmd
		if m.activeInput == 0 {
			m.taskInput, cmd = m.taskInput.Update(msg)
		} else {
			m.projectInput, cmd = m.projectInput.Update(msg)
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
