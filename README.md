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
- **Version flag** — `schmournal --version` prints the current version

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
| `s` | Sync with cloud (requires rclone configuration) |
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

## ☁️ Cloud Sync

Schmournal supports syncing your journal entries across devices using [rclone](https://rclone.org), a command-line tool that works with 70+ cloud storage providers (Google Drive, Dropbox, S3, OneDrive, Backblaze B2, SFTP, and more).

### Setup

1. **Install rclone** — see [rclone.org/install](https://rclone.org/install/)

2. **Configure a remote** — run `rclone config` and follow the prompts to add a remote for your provider. Example remotes:
   - `gdrive:journal` — Google Drive, a folder called `journal`
   - `dropbox:journal` — Dropbox
   - `s3:mybucket/journal` — Amazon S3

3. **Create `~/.journal/config.json`** with your remote path:

   ```json
   {
     "sync": {
       "remote": "gdrive:journal",
       "direction": "both"
     }
   }
   ```

   | Field | Values | Description |
   |-------|--------|-------------|
   | `remote` | rclone remote path | Destination/source for sync (**required**) |
   | `direction` | `"both"` (default), `"push"`, `"pull"` | `"both"` merges both sides; `"push"` uploads only; `"pull"` downloads only |

4. **Press `s`** in the list view to sync. Only day-record files (`YYYY-MM-DD.json`) are transferred — `config.json` and exports remain device-local.

> **Note:** Sync uses `rclone copy` which copies newer/missing files without deleting. It does not resolve conflicts; if the same day was edited on two devices simultaneously, the file written last wins.

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

