package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
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
		m.todoSelection.Top = -1
		m.todoSelection.Sub = -1
		m.todoSelection.Sub2 = -1
		return
	}
	idx := 0
	for i, c := range cursors {
		if c.top == m.todoSelection.Top && c.sub == m.todoSelection.Sub && c.sub2 == m.todoSelection.Sub2 {
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
	m.todoSelection.Top = cursors[idx].top
	m.todoSelection.Sub = cursors[idx].sub
	m.todoSelection.Sub2 = cursors[idx].sub2
}

func (m Model) todoCursors() []todoCursor {
	var out []todoCursor
	for i, t := range m.workspace.Todos {
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
	if m.todoSelection.Top < 0 || m.todoSelection.Top >= len(m.workspace.Todos) {
		return false
	}
	if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 >= 0 {
		if m.todoSelection.Sub >= len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
			return false
		}
		level2 := m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub]
		if m.todoSelection.Sub2 >= len(level2.Subtodos) {
			return false
		}
		m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos[m.todoSelection.Sub2].Completed = !m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Subtodos[m.todoSelection.Sub2].Completed
		return true
	}
	if m.todoSelection.Sub >= 0 {
		if m.todoSelection.Sub >= len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
			return false
		}
		m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Completed = !m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub].Completed
		return true
	}
	m.workspace.Todos[m.todoSelection.Top].Completed = !m.workspace.Todos[m.todoSelection.Top].Completed
	return true
}

func (m *Model) appendTodoDraft(s string) {
	m.todoEditor.Draft += s
}

func (m *Model) backspaceTodoDraft() {
	if m.todoEditor.Draft == "" {
		return
	}
	r := []rune(m.todoEditor.Draft)
	m.todoEditor.Draft = string(r[:len(r)-1])
}

func (m *Model) exitTodoInputMode() {
	m.todoEditor.InputMode = false
	m.todoEditor.Draft = ""
}

func (m *Model) commitTodoDraft() bool {
	title := strings.TrimSpace(m.todoEditor.Draft)
	if title == "" {
		return false
	}
	m.workspace.Todos = append(m.workspace.Todos, Todo{
		ID:       newID(),
		Title:    title,
		Subtodos: []Todo{},
	})
	m.todoSelection.Top = len(m.workspace.Todos) - 1
	m.todoSelection.Sub = -1
	m.todoSelection.Sub2 = -1
	m.todoEditor.Draft = ""
	return true
}

func (m *Model) indentSelectedTodo() bool {
	if m.todoSelection.Top < 0 || m.todoSelection.Top >= len(m.workspace.Todos) {
		return false
	}
	// Indent level-2 todo to level-3 under previous level-2 sibling.
	if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 == -1 {
		parent := m.workspace.Todos[m.todoSelection.Top]
		if m.todoSelection.Sub <= 0 || m.todoSelection.Sub >= len(parent.Subtodos) {
			return false
		}
		targetParentIdx := m.todoSelection.Sub - 1
		td := parent.Subtodos[m.todoSelection.Sub]
		parent.Subtodos = append(parent.Subtodos[:m.todoSelection.Sub], parent.Subtodos[m.todoSelection.Sub+1:]...)
		parent.Subtodos[targetParentIdx].Subtodos = append(parent.Subtodos[targetParentIdx].Subtodos, td)
		parent.Subtodos[targetParentIdx].Subtodos = clampTodoListAtDepth(parent.Subtodos[targetParentIdx].Subtodos, 2)
		m.workspace.Todos[m.todoSelection.Top] = parent
		m.todoSelection.Sub = targetParentIdx
		m.todoSelection.Sub2 = findTodoIndexByID(m.workspace.Todos[m.todoSelection.Top].Subtodos[targetParentIdx].Subtodos, td.ID)
		return true
	}
	// Already at max supported depth.
	if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 >= 0 {
		return false
	}
	// Indent top-level todo to level-2 under previous top-level sibling.
	if m.todoSelection.Top <= 0 {
		return false
	}
	parentIdx := m.todoSelection.Top - 1
	td := m.workspace.Todos[m.todoSelection.Top]
	m.workspace.Todos[parentIdx].Subtodos = append(m.workspace.Todos[parentIdx].Subtodos, td)
	m.workspace.Todos[parentIdx].Subtodos = clampTodoListAtDepth(m.workspace.Todos[parentIdx].Subtodos, 1)
	m.workspace.Todos = append(m.workspace.Todos[:m.todoSelection.Top], m.workspace.Todos[m.todoSelection.Top+1:]...)
	m.todoSelection.Top = parentIdx
	m.todoSelection.Sub = findTodoIndexByID(m.workspace.Todos[parentIdx].Subtodos, td.ID)
	m.todoSelection.Sub2 = -1
	return true
}

