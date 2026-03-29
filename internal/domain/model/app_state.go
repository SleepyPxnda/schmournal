package model

// AppState holds lightweight runtime state persisted across sessions.
type AppState struct {
	ActiveWorkspace string `json:"active_workspace"`
}
