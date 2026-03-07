# 📔 Schmournal

A minimal, distraction-free terminal journaling app built with Go and the [Charm](https://charm.sh) TUI stack, themed with **Catppuccin Mocha**.

![List view](images/overview.png)

## Features

- **List view** — all days sorted newest-first with an entry count and work-time preview
- **Stats bar** — current-week activity bar, monthly entry count, and streak tracking
- **Day view** — two-tab view: Work Log (entries table + live clock panel) and Summary
- **Notes editor** — full-screen textarea for free-form notes per day
- **Work day tracking** — log start/end times, work items (with optional project), and breaks
- **Multi-project split** — enter comma-separated projects to split a task across them automatically
- **Clock / timer** — start a live timer from the Work Log tab; it appears as a side panel next to the entry list and is automatically logged when stopped
- **Workspaces** — maintain multiple independent journal directories (e.g. personal vs. work) and switch between them with a picker dialog
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

![List view](images/overview.png)

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
| `p` | Open the workspace picker |
| `/` | Filter entries |
| `q` / `esc` | Quit |
| `ctrl+c` | Force quit |

### 👁 Day view — Work Log tab

![Day view](images/day-worklog.png)

![Day view with clock running](images/day-worklog-clock.png)

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
| `n` | Open notes editor |
| `c` | **Start** the clock timer |
| `t` | **Stop** the running clock and log the entry |
| `x` | Export this day's work log |
| `esc` | Back to list |

### 📅 Weekly summary view

![Week Summary view](images/week-overview.png)

| Key | Action |
|-----|--------|
| `←` / `h` | Previous week |
| `→` / `l` | Next week |
| `g` | Set a custom hours goal for the displayed week |
| `j` / `k` | Scroll content |
| `esc` / `q` | Back to list |

### ✏️ Notes editor

| Key | Action |
|-----|--------|
| `ctrl+s` | Save notes |
| `esc` | Cancel (discard changes) |

### 📝 Work / Break log form

![Work log form](images/day-worklog-form.png)

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
| `r` | Reset / clear the time |
| `esc` | Cancel |

Input format: `HH:MM` (e.g. `09:00`, `14:30`)

### 📆 Date input dialog (c)

| Key | Action |
|-----|--------|
| `enter` | Open or create the day |
| `esc` | Cancel |

Input format: `YYYY-MM-DD`

### ⏱ Clock / timer (c / t)

The clock lets you track time against a task in real time without having to estimate the duration up front.

1. From the **Work Log tab**, press `c` to open the **Start Clock** form — enter the task name and an optional project (comma-separated for multi-project split).
2. Press `enter` to start the timer. The Work Log tab immediately shows a live clock panel on the right-hand side next to your entry list, updating every second.
3. Press `t` at any time to stop the timer. The elapsed duration is rounded to the nearest minute and a new work entry is added automatically. If multiple projects were supplied the duration is split evenly across them.

![Clock panel running in Work Log tab](images/day-worklog-clock.png)

### 🗂 Workspace picker (p)

Workspaces let you keep entirely separate journal directories (e.g. one for personal use, one for your job). When two or more workspaces are configured in `~/.config/schmournal.config`, a workspace indicator appears in the list view header and you can switch between workspaces at any time.

Press `p` from the **list view** to open the picker.

<!-- Placeholder — take a new screenshot of the workspace picker dialog -->
![Workspace picker](images/workspace-picker.png)

| Key | Action |
|-----|--------|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `enter` | Switch to the selected workspace |
| `esc` | Cancel |

---

## Day view

Each day record has two tabs:

**Work Log tab** — shows the start/end time bar, a table of all work and break entries (with the currently selected entry highlighted), and a work/break/total summary line. On terminals 60 columns wide or wider, a **clock panel** is shown on the right-hand side of the tab. The panel displays "No active timer" when idle and an animated live elapsed timer (HH:MM:SS) when the clock is running.

**Summary tab** — shows a compact summary with start time, end time, day duration, total work, total breaks, and logged notes.

![Day Summary view](images/day-summary.png)


---

## Configuration

Schmournal reads its configuration from `~/.config/schmournal.config` (TOML format). The file is created automatically with defaults on first run.

### `weekly_hours_goal`

Sets the default weekly working-hours target used in the stats bar progress meter and the weekly summary view.

```toml
weekly_hours_goal = 40   # hours (default: 40)
```

You can also override this on a per-week basis from the **weekly summary view** by pressing `g`. The override is stored in `~/.journal/weekly_goals.json` and shown as "(custom)" next to the goal in the week total line. Leave the input empty and press `enter` to reset a week back to the global default.

### Workspaces

Workspaces let you maintain separate journal directories, each with its own `storage_path` and optional `weekly_hours_goal`. When at least one workspace is defined you can switch between them from the list view with `p`.

```toml
[[workspaces]]
name              = "Personal"
storage_path      = "~/.journal/personal"
weekly_hours_goal = 40

[[workspaces]]
name              = "Work"
storage_path      = "~/.journal/work"
weekly_hours_goal = 37.5
```

Fields omitted from a workspace entry fall back to the top-level defaults. Workspace names must be unique and must not have leading or trailing whitespace.

### Keybinds

All keybinds can be customised in `~/.config/schmournal.config`. Example:

```toml
[keybinds.list]
switch_workspace = "p"   # open the workspace picker

[keybinds.day]
clock_start = "c"        # start the clock timer
clock_stop  = "t"        # stop the clock and log the entry

[keybinds.week]
prev_week        = "h"
next_week        = "l"
set_weekly_hours = "g"   # set custom goal for the displayed week
```

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

