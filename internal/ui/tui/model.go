package tui

import (
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	domainmodel "github.com/sleepypxnda/schmournal/internal/domain/model"
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
	stateTodoForm
	stateConfirmDelete
	stateDateInput
	stateWeekView
	stateWorkspacePicker
	stateStats
)

const (
	headerHeight = 2 // title line + separator
	footerHeight = 1
	statsHeight  = 3 // week bar + progress bar + separator
)

// ── Messages ──────────────────────────────────────────────────────────────────

type recordsLoadedMsg struct{ records []DayRecord }
type daySavedMsg struct{ label string }
type dayDeletedMsg struct{}
type clearStatusMsg struct{}
type errMsg struct{ err error }
type workspaceTodosLoadedMsg struct{ todos WorkspaceTodos }
type workspaceTodosManagedMsg struct {
	todos          WorkspaceTodos
	completedToday []Todo
	label          string
}
type workFormSubmittedMsg struct {
	record   DayRecord
	label    string
	entryIdx int
}
type clockTickMsg struct{}

// ClockState holds all runtime clock/timer fields.
type ClockState struct {
	Running bool
	Start   time.Time
	Task    string
	Project string
	Frame   int // animation frame index (incremented each tick)
}

// DeleteState groups confirmation-dialog state.
type DeleteState struct {
	Day       bool // true = confirm delete whole day, false = confirm delete entry
	Idx       int  // index in records (Day) or entries (!Day)
	PrevState viewState
}

// StatusState groups transient footer status feedback.
type StatusState struct {
	Message string
	IsError bool
}

// WorkFormState groups work/break form state and controls.
type WorkFormState struct {
	TaskInput     textinput.Model
	ProjectInput  textinput.Model
	DurationInput textinput.Model
	ActiveInput   int
	IsBreakEntry  bool
	EditEntryIdx  int // -1 = new, >=0 = editing existing entry
}

// TimeInputState groups state for manual start/end time input.
type TimeInputState struct {
	Input   textinput.Model
	IsStart bool
}

// DateInputState groups state for opening/creating a day by date.
type DateInputState struct {
	Input textinput.Model
}

// SelectionState groups day-view selection/navigation cursor state.
type SelectionState struct {
	EntryIdx int // index into dayRecord.Entries; -1 = no selection
	DayTab   int // 0 = Work Log, 1 = Summary
	Pane     int // 0 = work log entries, 1 = todos
}

// TodoEditorState groups todo edit/input runtime state.
type TodoEditorState struct {
	EditTop   int // -1 = new, >=0 = editing top-level todo index
	EditSub   int // -1 = top-level, >=0 = editing level-2 todo index
	EditSub2  int // -1 = not level-3, >=0 = editing level-3 todo index
	Draft     string
	InputMode bool
	Input     textinput.Model
}

// TodoSelectionState tracks selection cursor within workspace todos.
type TodoSelectionState struct {
	Top  int // top-level todo index
	Sub  int // -1 = top-level, >=0 = level-2 todo index
	Sub2 int // -1 = not level-3, >=0 = level-3 todo index
}

// StatsViewState groups tab selection for the stats view.
type StatsViewState struct {
	Tab int // 0=Overview 1=Monthly 2=Yearly 3=All-time
}

// WorkspacePickerState groups current selection in workspace picker.
type WorkspacePickerState struct {
	Index int // currently highlighted row in the workspace picker
}

// ListState groups list component model and loaded day records.
type ListState struct {
	Model   list.Model
	Records []DayRecord
}

// UIState groups top-level app navigation state and static metadata.
type UIState struct {
	Current viewState
	Version string
}

// WindowState groups runtime terminal/window dimensions.
type WindowState struct {
	Width  int
	Height int
	Ready  bool
}

// AppContextState groups injected dependencies and app-level context.
type AppContextState struct {
	Config          domainmodel.AppConfig
	ActiveWorkspace string // empty = no workspaces
	UseCases        *UseCases
}

// WorkspaceDataState groups workspace-wide todo datasets.
type WorkspaceDataState struct {
	Todos []Todo
}

// DayViewState groups day-view record, selection and widgets.
type DayViewState struct {
	Record    DayRecord
	Selection SelectionState
	Viewport  viewport.Model
	Notes     textarea.Model
}

// ── List item ─────────────────────────────────────────────────────────────────

