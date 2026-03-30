package tui

import (
	"errors"

	"github.com/sleepypxnda/schmournal/internal/application/usecase"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

// UseCaseSet is an assembled bundle of repositories and use case instances.
// The composition root (main) owns how this set is created.
type UseCaseSet struct {
	DayRecordRepo repository.DayRecordRepository
	TodoRepo      repository.TodoRepository
	StateRepo     repository.StateRepository

	LoadDayRecord      *usecase.LoadDayRecordUseCase
	LoadAllDayRecords  *usecase.LoadAllDayRecordsUseCase
	SaveDayRecord      *usecase.SaveDayRecordUseCase
	DeleteDayRecord    *usecase.DeleteDayRecordUseCase
	LoadWorkspaceTodos *usecase.LoadWorkspaceTodosUseCase
	SaveWorkspaceTodos *usecase.SaveWorkspaceTodosUseCase

	AddWorkEntry    *usecase.AddWorkEntryUseCase
	UpdateWorkEntry *usecase.UpdateWorkEntryUseCase
	DeleteWorkEntry *usecase.DeleteWorkEntryUseCase
	SubmitWorkForm  *usecase.SubmitWorkFormUseCase
	SetDayTimes     *usecase.SetDayTimesUseCase
	UpdateNotes     *usecase.UpdateNotesUseCase
	ManageTodos     *usecase.ManageTodosUseCase
}

// UseCaseSetFactory creates a new set for a given storage path.
type UseCaseSetFactory func(storagePath string) (UseCaseSet, error)

// UseCases holds all application use cases for the UI layer.
type UseCases struct {
	UseCaseSet
	factory UseCaseSetFactory
}

// NewUseCases creates a UI use case container from an already-assembled set.
func NewUseCases(set UseCaseSet, factory UseCaseSetFactory) *UseCases {
	u := &UseCases{factory: factory}
	u.applySet(set)
	return u
}

func (u *UseCases) applySet(set UseCaseSet) {
	u.UseCaseSet = set
}

// ReinitializeForStorage rebuilds repositories and use cases for a new storage
// path by delegating to the injected factory from the composition root.
func (u *UseCases) ReinitializeForStorage(storagePath string) error {
	if u.factory == nil {
		return errors.New("use case factory is not configured")
	}
	set, err := u.factory(storagePath)
	if err != nil {
		return err
	}
	u.applySet(set)
	return nil
}

// SaveActiveWorkspace persists the currently active workspace in app state.
func (u *UseCases) SaveActiveWorkspace(name string) error {
	if u.StateRepo == nil {
		return errors.New("state repository is not configured")
	}
	return u.StateRepo.SaveState(usecase.AppStateDTO{ActiveWorkspace: name}.ToDomain())
}
