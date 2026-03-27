package ui

import (
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
	top  int
	sub  int // -1 top-level, >=0 level-2 todo index
	sub2 int // -1 not level-3, >=0 level-3 todo index
}

func todoLinePrefix(depth int, selected bool) string {
	indent := strings.Repeat("  ", depth)
	if selected {
		return indent + "▶ "
	}
	return indent + "  "
}

func (m *Model) todoMove(delta int) {
	cursors := m.todoCursors()
	if len(cursors) == 0 {
		m.selectedTodo = -1
		m.selectedSub = -1
		m.selectedSub2 = -1
		return
	}
	idx := 0
	for i, c := range cursors {
		if c.top == m.selectedTodo && c.sub == m.selectedSub && c.sub2 == m.selectedSub2 {
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
	m.selectedSub2 = cursors[idx].sub2
}

func (m Model) todoCursors() []todoCursor {
	var out []todoCursor
	for i, t := range m.workspaceTodos {
		out = append(out, todoCursor{top: i, sub: -1, sub2: -1})
		for j, st := range t.Subtodos {
			out = append(out, todoCursor{top: i, sub: j, sub2: -1})
			for k := range st.Subtodos {
				out = append(out, todoCursor{top: i, sub: j, sub2: k})
			}
		}
	}
	return out
}

func (m *Model) toggleSelectedTodo() bool {
	if m.selectedTodo < 0 || m.selectedTodo >= len(m.workspaceTodos) {
		return false
	}
	if m.selectedSub >= 0 && m.selectedSub2 >= 0 {
		if m.selectedSub >= len(m.workspaceTodos[m.selectedTodo].Subtodos) {
			return false
		}
		level2 := m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub]
		if m.selectedSub2 >= len(level2.Subtodos) {
			return false
		}
		m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos[m.selectedSub2].Completed = !m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Subtodos[m.selectedSub2].Completed
		return true
	}
	if m.selectedSub >= 0 {
		if m.selectedSub >= len(m.workspaceTodos[m.selectedTodo].Subtodos) {
			return false
		}
		m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Completed = !m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub].Completed
		return true
	}
	m.workspaceTodos[m.selectedTodo].Completed = !m.workspaceTodos[m.selectedTodo].Completed
	return true
}

func (m *Model) appendTodoDraft(s string) {
	m.todoDraft += s
}

func (m *Model) backspaceTodoDraft() {
	if m.todoDraft == "" {
		return
	}
	r := []rune(m.todoDraft)
	m.todoDraft = string(r[:len(r)-1])
}

func (m *Model) exitTodoInputMode() {
	m.todoInputMode = false
	m.todoDraft = ""
}

func (m *Model) commitTodoDraft() bool {
	title := strings.TrimSpace(m.todoDraft)
	if title == "" {
		return false
	}
	m.workspaceTodos = append(m.workspaceTodos, journal.Todo{
		ID:       journal.NewID(),
		Title:    title,
		Subtodos: []journal.Todo{},
	})
	m.selectedTodo = len(m.workspaceTodos) - 1
	m.selectedSub = -1
	m.selectedSub2 = -1
	m.todoDraft = ""
	return true
}