type dayListItem struct {
	rec       DayRecord
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

// Model holds the entire application state grouped into focused sub-states.
type Model struct {
	// ── App shell state ────────────────────────────────────────────────────────
	ui      UIState
	window  WindowState
	context AppContextState

	// ── List view state ────────────────────────────────────────────────────────
	listState ListState

	// ── Workspace-level data ───────────────────────────────────────────────────
	workspace WorkspaceDataState

	// ── Day view state ─────────────────────────────────────────────────────────
	day DayViewState

	// ── Todo pane state (within day view) ──────────────────────────────────────
	todoSelection TodoSelectionState
	todoEditor    TodoEditorState

	// ── Work/Break form state ──────────────────────────────────────────────────
	workForm WorkFormState

	// ── Time input state ───────────────────────────────────────────────────────
	timeForm TimeInputState

	// ── Date input state ───────────────────────────────────────────────────────
	dateForm DateInputState

	// ── Delete confirmation state ──────────────────────────────────────────────
	delete DeleteState

	// ── Stats and picker state ─────────────────────────────────────────────────
	stats           StatsViewState
	workspacePicker WorkspacePickerState
	weekOffset      int

	// ── Clock timer state ──────────────────────────────────────────────────────
	clock ClockState

	// ── Status message (global) ────────────────────────────────────────────────
	status StatusState
}

func (m Model) contentHeight() int {
	return m.window.Height - headerHeight - footerHeight
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
func New(cfg domainmodel.AppConfig, activeWorkspace string, version string, useCases *UseCases) Model {
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

	todoIn := textinput.New()
	todoIn.Placeholder = "TODO title…"
	todoIn.CharLimit = 160
	todoIn.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cMauve))
	todoIn.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cText))
	todoIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cOverlay0))

	return Model{
		ui: UIState{
			Current: stateList,
			Version: version,
		},
		context: AppContextState{
			Config:          cfg,
			ActiveWorkspace: activeWorkspace,
			UseCases:        useCases,
		},
		listState: ListState{
			Model: l,
		},
		day: DayViewState{
			Selection: SelectionState{
				EntryIdx: -1,
				DayTab:   0,
				Pane:     0,
			},
			Notes: ta,
		},
		workForm: WorkFormState{
			TaskInput:     taskIn,
			ProjectInput:  projIn,
			DurationInput: durIn,
			EditEntryIdx:  -1,
		},
		timeForm: TimeInputState{
			Input: timeIn,
		},
		dateForm: DateInputState{
			Input: dateIn,
		},
		workspace: WorkspaceDataState{
			Todos: []Todo{},
		},
		todoSelection: TodoSelectionState{
			Top:  0,
			Sub:  -1,
			Sub2: -1,
		},
		todoEditor: TodoEditorState{
			EditTop:  -1,
			EditSub:  -1,
			EditSub2: -1,
			Input:    todoIn,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadRecordsCmd(), m.loadWorkspaceTodosCmd())
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.window.Width = msg.Width
		m.window.Height = msg.Height
		ch := m.contentHeight()
		vpH := ch - 2 // 2 lines for tab bar + separator
		if !m.window.Ready {
			m.day.Viewport = viewport.New(m.window.Width, vpH)
			m.window.Ready = true
		} else {
			m.day.Viewport.Width = m.window.Width
			m.day.Viewport.Height = vpH
		}
		listH := ch - statsHeight
		if m.renderEOMBanner() != "" {
			listH--
		}
		m.listState.Model.SetSize(m.window.Width, listH)
		m.day.Notes.SetWidth(m.window.Width - 4)
		m.day.Notes.SetHeight(ch - 2)
		return m, nil

	case recordsLoadedMsg:
		m.listState.Records = msg.records
		items := make([]list.Item, len(m.listState.Records))
		for i, r := range m.listState.Records {
			isWork := true
			if t, err := r.ParseDate(); err == nil {
				isWork = m.effectiveIsWorkDay(t)
			}
			items[i] = dayListItem{rec: r, isWorkDay: isWork}
		}
		m.listState.Model.SetItems(items)
		return m, nil

	case workspaceTodosLoadedMsg:
		m.workspace.Todos = msg.todos.Todos
		if m.ui.Current == stateDayView && m.day.Selection.DayTab == 0 {
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		return m, nil

	case workspaceTodosManagedMsg:
		m.workspace.Todos = msg.todos.Todos
		if len(msg.completedToday) > 0 {
			m.day.Record.TodayDone = mergeTodayDoneTrees(m.day.Record.TodayDone, msg.completedToday)
			return m, m.saveDayCmd("")
		}
		if m.ui.Current == stateDayView && m.day.Selection.DayTab == 0 {
			m.day.Viewport.SetContent(m.renderDayContent())
		}
		if msg.label != "" {
			m.status.Message = msg.label
			m.status.IsError = false
			return m, clearStatusCmd()
		}
		return m, nil

	case workFormSubmittedMsg:
		m.day.Record = msg.record
		m.day.Selection.EntryIdx = msg.entryIdx
		m.ui.Current = stateDayView
		m.day.Viewport.SetContent(m.renderDayContent())
		m.scrollToSelected()
		return m, func() tea.Msg { return daySavedMsg{label: msg.label} }

	case daySavedMsg:
		m.status.Message = msg.label
		m.status.IsError = false
		return m, tea.Batch(m.loadRecordsCmd(), clearStatusCmd())

	case dayDeletedMsg:
		m.status.Message = "✓ Day deleted"
		m.status.IsError = false
		m.ui.Current = stateList
		return m, tea.Batch(m.loadRecordsCmd(), clearStatusCmd())

	case errMsg:
		m.status.Message = "✗ " + msg.err.Error()
		m.status.IsError = true
		return m, clearStatusCmd()

	case clearStatusMsg:
		m.status.Message = ""
		m.status.IsError = false
		return m, nil

	case clockTickMsg:
		if m.clock.Running {
			m.clock.Frame++
			if m.ui.Current == stateDayView && m.day.Selection.DayTab == 0 {
				m.day.Viewport.SetContent(m.renderDayContent())
			}
			return m, clockTickCmd()
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.routeKeyMsg(msg)
	}

	return m.routeSubModelMsg(msg)
}
