# 📔 Schmournal

A minimal, distraction-free terminal journaling app built with Go and the [Charm](https://charm.sh) TUI stack, themed with **Catppuccin Mocha**.

<!-- screenshot: list-view — the main list of days with the stats bar -->
<!-- ![List view](docs/screenshots/list-view.png) -->

## Features

- **List view** — all days sorted newest-first with an entry count and work-time preview
- **Stats bar** — current-week activity bar, monthly entry count, and streak tracking
- **Day view** — two-tab view: Work Log (entries table) and Summary
- **Notes editor** — full-screen textarea for free-form notes per day
- **Work day tracking** — log start/end times, work items (with optional project), and breaks
- **Multi-project split** — enter comma-separated projects to split a task across them automatically
- **Weekly summary** — scrollable week overview with per-day totals, navigable across past weeks
- **Daily export** — generates a Markdown report grouped by project
- **Open any day** — open or create a journal entry for any arbitrary date
- **Delete** — with confirmation dialog (single entry or whole day)
- **Filter / search** — built-in fuzzy filtering with `/`
- **Version flag** — `schmournal --version` prints the current version

Records are stored as JSON files in `~/.journal/YYYY-MM-DD.json`.  
Exports are written to `~/.journal/exports/export-YYYY-MM-DD.md`.

---

## Key bindings

### 📋 List view

<!-- screenshot: list-view -->

| Key | Action |
|-----|--------|
| `n` | Open today's entry (creates it if it doesn't exist) |
| `c` | Open or create an entry for any date |
| `enter` | View selected day |
| `d` | Delete selected day (with confirmation) |
| `w` | Log a work item for today |
| `b` | Log a break for today |
| `x` | Export the selected day's work log |
| `v` | Weekly summary view |
| `/` | Filter entries |
| `q` / `esc` | Quit |
| `ctrl+c` | Force quit |

### 👁 Day view — Work Log tab

<!-- screenshot: day-view-work-log -->

| Key | Action |
|-----|--------|
| `←` / `→` | Switch between Work Log and Summary tabs |
| `j` / `↓` | Select next entry |
| `k` / `↑` | Select previous entry |
| `w` | Log a new work item |
| `b` | Log a new break |
| `e` | Edit the selected entry (or open notes editor if none selected) |
| `d` | Delete the selected entry (or the whole day if none selected) |
| `s` | Stamp current time as **Start** |
| `S` | Open dialog to manually **set Start time** |
| `f` | Stamp current time as **End** (finish) |
| `F` | Open dialog to manually **set End time** |
| `N` | Open notes editor |
| `x` | Export this day's work log |
| `esc` | Back to list |

### 📅 Weekly summary view

<!-- screenshot: week-view -->

| Key | Action |
|-----|--------|
| `←` / `h` | Previous week |
| `→` / `l` | Next week |
| `j` / `k` | Scroll content |
| `esc` / `q` | Back to list |

### ✏️ Notes editor

| Key | Action |
|-----|--------|
| `ctrl+s` | Save notes |
| `esc` | Cancel (discard changes) |

### 📝 Work / Break log form

<!-- screenshot: work-form -->

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Cycle between fields |
| `enter` | Advance to next field / submit on last field |
| `esc` | Cancel |

Work items have three fields: **Task**, **Project** (optional), **Duration**.  
Break items have two fields: **Label**, **Duration**.

The **Project** field accepts a comma-separated list of projects (e.g. `Frontend, Backend`).
When multiple projects are supplied the logged duration is split evenly across them.

Duration examples: `1h 30m` · `45m` · `2h` · `1.5h` · `90` (bare number = minutes)

### ⏰ Time input dialog (S / F)

| Key | Action |
|-----|--------|
| `enter` | Confirm time |
| `esc` | Cancel |

Input format: `HH:MM` (e.g. `09:00`, `14:30`)

### 📆 Date input dialog (c)

| Key | Action |
|-----|--------|
| `enter` | Open or create the day |
| `esc` | Cancel |

Input format: `YYYY-MM-DD`

---

## Day view

Each day record has two tabs:

**Work Log tab** — shows the start/end time bar, a table of all work and break entries (with the currently selected entry highlighted), and a work/break/total summary line.

**Summary tab** — shows a compact summary with start time, end time, day duration, total work, total breaks, and logged notes.

<!-- screenshot: day-view-summary-tab -->

---

## Export

Pressing `x` generates a Markdown report at `~/.journal/exports/export-YYYY-MM-DD.md` containing:

- **🕐 Work Day** — start, end, day duration
- **📋 Work Items** — grouped by project with per-project subtotals; same-named tasks within a project are consolidated
- **☕ Breaks** — consolidated break list with total
- **📊 Summary** — work, breaks, total logged, day duration

---

## Installation

### Homebrew (macOS / Linux)

```bash
brew install SleepyPxnda/schmournal/schmournal
```

Or tap first if you want to keep it updated via `brew upgrade`:

```bash
brew tap SleepyPxnda/schmournal https://github.com/SleepyPxnda/schmournal
brew install schmournal
```

> **Note:** The formula is automatically updated on every release. Run `brew upgrade schmournal` to get the latest version.

### Build from source

```bash
# Build for your current platform
go build -o schmournal .
./schmournal

# Cross-compile for all platforms (output in dist/)
make build          # all platforms
make build-mac      # macOS arm64 + amd64
make build-linux    # Linux amd64 + arm64
make build-windows  # Windows amd64 + arm64

# Clean build artefacts
make clean
```

### Version

```bash
schmournal --version
```

---

## Theme

Uses the **Catppuccin Mocha** palette throughout — Mauve accents, Lavender highlights, and the full Base/Surface/Overlay colour system for a consistent dark-mode look.