func (m *Model) indentSelectedTodo() bool {
	if m.selectedTodo < 0 || m.selectedTodo >= len(m.workspaceTodos) {
		return false
	}
	// Indent level-2 todo to level-3 under previous level-2 sibling.
	if m.selectedSub >= 0 && m.selectedSub2 == -1 {
		parent := m.workspaceTodos[m.selectedTodo]
		if m.selectedSub <= 0 || m.selectedSub >= len(parent.Subtodos) {
			return false
		}
		targetParentIdx := m.selectedSub - 1
		td := parent.Subtodos[m.selectedSub]
		parent.Subtodos = append(parent.Subtodos[:m.selectedSub], parent.Subtodos[m.selectedSub+1:]...)
		parent.Subtodos[targetParentIdx].Subtodos = append(parent.Subtodos[targetParentIdx].Subtodos, td)
		parent.Subtodos[targetParentIdx].Subtodos = clampTodoListAtDepth(parent.Subtodos[targetParentIdx].Subtodos, 2)
		m.workspaceTodos[m.selectedTodo] = parent
		m.selectedSub = targetParentIdx
		m.selectedSub2 = findTodoIndexByID(m.workspaceTodos[m.selectedTodo].Subtodos[targetParentIdx].Subtodos, td.ID)
		return true
	}
	// Already at max supported depth.
	if m.selectedSub >= 0 && m.selectedSub2 >= 0 {
		return false
	}
	// Indent top-level todo to level-2 under previous top-level sibling.
	if m.selectedTodo <= 0 {
		return false
	}
	parentIdx := m.selectedTodo - 1
	td := m.workspaceTodos[m.selectedTodo]
	m.workspaceTodos[parentIdx].Subtodos = append(m.workspaceTodos[parentIdx].Subtodos, td)
	m.workspaceTodos[parentIdx].Subtodos = clampTodoListAtDepth(m.workspaceTodos[parentIdx].Subtodos, 1)
	m.workspaceTodos = append(m.workspaceTodos[:m.selectedTodo], m.workspaceTodos[m.selectedTodo+1:]...)
	m.selectedTodo = parentIdx
	m.selectedSub = findTodoIndexByID(m.workspaceTodos[parentIdx].Subtodos, td.ID)
	m.selectedSub2 = -1
	return true
}

func clampTodoListAtDepth(items []journal.Todo, depth int) []journal.Todo {
	if depth >= 2 {
		out := make([]journal.Todo, 0, len(items))
		for _, item := range items {
			descendants := flattenTodos(item.Subtodos)
			item.Subtodos = nil
			out = append(out, item)
			out = append(out, descendants...)
		}
		return out
	}
	for i := range items {
		items[i].Subtodos = clampTodoListAtDepth(items[i].Subtodos, depth+1)
	}
	return items
}

func flattenTodos(items []journal.Todo) []journal.Todo {
	out := make([]journal.Todo, 0, len(items))
	var walk func(todo journal.Todo)
	walk = func(todo journal.Todo) {
		children := todo.Subtodos
		todo.Subtodos = nil
		out = append(out, todo)
		for _, child := range children {
			walk(child)
		}
	}
	for _, item := range items {
		walk(item)
	}
	return out
}

