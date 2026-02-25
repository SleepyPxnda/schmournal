# Copilot Instructions — Schmournal

## What this is
A terminal-based work journal ("Schmournal") written in Go using the Charmbracelet stack:
- **bubbletea** — Elm-architecture TUI framework
- **bubbles** — pre-built components (list, textarea, textinput, viewport, cursor)
- **lipgloss** — ANSI styling/layout
- **glamour** — Markdown rendering (used in export preview)

Data is stored as JSON files in `~/.journal/`, one file per day named `YYYY-MM-DD.json`.
Exports go to `~/.journal/exports/export-YYYY-MM-DD.md`.

---

## Package layout

```
main.go            — entry point; launches tea.NewProgram with AltScreen + MouseCellMotion
journal/
  types.go         — WorkEntry and DayRecord structs + pure logic (WorkTotals, DayDuration, Summary, ParseDate)
  journal.go       — file I/O: Dir, TodayPath, PathForDate, Load, Save, LoadAll, Delete, NewID
  worklog.go       — ParseDuration (flexible string → time.Duration), FormatDuration
  export.go        — ExportDay (Markdown string), SaveExport, consolidateEntries
ui/
  model.go         — single bubbletea Model: all state, Init/Update/View, key handlers, nav helpers, view renderers
  styles.go        — all lipgloss styles; colour palette is Catppuccin Mocha (hex constants cBase … cRosewater)
```

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
stateList          — scrollable list of all days (bubbles/list)
stateDayView       — two-tab view of a single day (tab 0: Work Log, tab 1: Summary)
stateWorkForm      — add/edit a work or break entry (2 textinputs for breaks, 3 for work)
stateTimeInput     — set start or end time (single textinput)
stateNotesEditor   — freeform notes editor (bubbles/textarea)
stateConfirmDelete — y/n confirmation for deleting a day or an entry
stateDateInput     — open or create a record for any arbitrary date
```

`Update()` dispatches `tea.KeyMsg` to a `handle*Key()` method per state. Non-key messages are forwarded to the active sub-model afterwards.

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
