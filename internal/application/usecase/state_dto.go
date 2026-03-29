package usecase

import "github.com/sleepypxnda/schmournal/internal/domain/model"

type AppStateDTO struct {
	ActiveWorkspace string
}

func (s AppStateDTO) ToDomain() model.AppState {
	return model.AppState{
		ActiveWorkspace: s.ActiveWorkspace,
	}
}
