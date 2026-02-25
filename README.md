# 📔 Schmournal

A minimal, distraction-free terminal journaling app built with Go and the [Charm](https://charm.sh) TUI stack, themed with **Catppuccin Mocha**.

## Features

- **List view** — all entries sorted newest-first with a content preview
- **Viewer** — beautifully rendered Markdown via Glamour
- **Editor** — full-screen textarea for writing
- **Create** — one keypress opens today's entry (creates it if missing)
- **Delete** — with confirmation dialog
- **Filter / search** — built-in fuzzy filtering with `/`

Entries are stored as plain Markdown files in `~/.journal/YYYY-MM-DD.md`.

## Key bindings

| Key | Action |
|-----|--------|
| `n` | New / open today's entry |
| `enter` | View selected entry |
| `e` | Edit selected entry |
| `d` | Delete selected entry |
| `/` | Filter entries |
| `q` / `esc` | Quit / go back |
| `ctrl+s` | Save (in editor) |
| `ctrl+c` | Force quit |

## Install & run

```bash
go install github.com/fgrohme/tui-journal@latest
# or build from source:
go build -o tui-journal .
./tui-journal
```

## Theme

Uses the **Catppuccin Mocha** palette throughout — Mauve accents, Lavender highlights, and the full Base/Surface/Overlay colour system for a consistent dark-mode look.
