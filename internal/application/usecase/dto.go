package usecase

import "github.com/sleepypxnda/schmournal/internal/domain/model"

type WorkEntryDTO struct {
	ID          string
	Project     string
	Task        string
	DurationMin int
	IsBreak     bool
}

type DayRecordDTO struct {
	Date      string
	StartTime string
	EndTime   string
	Entries   []WorkEntryDTO
	Notes     string
	TodayDone []TodoDTO
}

type TodoDTO struct {
	ID        string
	Title     string
	Completed bool
	Subtodos  []TodoDTO
}

type WorkspaceTodosDTO struct {
	Todos    []TodoDTO
	Archived []TodoDTO
}

func mapDomainDayRecordToDTO(rec model.DayRecord) DayRecordDTO {
	entries := make([]WorkEntryDTO, len(rec.Entries))
	for i, entry := range rec.Entries {
		entries[i] = WorkEntryDTO{
			ID:          entry.ID,
			Project:     entry.Project,
			Task:        entry.Task,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		}
	}
	return DayRecordDTO{
		Date:      rec.Date,
		StartTime: rec.StartTime,
		EndTime:   rec.EndTime,
		Entries:   entries,
		Notes:     rec.Notes,
		TodayDone: mapDomainTodosToDTO(rec.TodayDone),
	}
}

func mapDayRecordDTOToDomain(rec DayRecordDTO) model.DayRecord {
	entries := make([]model.WorkEntry, len(rec.Entries))
	for i, entry := range rec.Entries {
		entries[i] = model.WorkEntry{
			ID:          entry.ID,
			Project:     entry.Project,
			Task:        entry.Task,
			DurationMin: entry.DurationMin,
			IsBreak:     entry.IsBreak,
		}
	}
	return model.DayRecord{
		Date:      rec.Date,
		StartTime: rec.StartTime,
		EndTime:   rec.EndTime,
		Entries:   entries,
		Notes:     rec.Notes,
		TodayDone: mapTodosDTOToDomain(rec.TodayDone),
	}
}

func mapDomainWorkspaceTodosToDTO(todos model.WorkspaceTodos) WorkspaceTodosDTO {
	return WorkspaceTodosDTO{
		Todos:    mapDomainTodosToDTO(todos.Todos),
		Archived: mapDomainTodosToDTO(todos.Archived),
	}
}

func mapWorkspaceTodosDTOToDomain(todos WorkspaceTodosDTO) model.WorkspaceTodos {
	return model.WorkspaceTodos{
		Todos:    mapTodosDTOToDomain(todos.Todos),
		Archived: mapTodosDTOToDomain(todos.Archived),
	}
}

func mapDomainTodosToDTO(todos []model.Todo) []TodoDTO {
	out := make([]TodoDTO, len(todos))
	for i, todo := range todos {
		out[i] = TodoDTO{
			ID:        todo.ID,
			Title:     todo.Title,
			Completed: todo.Completed,
			Subtodos:  mapDomainTodosToDTO(todo.Subtodos),
		}
	}
	return out
}

func mapTodosDTOToDomain(todos []TodoDTO) []model.Todo {
	out := make([]model.Todo, len(todos))
	for i, todo := range todos {
		out[i] = model.Todo{
			ID:        todo.ID,
			Title:     todo.Title,
			Completed: todo.Completed,
			Subtodos:  mapTodosDTOToDomain(todo.Subtodos),
		}
	}
	return out
}