func clampTodoListAtDepth(items []Todo, depth int) []Todo {
	if depth >= 2 {
		out := make([]Todo, 0, len(items))
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

func flattenTodos(items []Todo) []Todo {
	out := make([]Todo, 0, len(items))
	var walk func(todo Todo)
	walk = func(todo Todo) {
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

// mergeArchivedTodoTrees recursively merges incoming archived trees into existing
// trees by ID. Matching nodes (and descendants) are merged, non-matching nodes
// are appended, and incomplete context nodes are upgraded when a completed
// version of the same TODO arrives.
func mergeArchivedTodoTrees(existing []Todo, incoming []Todo) []Todo {
	merged := append([]Todo(nil), existing...)
	for _, in := range incoming {
		merged = mergeOrAppendTodo(merged, in)
	}
	return merged
}

func mergeOrAppendTodo(items []Todo, in Todo) []Todo {
	for i := range items {
		if items[i].ID != in.ID {
			continue
		}
		items[i] = mergeTodoNode(items[i], in)
		return items
	}
	return append(items, in)
}

// mergeTodoNode merges two TODO nodes that share the same ID.
// Completion is promoted (true if either node is complete) and subtodos are
// recursively merged by ID. Existing node title/other metadata are preserved.
func mergeTodoNode(existing Todo, incoming Todo) Todo {
	existing.Completed = existing.Completed || incoming.Completed
	for _, inSub := range incoming.Subtodos {
		existing.Subtodos = mergeOrAppendTodo(existing.Subtodos, inSub)
	}
	return existing
}

func findTodoIndexByID(items []Todo, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func (m *Model) outdentSelectedTodo() bool {
	if m.todoSelection.Top < 0 || m.todoSelection.Top >= len(m.workspace.Todos) || m.todoSelection.Sub < 0 {
		return false
	}
	// Outdent level-3 todo to level-2.
	if m.todoSelection.Sub2 >= 0 {
		parent := m.workspace.Todos[m.todoSelection.Top]
		if m.todoSelection.Sub >= len(parent.Subtodos) {
			return false
		}
		level2 := parent.Subtodos[m.todoSelection.Sub]
		if m.todoSelection.Sub2 >= len(level2.Subtodos) {
			return false
		}
		td := level2.Subtodos[m.todoSelection.Sub2]
		level2.Subtodos = append(level2.Subtodos[:m.todoSelection.Sub2], level2.Subtodos[m.todoSelection.Sub2+1:]...)
		parent.Subtodos[m.todoSelection.Sub] = level2

		insertIdx := m.todoSelection.Sub + 1
		parent.Subtodos = append(parent.Subtodos, Todo{})
		copy(parent.Subtodos[insertIdx+1:], parent.Subtodos[insertIdx:])
		parent.Subtodos[insertIdx] = td
		m.workspace.Todos[m.todoSelection.Top] = parent
		m.todoSelection.Sub = insertIdx
		m.todoSelection.Sub2 = -1
		return true
	}
	parentIdx := m.todoSelection.Top
	parent := m.workspace.Todos[parentIdx]
	if m.todoSelection.Sub >= len(parent.Subtodos) {
		return false
	}
	td := parent.Subtodos[m.todoSelection.Sub]
	parent.Subtodos = append(parent.Subtodos[:m.todoSelection.Sub], parent.Subtodos[m.todoSelection.Sub+1:]...)
	m.workspace.Todos[parentIdx].Subtodos = parent.Subtodos

	insertIdx := parentIdx + 1
	m.workspace.Todos = append(m.workspace.Todos, Todo{})
	copy(m.workspace.Todos[insertIdx+1:], m.workspace.Todos[insertIdx:])
	m.workspace.Todos[insertIdx] = td
	m.todoSelection.Top = insertIdx
	m.todoSelection.Sub = -1
	m.todoSelection.Sub2 = -1
	return true
}

// isFullyCompleted reports whether t and every nested subtodo are all completed.
func isFullyCompleted(t Todo) bool {
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

// collectFullyCompleted returns all top-level todos (and their subtree) for
// which isFullyCompleted is true. These are the items that will be moved to
// the archive when the user leaves the day view.
func collectFullyCompleted(todos []Todo) []Todo {
	var result []Todo
	for _, t := range todos {
		if isFullyCompleted(t) {
			result = append(result, t)
		}
	}
	return result
}

// pruneCompletedTodos removes todos (at any depth) where the todo itself and
// all its descendants are completed. Partial branches (some children
// incomplete) are kept intact.
func pruneCompletedTodos(todos []Todo) []Todo {
	result := make([]Todo, 0, len(todos))
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
	if m.todoSelection.Top < 0 || m.todoSelection.Top >= len(m.workspace.Todos) {
		return false
	}
	// Level 3
	if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 >= 0 {
		if m.todoSelection.Sub >= len(m.workspace.Todos[m.todoSelection.Top].Subtodos) {
			return false
		}
		sub := &m.workspace.Todos[m.todoSelection.Top].Subtodos[m.todoSelection.Sub]
		newIdx := m.todoSelection.Sub2 + delta
		if newIdx < 0 || newIdx >= len(sub.Subtodos) {
			return false
		}
		sub.Subtodos[m.todoSelection.Sub2], sub.Subtodos[newIdx] = sub.Subtodos[newIdx], sub.Subtodos[m.todoSelection.Sub2]
		m.todoSelection.Sub2 = newIdx
		return true
	}
	// Level 2
	if m.todoSelection.Sub >= 0 {
		parent := &m.workspace.Todos[m.todoSelection.Top]
		newIdx := m.todoSelection.Sub + delta
		if newIdx < 0 || newIdx >= len(parent.Subtodos) {
			return false
		}
		parent.Subtodos[m.todoSelection.Sub], parent.Subtodos[newIdx] = parent.Subtodos[newIdx], parent.Subtodos[m.todoSelection.Sub]
		m.todoSelection.Sub = newIdx
		return true
	}
	// Level 1 (top-level)
	newIdx := m.todoSelection.Top + delta
	if newIdx < 0 || newIdx >= len(m.workspace.Todos) {
		return false
	}
	m.workspace.Todos[m.todoSelection.Top], m.workspace.Todos[newIdx] = m.workspace.Todos[newIdx], m.workspace.Todos[m.todoSelection.Top]
	m.todoSelection.Top = newIdx
	return true
}

func (m *Model) deleteSelectedTodoNow() bool {
	if m.todoSelection.Top < 0 || m.todoSelection.Top >= len(m.workspace.Todos) {
		return false
	}
	if m.todoSelection.Sub >= 0 && m.todoSelection.Sub2 >= 0 {
		level2 := m.workspace.Todos[m.todoSelection.Top].Subtodos
		if m.todoSelection.Sub >= len(level2) {
			return false
		}
		level3 := level2[m.todoSelection.Sub].Subtodos
		if m.todoSelection.Sub2 >= len(level3) {
			return false
		}
		level2[m.todoSelection.Sub].Subtodos = append(level3[:m.todoSelection.Sub2], level3[m.todoSelection.Sub2+1:]...)
		m.workspace.Todos[m.todoSelection.Top].Subtodos = level2
		m.todoSelection.Sub2 = -1
		return true
	}
	if m.todoSelection.Sub >= 0 {
		st := m.workspace.Todos[m.todoSelection.Top].Subtodos
		if m.todoSelection.Sub >= len(st) {
			return false
		}
		m.workspace.Todos[m.todoSelection.Top].Subtodos = append(st[:m.todoSelection.Sub], st[m.todoSelection.Sub+1:]...)
		m.todoSelection.Sub = -1
		m.todoSelection.Sub2 = -1
		return true
	}
	m.workspace.Todos = append(m.workspace.Todos[:m.todoSelection.Top], m.workspace.Todos[m.todoSelection.Top+1:]...)
	if len(m.workspace.Todos) == 0 {
		m.todoSelection.Top = -1
		m.todoSelection.Sub = -1
		m.todoSelection.Sub2 = -1
		return true
	}
	if m.todoSelection.Top >= len(m.workspace.Todos) {
		m.todoSelection.Top = len(m.workspace.Todos) - 1
	}
	m.todoSelection.Sub = -1
	m.todoSelection.Sub2 = -1
	return true
}

func (m Model) renderTodosPanel(w int) string {
	var b strings.Builder
	b.WriteString(dayViewSectionStyle.Render("✅  Todos") + "\n")
	b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
	if m.day.Selection.Pane == 1 {
		draft := strings.TrimSpace(m.todoEditor.Draft)
		if draft == "" {
			hint := dayViewMutedStyle.Render("  type to add, enter to save")
			if m.todoEditor.InputMode {
				hint = todoInputActiveStyle.Render("  type to add, enter to save")
			}
			b.WriteString(hint + "\n")
		} else {
			draftLine := dayViewValueStyle.Render("  + " + m.todoEditor.Draft)
			if m.todoEditor.InputMode {
				draftLine = todoInputActiveStyle.Render("  + " + m.todoEditor.Draft)
			}
			b.WriteString(draftLine + "\n")
		}
		b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
	}
	if len(m.workspace.Todos) == 0 {
		b.WriteString(dayViewMutedStyle.Render("  No todos yet") + "\n")
		if m.day.Selection.Pane != 1 {
			b.WriteString(dayViewMutedStyle.Render("  t open todo overview") + "\n")
		}
		if len(m.workspace.Archived) == 0 && len(m.day.Record.TodayDone) == 0 {
			return b.String()
		}
	}
	for i, td := range m.workspace.Todos {
		mark := todoIncompleteStyle.Render("—")
		if td.Completed {
			mark = todoCompleteStyle.Render("✓")
		}
		prefix := todoLinePrefix(0, m.day.Selection.Pane == 1 && m.todoSelection.Top == i && m.todoSelection.Sub == -1)
		line := prefix + mark + " " + td.Title
		if lipgloss.Width(line) > w {
			line = truncateRunes(line, w)
		}
		if m.day.Selection.Pane == 1 && m.todoSelection.Top == i && m.todoSelection.Sub == -1 {
			line = selectedEntryStyle.Render(line)
		}
		b.WriteString(line + "\n")

		for j, st := range td.Subtodos {
			smark := todoIncompleteStyle.Render("—")
			if st.Completed {
				smark = todoCompleteStyle.Render("✓")
			}
			sprefix := todoLinePrefix(1, m.day.Selection.Pane == 1 && m.todoSelection.Top == i && m.todoSelection.Sub == j && m.todoSelection.Sub2 == -1)
			sline := sprefix + smark + " " + st.Title
			if lipgloss.Width(sline) > w {
				sline = truncateRunes(sline, w)
			}
			if m.day.Selection.Pane == 1 && m.todoSelection.Top == i && m.todoSelection.Sub == j && m.todoSelection.Sub2 == -1 {
				sline = selectedEntryStyle.Render(sline)
			}
			b.WriteString(sline + "\n")

			for k, thirdLevelTodo := range st.Subtodos {
				ssmark := todoIncompleteStyle.Render("—")
				if thirdLevelTodo.Completed {
					ssmark = todoCompleteStyle.Render("✓")
				}
				thirdLevelPrefix := todoLinePrefix(2, m.day.Selection.Pane == 1 && m.todoSelection.Top == i && m.todoSelection.Sub == j && m.todoSelection.Sub2 == k)
				thirdLevelLine := thirdLevelPrefix + ssmark + " " + thirdLevelTodo.Title
				if lipgloss.Width(thirdLevelLine) > w {
					thirdLevelLine = truncateRunes(thirdLevelLine, w)
				}
				if m.day.Selection.Pane == 1 && m.todoSelection.Top == i && m.todoSelection.Sub == j && m.todoSelection.Sub2 == k {
					thirdLevelLine = selectedEntryStyle.Render(thirdLevelLine)
				}
				b.WriteString(thirdLevelLine + "\n")
			}
		}
	}
	if len(m.day.Record.TodayDone) > 0 {
		b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
		b.WriteString(dayViewMutedStyle.Render("  Today Done") + "\n")
		for _, td := range m.day.Record.TodayDone {
			b.WriteString(renderArchivedTodoTree(td, 0, w))
		}
	}
	if len(m.workspace.Archived) > 0 {
		b.WriteString(dayViewDividerStyle.Render(strings.Repeat("─", w)) + "\n")
		b.WriteString(dayViewMutedStyle.Render("  Archived") + "\n")
		for _, td := range m.workspace.Archived {
			b.WriteString(renderArchivedTodoTree(td, 0, w))
		}
		if m.day.Selection.Pane == 1 {
			b.WriteString(dayViewMutedStyle.Render("  X clear archive") + "\n")
		}
	}
	return b.String()
}

// renderArchivedTodoTree renders a single archived todo (and its subtree) at the
// given indentation depth, capped at the available width w.
func renderArchivedTodoTree(td Todo, depth int, w int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)
	mark := "✓"
	style := todoArchivedStyle
	if !td.Completed {
		mark = "-"
		style = dayViewMutedStyle
	}
	line := style.Render(indent + "  " + mark + " " + td.Title)
	if lipgloss.Width(line) > w {
		line = truncateRunes(line, w)
	}
	b.WriteString(line + "\n")
	for _, sub := range td.Subtodos {
		b.WriteString(renderArchivedTodoTree(sub, depth+1, w))
	}
	return b.String()
}
