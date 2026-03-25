package ui

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/sleepypxnda/schmournal/journal"
)

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

type todoCursor struct {
	top int
	sub int // -1 parent, >=0 subtodo
}

func (m *Model) todoMove(delta int) {
	cursors := m.todoCursors()
	if len(cursors) == 0 {
		m.selectedTodo = -1
		m.selectedSub = -1
		return
	}
	idx := 0
	for i, c := range cursors {
		if c.top == m.selectedTodo && c.sub == m.selectedSub {
			idx = i
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cursors) {
		idx = len(cursors) - 1
	}
	m.selectedTodo = cursors[idx].top
	m.selectedSub = cursors[idx].sub
}

func (m Model) todoCursors() []todoCursor {
	var out []todoCursor
	for i, t := range m.dayRecord.Todos {
		out = append(out, todoCursor{top: i, sub: -1})
		for j := range t.Subtodos {
			out = append(out, todoCursor{top: i, sub: j})
		}
	}
	return out
}

func (m *Model) toggleSelectedTodo() bool {
	if m.selectedTodo < 0 || m.selectedTodo >= len(m.dayRecord.Todos) {
		return false
	}
	if m.selectedSub >= 0 {
		if m.selectedSub >= len(m.dayRecord.Todos[m.selectedTodo].Subtodos) {
			return false
		}
		m.dayRecord.Todos[m.selectedTodo].Subtodos[m.selectedSub].Completed = !m.dayRecord.Todos[m.selectedTodo].Subtodos[m.selectedSub].Completed
		return true
	}
	m.dayRecord.Todos[m.selectedTodo].Completed = !m.dayRecord.Todos[m.selectedTodo].Completed
	return true
}

func (m Model) buildTodoOverviewItems() []todoOverviewItem {
	records := m.records
	if loaded, err := journal.LoadAll(); err == nil {
		records = loaded
	}
	type dayRec struct {
		date string
		path string
		t    []todoOverviewItem
	}
	var days []dayRec
	for _, r := range records {
		if len(r.Todos) == 0 {
			continue
		}
		items := make([]todoOverviewItem, 0, len(r.Todos))
		for _, td := range r.Todos {
			if !m.todoOverviewOnlyU || !td.Completed {
				items = append(items, todoOverviewItem{
					date:      r.Date,
					path:      r.Path,
					title:     td.Title,
					completed: td.Completed,
					parentID:  td.ID,
					depth:     0,
				})
			}
			for _, st := range td.Subtodos {
				if m.todoOverviewOnlyU && st.Completed {
					continue
				}
				items = append(items, todoOverviewItem{
					date:      r.Date,
					path:      r.Path,
					title:     st.Title,
					completed: st.Completed,
					parentID:  td.ID,
					subID:     st.ID,
					depth:     1,
				})
			}
		}
		if len(items) > 0 {
			days = append(days, dayRec{date: r.Date, path: r.Path, t: items})
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i].date < days[j].date })
	var out []todoOverviewItem
	for _, d := range days {
		out = append(out, d.t...)
	}
	return out
}

func (m Model) renderTodosPanel(w int) string {
	var b strings.Builder
	b.WriteString(dayViewSectionStyle.Render("✅  Todos") + "\n")
	if len(m.dayRecord.Todos) == 0 {
		b.WriteString(dayViewMutedStyle.Render("  No todos yet") + "\n")
		b.WriteString(dayViewMutedStyle.Render("  a add todo") + "\n")
		return b.String()
	}
	for i, td := range m.dayRecord.Todos {
		mark := todoIncompleteStyle.Render("—")
		if td.Completed {
			mark = todoCompleteStyle.Render("✓")
		}
		prefix := "  "
		if m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == -1 {
			prefix = "▶ "
		}
		line := prefix + mark + " " + td.Title
		if lipgloss.Width(line) > w {
			line = truncateRunes(line, w)
		}
		if m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == -1 {
			line = selectedEntryStyle.Render(line)
		}
		b.WriteString(line + "\n")

		for j, st := range td.Subtodos {
			smark := todoIncompleteStyle.Render("—")
			if st.Completed {
				smark = todoCompleteStyle.Render("✓")
			}
			sprefix := "    "
			if m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == j {
				sprefix = "  ▶ "
			}
			sline := sprefix + smark + " " + st.Title
			if lipgloss.Width(sline) > w {
				sline = truncateRunes(sline, w)
			}
			if m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == j {
				sline = selectedEntryStyle.Render(sline)
			}
			b.WriteString(sline + "\n")
		}
	}
	return b.String()
}

func (m Model) renderTodoOverviewContent() string {
	var b strings.Builder
	b.WriteString("\n")
	if len(m.todoOverviewItems) == 0 {
		if m.todoOverviewOnlyU {
			b.WriteString(dayViewMutedStyle.Render("  No uncompleted todos") + "\n")
		} else {
			b.WriteString(dayViewMutedStyle.Render("  No todos found") + "\n")
		}
		return b.String()
	}
	lastDate := ""
	for i, it := range m.todoOverviewItems {
		if it.date != lastDate {
			if lastDate != "" {
				b.WriteString("\n")
			}
			b.WriteString(dayViewSectionStyle.Render("  "+it.date) + "\n")
			lastDate = it.date
		}
		prefix := "  "
		if i == m.todoOverviewIdx {
			prefix = "▶ "
		}
		indent := ""
		if it.depth > 0 {
			indent = "  "
		}
		mark := todoIncompleteStyle.Render("—")
		if it.completed {
			mark = todoCompleteStyle.Render("✓")
		}
		line := prefix + indent + mark + " " + it.title
		if i == m.todoOverviewIdx {
			line = selectedEntryStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
