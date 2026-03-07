# Copilot Instructions — Schmournal

## What this is
A terminal-based work journal ("Schmournal") written in Go using the Charmbracelet stack:
- **bubbletea** — Elm-architecture TUI framework
- **bubbles** — pre-built components (list, textarea, textinput, viewport, cursor)
- **lipgloss** — ANSI styling/layout
- **glamour** — Markdown rendering (used in export preview)

Go module: `github.com/sleepypxnda/schmournal`

Data is stored as JSON files in `~/.journal/`, one file per day named `YYYY-MM-DD.json`.
Exports go to `~/.journal/exports/export-YYYY-MM-DD.md`.

---

## Package layout

```
main.go            — entry point; launches tea.NewProgram with AltScreen + MouseCellMotion
config/
  config.go        — Config/Keybinds structs, Load, Default, WriteDefault, ExpandPath, validate, migration
journal/
  types.go         — WorkEntry and DayRecord structs + pure logic (WorkTotals, DayDuration, Summary, ParseDate)
  journal.go       — file I/O: Dir, TodayPath, PathForDate, Load, Save, LoadAll, Delete, NewID
  worklog.go       — ParseDuration (flexible string → time.Duration), FormatDuration
  export.go        — ExportDay (Markdown string), SaveExport, consolidateEntries
  weekly_goals.go  — WeeklyGoals map, LoadWeeklyGoals, SaveWeeklyGoals
ui/
  model.go         — Model struct, view-state constants, messages, dayListItem, New, Init, Update
  commands.go      — tea.Cmd factories: loadRecords, loadWeeklyGoals, clearStatusCmd, saveDayCmd
  handlers.go      — handle*Key methods (one per view state)
  navigation.go    — openXxx helpers, focusField, scrollToSelected, weekKey/weeklyGoal helpers
  views.go         — View(), all viewXxx() and renderXxx() methods, joinKeyLabels, uniqueAppend
  styles.go        — all lipgloss styles; colour palette is Catppuccin Mocha (hex constants cBase … cRosewater)
```

### Test files
```
config/config_test.go   — Default, validate, collectTOMLPaths, ExpandPath
journal/types_test.go   — WorkTotals, DayDuration, Summary, ParseDate
journal/worklog_test.go — ParseDuration, FormatDuration (including round-trip)
journal/export_test.go  — ExportDay sections, consolidateEntries
```

Run all tests: `make test` or `go test ./...`  
Run a single package: `go test ./journal/...`  
Run a single test: `go test ./journal/... -run TestParseDuration`  
Build for current platform: `go build -o schmournal .`  
Cross-compile all platforms: `make build`

---

## Data model

```go
type WorkEntry struct {
    ID          string  // nanosecond timestamp string
    Project     string  // optional
    Task        string
    DurationMin int
    IsBreak     bool
}

type DayRecord struct {
    Date      string       // "YYYY-MM-DD"
    StartTime string       // "HH:MM", may be empty
    EndTime   string       // "HH:MM", may be empty
    Entries   []WorkEntry
    Notes     string       // freeform markdown text
    Path      string       // runtime only, not serialised
}
```

---

## View states (ui/model.go)

```
stateList             — scrollable list of all days (bubbles/list)
stateDayView          — two-tab view of a single day (tab 0: Work Log, tab 1: Summary)
stateWorkForm         — add/edit a work or break entry (2 textinputs for breaks, 3 for work)
stateClockForm        — start-clock dialog (task + optional project)
stateTimeInput        — set start or end time (single textinput)
stateNotesEditor      — freeform notes editor (bubbles/textarea)
stateConfirmDelete    — y/n confirmation for deleting a day or an entry
stateDateInput        — open or create a record for any arbitrary date
stateWeekView         — scrollable weekly summary with per-day totals
stateWeekHoursInput   — single textinput to set a custom weekly hours goal
stateWorkspacePicker  — scrollable picker for switching between workspaces
stateStats            — stats overview (multiple tabs: Overview, Monthly, Yearly, All-time)
```

`Update()` in `ui/model.go` dispatches `tea.KeyMsg` to a `handle*Key()` method per state (defined in `ui/handlers.go`). Non-key messages are forwarded to the active sub-model afterwards.

---

## Key patterns and conventions

- **Model is a value type** — all handler methods return `(tea.Model, tea.Cmd)`.
- **Status messages** auto-clear after 2 s via `clearStatusCmd()` (a `tea.Tick`).
- **`saveDayCmd(label)`** saves to disk then on success dispatches `loadRecords` (to refresh the list) + `clearStatusCmd`.
- **`openDayView(rec)`** always re-reads the record from disk before displaying it.
- **`viewport`** (bubbles) is used as the scrollable body for `stateDayView`. Its content is rebuilt via `m.renderDayContent()` whenever data or selection changes.
- **`scrollToSelected()`** uses a hardcoded `entryStartLine = 7` to map the selected entry index to a viewport line.
- **New breaks** with the same label (case-insensitive) are merged into the existing break entry rather than appended.
- **Form focus management**: `focusField(n)` blurs all three inputs then focuses the requested one.
- **Active workspace** is persisted across sessions via `config.AppState` in `~/.config/schmournal.state` (JSON). `config.LoadState`/`config.SaveState` manage this file; `main.go` resolves the startup workspace via `resolveActiveWorkspace`.
- **Config migration**: on load, if any TOML keys are missing the old file is renamed to `schmournal.old.config` and a fresh file is written preserving all user values. `collectTOMLPaths` derives expected keys from struct tags automatically — add a `toml` tag to any new `Config` field and migration is handled for free.
- **Adding a new keybind**: add a field to the appropriate `*Keybinds` struct in `config/config.go`, add a default in `Default()`, add a `fill(...)` call in `validate()`, add it to the `checkDuplicates(...)` call for that view, and update `generateConfigContent`.

---

## textarea (notes editor) — known quirks

- `ta.ShowLineNumbers = false` — line numbers are off, but the default prompt (`"┃ "`) is still present; this is accounted for inside `bubbles/textarea.SetWidth`.
- `ta.FocusedStyle.CursorLine = lipgloss.NewStyle()` — deliberately set to an empty style. A background colour on `CursorLine` caused the current line to visually wrap (the style extended the rendered line width, tricking the internal viewport into soft-wrapping). Do **not** restore a background here without testing for the wrapping regression.
- Sizing: `textarea.SetWidth(m.width - 4)` and the surrounding `editorBorderStyle.Width(m.width - 4)` are intentionally the same value because `editorBorderStyle` has `Border(RoundedBorder()) + Padding(0,1)` = 4 columns of frame, making the outer box exactly `m.width` wide.

---

## Styling

All colours are Catppuccin Mocha defined as hex constants at the top of `ui/styles.go`.  
Named style variables live in package-level `var` blocks grouped by concern (chrome, form, list delegate, day view).  
The `editorBorderStyle` wraps the textarea with a rounded border and 1-column horizontal padding.

---

## Export

`journal.ExportDay(rec)` returns a Markdown string. Work entries are grouped by project; duplicate task names within a project are consolidated (durations summed). Breaks are consolidated by label (case-sensitive in export, case-insensitive when logging). The output is written to `~/.journal/exports/export-YYYY-MM-DD.md` by `journal.SaveExport`.

