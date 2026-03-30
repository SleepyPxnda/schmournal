package tui

import (
	"strconv"
	"time"

	"github.com/sleepypxnda/schmournal/internal/application/usecase"
	domainmodel "github.com/sleepypxnda/schmournal/internal/domain/model"
)

type WorkEntry = domainmodel.WorkEntry
type Todo = domainmodel.Todo
type WorkspaceTodos = domainmodel.WorkspaceTodos

// DayRecord is a UI-facing day aggregate that keeps a runtime-only Path field
// while delegating business behavior to the domain entity.
type DayRecord struct {
	Date      string
	StartTime string
	EndTime   string
	Entries   []WorkEntry
	Notes     string
	TodayDone []Todo
	Path      string
}

func (r DayRecord) toDomain() domainmodel.DayRecord {
	return domainmodel.DayRecord{
		Date:      r.Date,
		StartTime: r.StartTime,
		EndTime:   r.EndTime,
		Entries:   append([]WorkEntry(nil), r.Entries...),
		Notes:     r.Notes,
		TodayDone: append([]Todo(nil), r.TodayDone...),
	}
}

func dayRecordFromDomain(r domainmodel.DayRecord) DayRecord {
	return DayRecord{
		Date:      r.Date,
		StartTime: r.StartTime,
		EndTime:   r.EndTime,
		Entries:   append([]WorkEntry(nil), r.Entries...),
		Notes:     r.Notes,
		TodayDone: append([]Todo(nil), r.TodayDone...),
	}
}

func (r DayRecord) ParseDate() (time.Time, error) {
	return r.toDomain().ParseDate()
}

func (r DayRecord) WorkTotals() (work, breaks, total time.Duration) {
	return r.toDomain().WorkTotals()
}

func (r DayRecord) DayDuration() (time.Duration, bool) {
	return r.toDomain().DayDuration()
}

func (r DayRecord) Summary() string {
	return r.toDomain().Summary(uiDurationFormatter)
}

func toUIDayRecords(records []usecase.DayRecordDTO) []DayRecord {
	out := make([]DayRecord, len(records))
	for i, rec := range records {
		out[i] = toUIDayRecord(rec)
	}
	return out
}

func toUIDayRecord(rec usecase.DayRecordDTO) DayRecord {
	entries := make([]WorkEntry, len(rec.Entries))
	for i, entry := range rec.Entries {
		entries[i] = WorkEntry{
			ID:          entry.ID,
			Project:     entry.Project,
			Task:        entry.Task,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		}
	}

	return DayRecord{
		Date:      rec.Date,
		StartTime: rec.StartTime,
		EndTime:   rec.EndTime,
		Entries:   entries,
		Notes:     rec.Notes,
		TodayDone: toUITodos(rec.TodayDone),
	}
}

func toUseCaseDayRecord(rec DayRecord) usecase.DayRecordDTO {
	entries := make([]usecase.WorkEntryDTO, len(rec.Entries))
	for i, entry := range rec.Entries {
		entries[i] = usecase.WorkEntryDTO{
			ID:          entry.ID,
			Project:     entry.Project,
			Task:        entry.Task,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		}
	}

	return usecase.DayRecordDTO{
		Date:      rec.Date,
		StartTime: rec.StartTime,
		EndTime:   rec.EndTime,
		Entries:   entries,
		Notes:     rec.Notes,
		TodayDone: toUseCaseTodos(rec.TodayDone),
	}
}

func toUIWorkspaceTodos(todos usecase.WorkspaceTodosDTO) WorkspaceTodos {
	return WorkspaceTodos{
		Todos: toUITodos(todos.Todos),
	}
}

func toUITodos(todos []usecase.TodoDTO) []Todo {
	out := make([]Todo, len(todos))
	for i, todo := range todos {
		out[i] = Todo{
			ID:        todo.ID,
			Title:     todo.Title,
			Completed: todo.Completed,
			Subtodos:  toUITodos(todo.Subtodos),
		}
	}
	return out
}

func toUseCaseWorkspaceTodos(todos WorkspaceTodos) usecase.WorkspaceTodosDTO {
	return usecase.WorkspaceTodosDTO{
		Todos: toUseCaseTodos(todos.Todos),
	}
}

func toUseCaseTodos(todos []Todo) []usecase.TodoDTO {
	out := make([]usecase.TodoDTO, len(todos))
	for i, todo := range todos {
		out[i] = usecase.TodoDTO{
			ID:        todo.ID,
			Title:     todo.Title,
			Completed: todo.Completed,
			Subtodos:  toUseCaseTodos(todo.Subtodos),
		}
	}
	return out
}

func newID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