func findTodoIndexByID(items []journal.Todo, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func (m *Model) outdentSelectedTodo() bool {
	if m.selectedTodo < 0 || m.selectedTodo >= len(m.workspaceTodos) || m.selectedSub < 0 {
		return false
	}
	// Outdent level-3 todo to level-2.
	if m.selectedSub2 >= 0 {
		parent := m.workspaceTodos[m.selectedTodo]
		if m.selectedSub >= len(parent.Subtodos) {
			return false
		}
		level2 := parent.Subtodos[m.selectedSub]
		if m.selectedSub2 >= len(level2.Subtodos) {
			return false
		}
		td := level2.Subtodos[m.selectedSub2]
		level2.Subtodos = append(level2.Subtodos[:m.selectedSub2], level2.Subtodos[m.selectedSub2+1:]...)
		parent.Subtodos[m.selectedSub] = level2

		insertIdx := m.selectedSub + 1
		parent.Subtodos = append(parent.Subtodos, journal.Todo{})
		copy(parent.Subtodos[insertIdx+1:], parent.Subtodos[insertIdx:])
		parent.Subtodos[insertIdx] = td
		m.workspaceTodos[m.selectedTodo] = parent
		m.selectedSub = insertIdx
		m.selectedSub2 = -1
		return true
	}
	parentIdx := m.selectedTodo
	parent := m.workspaceTodos[parentIdx]
	if m.selectedSub >= len(parent.Subtodos) {
		return false
	}
	td := parent.Subtodos[m.selectedSub]
	parent.Subtodos = append(parent.Subtodos[:m.selectedSub], parent.Subtodos[m.selectedSub+1:]...)
	m.workspaceTodos[parentIdx].Subtodos = parent.Subtodos

	insertIdx := parentIdx + 1
	m.workspaceTodos = append(m.workspaceTodos, journal.Todo{})
	copy(m.workspaceTodos[insertIdx+1:], m.workspaceTodos[insertIdx:])
	m.workspaceTodos[insertIdx] = td
	m.selectedTodo = insertIdx
	m.selectedSub = -1
	m.selectedSub2 = -1
	return true
}

// isFullyCompleted reports whether t and every nested subtodo are all completed.
func isFullyCompleted(t journal.Todo) bool {
	if !t.Completed {
		return false
	}
	for _, sub := range t.Subtodos {
		if !isFullyCompleted(sub) {
			return false
		}
	}
	return true
}

// pruneCompletedTodos removes todos (at any depth) where the todo itself and
// all its descendants are completed. Partial branches (some children
// incomplete) are kept intact.
func pruneCompletedTodos(todos []journal.Todo) []journal.Todo {
	result := make([]journal.Todo, 0, len(todos))
	for _, t := range todos {
		if isFullyCompleted(t) {
			continue
		}
		t.Subtodos = pruneCompletedTodos(t.Subtodos)
		result = append(result, t)
	}
	return result
}

// moveSelectedTodoDelta swaps the currently selected todo with its adjacent
// sibling in the direction indicated by delta (+1 = down, -1 = up). The
// selection cursor is updated to follow the moved item. Returns true when a
// swap was performed.
func (m *Model) moveSelectedTodoDelta(delta int) bool {
	if m.selectedTodo < 0 || m.selectedTodo >= len(m.workspaceTodos) {
		return false
	}
	// Level 3
	if m.selectedSub >= 0 && m.selectedSub2 >= 0 {
		if m.selectedSub >= len(m.workspaceTodos[m.selectedTodo].Subtodos) {
			return false
		}
		sub := &m.workspaceTodos[m.selectedTodo].Subtodos[m.selectedSub]
		newIdx := m.selectedSub2 + delta
		if newIdx < 0 || newIdx >= len(sub.Subtodos) {
			return false
		}
		sub.Subtodos[m.selectedSub2], sub.Subtodos[newIdx] = sub.Subtodos[newIdx], sub.Subtodos[m.selectedSub2]
		m.selectedSub2 = newIdx
		return true
	}
	// Level 2
	if m.selectedSub >= 0 {
		parent := &m.workspaceTodos[m.selectedTodo]
		newIdx := m.selectedSub + delta
		if newIdx < 0 || newIdx >= len(parent.Subtodos) {
			return false
		}
		parent.Subtodos[m.selectedSub], parent.Subtodos[newIdx] = parent.Subtodos[newIdx], parent.Subtodos[m.selectedSub]
		m.selectedSub = newIdx
		return true
	}
	// Level 1 (top-level)
	newIdx := m.selectedTodo + delta
	if newIdx < 0 || newIdx >= len(m.workspaceTodos) {
		return false
	}
	m.workspaceTodos[m.selectedTodo], m.workspaceTodos[newIdx] = m.workspaceTodos[newIdx], m.workspaceTodos[m.selectedTodo]
	m.selectedTodo = newIdx
	return true
}

func (m *Model) deleteSelectedTodoNow() bool {
	if m.selectedTodo < 0 || m.selectedTodo >= len(m.workspaceTodos) {
		return false
	}
	if m.selectedSub >= 0 && m.selectedSub2 >= 0 {
		level2 := m.workspaceTodos[m.selectedTodo].Subtodos
		if m.selectedSub >= len(level2) {
			return false
		}
		level3 := level2[m.selectedSub].Subtodos
		if m.selectedSub2 >= len(level3) {
			return false
		}
		level2[m.selectedSub].Subtodos = append(level3[:m.selectedSub2], level3[m.selectedSub2+1:]...)
		m.workspaceTodos[m.selectedTodo].Subtodos = level2
		m.selectedSub2 = -1
		return true
	}
	if m.selectedSub >= 0 {
		st := m.workspaceTodos[m.selectedTodo].Subtodos
		if m.selectedSub >= len(st) {
			return false
		}
		m.workspaceTodos[m.selectedTodo].Subtodos = append(st[:m.selectedSub], st[m.selectedSub+1:]...)
		m.selectedSub = -1
		m.selectedSub2 = -1
		return true
	}
	m.workspaceTodos = append(m.workspaceTodos[:m.selectedTodo], m.workspaceTodos[m.selectedTodo+1:]...)
	if len(m.workspaceTodos) == 0 {
		m.selectedTodo = -1
		m.selectedSub = -1
		m.selectedSub2 = -1
		return true
	}
	if m.selectedTodo >= len(m.workspaceTodos) {
		m.selectedTodo = len(m.workspaceTodos) - 1
	}
	m.selectedSub = -1
	m.selectedSub2 = -1
	return true
}

func (m Model) renderTodosPanel(w int) string {
	var b strings.Builder
	b.WriteString(dayViewSectionStyle.Render("✅  Todos") + "\n")
	b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
	if m.selectedPane == 1 {
		draft := strings.TrimSpace(m.todoDraft)
		if draft == "" {
			hint := dayViewMutedStyle.Render("  type to add, enter to save")
			if m.todoInputMode {
				hint = todoInputActiveStyle.Render("  type to add, enter to save")
			}
			b.WriteString(hint + "\n")
		} else {
			draftLine := dayViewValueStyle.Render("  + " + m.todoDraft)
			if m.todoInputMode {
				draftLine = todoInputActiveStyle.Render("  + " + m.todoDraft)
			}
			b.WriteString(draftLine + "\n")
		}
		b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
	}
	if len(m.workspaceTodos) == 0 {
		b.WriteString(dayViewMutedStyle.Render("  No todos yet") + "\n")
		if m.selectedPane != 1 {
			b.WriteString(dayViewMutedStyle.Render("  t open todo overview") + "\n")
		}
		return b.String()
	}
	for i, td := range m.workspaceTodos {
		mark := todoIncompleteStyle.Render("—")
		if td.Completed {
			mark = todoCompleteStyle.Render("✓")
		}
		prefix := todoLinePrefix(0, m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == -1)
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
			sprefix := todoLinePrefix(1, m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == j && m.selectedSub2 == -1)
			sline := sprefix + smark + " " + st.Title
			if lipgloss.Width(sline) > w {
				sline = truncateRunes(sline, w)
			}
			if m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == j && m.selectedSub2 == -1 {
				sline = selectedEntryStyle.Render(sline)
			}
			b.WriteString(sline + "\n")

			for k, thirdLevelTodo := range st.Subtodos {
				ssmark := todoIncompleteStyle.Render("—")
				if thirdLevelTodo.Completed {
					ssmark = todoCompleteStyle.Render("✓")
				}
				thirdLevelPrefix := todoLinePrefix(2, m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == j && m.selectedSub2 == k)
				thirdLevelLine := thirdLevelPrefix + ssmark + " " + thirdLevelTodo.Title
				if lipgloss.Width(thirdLevelLine) > w {
					thirdLevelLine = truncateRunes(thirdLevelLine, w)
				}
				if m.selectedPane == 1 && m.selectedTodo == i && m.selectedSub == j && m.selectedSub2 == k {
					thirdLevelLine = selectedEntryStyle.Render(thirdLevelLine)
				}
				b.WriteString(thirdLevelLine + "\n")
			}
		}
	}
	return b.String()
}
