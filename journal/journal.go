package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const workspaceTodosFile = "todos.json"

func normalizeTodos(todos []Todo) []Todo {
	if todos == nil {
		return []Todo{}
	}
	out := make([]Todo, len(todos))
	for i, t := range todos {
		t.Subtodos = normalizeTodos(t.Subtodos)
		out[i] = t
	}
	return out
}

func normalizeWorkspaceTodos(w WorkspaceTodos) WorkspaceTodos {
	w.Todos = normalizeTodos(w.Todos)
	return w
}

// storagePath overrides the default ~/.journal directory when set via SetStoragePath.
var storagePath string

// SetStoragePath overrides the default ~/.journal storage directory. The path
// may contain a leading ~ which is expanded to the user's home directory.
func SetStoragePath(path string) error {
	if path == "" {
		storagePath = ""
		return nil
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		// TrimPrefix strips a leading "/" so that filepath.Join does not
		// interpret the remainder as an absolute path (e.g. "~/.journal" →
		// "<home>/.journal" rather than "/.journal").
		rest := strings.TrimPrefix(path[1:], "/")
		if rest == "" {
			path = home
		} else {
			path = filepath.Join(home, rest)
		}
	}
	storagePath = path
	return nil
}

// Dir returns (and creates if necessary) the journal storage directory.
// Defaults to ~/.journal; can be overridden with SetStoragePath.
func Dir() (string, error) {
	if storagePath != "" {
		return storagePath, os.MkdirAll(storagePath, 0o755)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".journal")
	return dir, os.MkdirAll(dir, 0o755)
}

// TodayPath returns the file path for today's .json record.
func TodayPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, time.Now().Format("2006-01-02")+".json"), nil
}

// PathForDate returns the file path for a given date string (YYYY-MM-DD).
func PathForDate(date string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, date+".json"), nil
}

// WorkspaceTodosPath returns the file path for workspace-level todos.
func WorkspaceTodosPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, workspaceTodosFile), nil
}

// LoadWorkspaceTodos reads workspace-level todos. Missing file returns empty todos.
func LoadWorkspaceTodos() (WorkspaceTodos, error) {
	path, err := WorkspaceTodosPath()
	if err != nil {
		return WorkspaceTodos{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return WorkspaceTodos{Todos: []Todo{}}, nil
	}
	if err != nil {
		return WorkspaceTodos{}, err
	}
	var todos WorkspaceTodos
	if err := json.Unmarshal(data, &todos); err != nil {
		return WorkspaceTodos{}, err
	}
	return normalizeWorkspaceTodos(todos), nil
}

// SaveWorkspaceTodos persists workspace-level todos to disk.
func SaveWorkspaceTodos(todos WorkspaceTodos) error {
	path, err := WorkspaceTodosPath()
	if err != nil {
		return err
	}
	todos = normalizeWorkspaceTodos(todos)
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads a DayRecord from path. If the file does not exist, it returns an
// empty DayRecord (not an error).
func Load(path string) (DayRecord, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		dateStr := filepath.Base(path)
		if len(dateStr) >= 10 {
			dateStr = dateStr[:10]
		}
		return DayRecord{Date: dateStr, Path: path, Todos: []Todo{}}, nil
	}
	if err != nil {
		return DayRecord{}, err
	}
	var rec DayRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return DayRecord{}, err
	}
	rec.Todos = []Todo{}
	rec.Path = path
	return rec, nil
}

// Save writes rec to disk as JSON. If rec.Path is empty it is derived from rec.Date.
func Save(rec DayRecord) error {
	if rec.Path == "" {
		dir, err := Dir()
		if err != nil {
			return err
		}
		rec.Path = filepath.Join(dir, rec.Date+".json")
	}
	rec.Todos = nil
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rec.Path, data, 0o644)
}

// LoadAll loads every DayRecord from the journal directory, sorted newest first.
func LoadAll() ([]DayRecord, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, "????-??-??.json"))
	if err != nil {
		return nil, err
	}
	var records []DayRecord
	for _, f := range files {
		rec, err := Load(f)
		if err != nil {
			continue
		}
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Date > records[j].Date
	})
	return records, nil
}

// Delete removes a record file.
func Delete(path string) error {
	return os.Remove(path)
}

// NewID returns a unique string ID based on the current nanosecond timestamp.
func NewID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
