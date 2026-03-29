package config

import (
	"reflect"
	"testing"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
)

func TestCollectTOMLPathsCoversAllLeafs(t *testing.T) {
	paths := collectTOMLPaths(reflect.TypeOf(model.AppConfig{}), nil)
	if len(paths) == 0 {
		t.Fatal("collectTOMLPaths returned no paths")
	}
	for _, p := range paths {
		if len(p) == 0 {
			t.Error("collectTOMLPaths returned an empty path slice")
		}
	}
}

func TestCollectTOMLPathsIncludesKnownPaths(t *testing.T) {
	paths := collectTOMLPaths(reflect.TypeOf(model.AppConfig{}), nil)
	want := [][]string{
		{"storage_path"},
		{"weekly_hours_goal"},
		{"keybinds", "list", "quit"},
		{"keybinds", "list", "week_view"},
		{"keybinds", "day", "add_work"},
		{"keybinds", "day", "todo_overview"},
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[joinPath(p)] = true
	}

	for _, w := range want {
		if !pathSet[joinPath(w)] {
			t.Errorf("collectTOMLPaths missing expected path %v", w)
		}
	}
}

func joinPath(p []string) string {
	s := ""
	for i, part := range p {
		if i > 0 {
			s += "."
		}
		s += part
	}
	return s
}
