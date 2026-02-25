# 📔 Schmournal

A minimal, distraction-free terminal journaling app built with Go and the [Charm](https://charm.sh) TUI stack, themed with **Catppuccin Mocha**.

## Features

- **List view** — all entries sorted newest-first with a content preview
- **Viewer** — beautifully rendered Markdown via Glamour
- **Editor** — full-screen textarea for writing
- **Work day tracking** — log start/end times, work items (with project), and breaks
- **Daily export** — generates a Markdown report grouped by project
- **Create** — one keypress opens today's entry (creates it from template if missing)
- **Delete** — with confirmation dialog
- **Filter / search** — built-in fuzzy filtering with `/`

Entries are stored as plain Markdown files in `~/.journal/YYYY-MM-DD.md`.  
Exports are written to `~/.journal/exports/export-YYYY-MM-DD.md`.

---

## Key bindings

### 📋 List view

| Key | Action |
|-----|--------|
| `n` | New / open today's entry |
| `enter` | View selected entry |
| `e` | Edit selected entry |
| `d` | Delete selected entry (with confirmation) |
| `w` | Log a work item for today |
| `b` | Log a break for today |
| `x` | Export today's work log |
| `/` | Filter entries |
| `q` | Quit |
| `ctrl+c` | Force quit |

### 👁 Viewer

| Key | Action |
|-----|--------|
| `e` | Edit entry |
| `d` | Delete entry |
| `w` | Log a work item to this entry |
| `b` | Log a break to this entry |
| `s` | Stamp current time as **Start** |
| `S` | Open dialog to manually **set Start time** |
| `f` | Stamp current time as **End** (finish) |
| `F` | Open dialog to manually **set End time** |
| `x` | Export this entry's work log |
| `↑ / ↓` | Scroll content |
| `esc` / `q` | Back to list |

### ✏️ Editor

| Key | Action |
|-----|--------|
| `ctrl+s` | Save entry |
| `esc` | Cancel (discard changes) |

### 📝 Work / Break log form

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Cycle between fields |
| `enter` | Advance to next field / submit on last field |
| `esc` | Cancel |

Work items have three fields: **Task**, **Project** (optional), **Duration**.  
Break items have two fields: **Label**, **Duration**.

Duration examples: `1h 30m` · `45m` · `2h` · `1.5h` · `90` (bare number = minutes)

### ⏰ Time input dialog (S / F)

| Key | Action |
|-----|--------|
| `enter` | Confirm time |
| `esc` | Cancel |

Input format: `HH:MM` (e.g. `09:00`, `14:30`)

---

## Daily template

Each new entry is created with the following sections:

```
## 🕐 Work Day
Start / End timestamps

## 📋 Work Log
Project | Task | Duration table + Work / Breaks / Total summary

## 📝 Notes
Free-form notes
```

---

## Export

Pressing `x` generates a Markdown report at `~/.journal/exports/export-YYYY-MM-DD.md` containing:

- **🕐 Work Day** — start, end, day duration
- **📋 Work Items** — grouped by project with per-project subtotals; same-named tasks within a project are consolidated
- **☕ Breaks** — consolidated break list with total
- **📊 Summary** — work, breaks, total logged, day duration

---

## Build & run

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

---

## Theme

Uses the **Catppuccin Mocha** palette throughout — Mauve accents, Lavender highlights, and the full Base/Surface/Overlay colour system for a consistent dark-mode look.

