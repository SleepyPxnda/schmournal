package journal

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry represents a single journal entry stored as a markdown file.
type Entry struct {
	Date    time.Time
	Title   string
	Content string
	Path    string
}

// Dir returns (and creates if necessary) the ~/.journal directory.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".journal")
	return dir, os.MkdirAll(dir, 0o755)
}

// TodayPath returns the file path for today's entry.
func TodayPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, time.Now().Format("2006-01-02")+".md"), nil
}

// NewEntryContent returns the daily template for a new entry.
func NewEntryContent(t time.Time) string {
	return NewEntryTemplate(t)
}

// LoadAll loads every entry from the journal directory, sorted newest first.
func LoadAll() ([]Entry, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, f := range files {
		e, err := Load(f)
		if err != nil {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.After(entries[j].Date)
	})
	return entries, nil
}

// Load reads a single entry from disk.
func Load(path string) (Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, err
	}
	dateStr := strings.TrimSuffix(filepath.Base(path), ".md")
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return Entry{}, err
	}
	content := string(data)
	return Entry{
		Date:    t,
		Title:   extractTitle(content, t),
		Content: content,
		Path:    path,
	}, nil
}

// Save writes content to path, creating or overwriting the file.
func Save(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// Delete removes an entry file.
func Delete(path string) error {
	return os.Remove(path)
}

func extractTitle(content string, t time.Time) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
		if line != "" {
			if len(line) > 50 {
				return line[:50] + "…"
			}
			return line
		}
	}
	return t.Format("January 2, 2006")
}
