package usecase

import (
	"fmt"

	"github.com/sleepypxnda/schmournal/internal/domain/model"
	"github.com/sleepypxnda/schmournal/internal/domain/repository"
)

type LoadWorkspaceTodosInput struct {
	Workspace string
}

type LoadWorkspaceTodosUseCase struct {
	todoRepo repository.TodoRepository
}

func NewLoadWorkspaceTodosUseCase(todoRepo repository.TodoRepository) *LoadWorkspaceTodosUseCase {
	return &LoadWorkspaceTodosUseCase{todoRepo: todoRepo}
}

func (uc *LoadWorkspaceTodosUseCase) Execute(input LoadWorkspaceTodosInput) (model.WorkspaceTodos, error) {
	if input.Workspace == "" {
		return model.WorkspaceTodos{}, fmt.Errorf("workspace is required")
	}
	todos, err := uc.todoRepo.Load(input.Workspace)
	if err != nil {
		return model.WorkspaceTodos{}, fmt.Errorf("failed to load workspace todos: %w", err)
	}
	return todos, nil
}

func (uc *LoadWorkspaceTodosUseCase) ExecuteDTO(input LoadWorkspaceTodosInput) (WorkspaceTodosDTO, error) {
	todos, err := uc.Execute(input)
	if err != nil {
		return WorkspaceTodosDTO{}, err
	}
	return mapDomainWorkspaceTodosToDTO(todos), nil
}

type SaveWorkspaceTodosInput struct {
	Workspace string
	Todos     model.WorkspaceTodos
}

type SaveWorkspaceTodosUseCase struct {
	todoRepo repository.TodoRepository
}

func NewSaveWorkspaceTodosUseCase(todoRepo repository.TodoRepository) *SaveWorkspaceTodosUseCase {
	return &SaveWorkspaceTodosUseCase{todoRepo: todoRepo}
}

func (uc *SaveWorkspaceTodosUseCase) Execute(input SaveWorkspaceTodosInput) error {
	if input.Workspace == "" {
		return fmt.Errorf("workspace is required")
	}
	if err := uc.todoRepo.Save(input.Workspace, input.Todos); err != nil {
		return fmt.Errorf("failed to save workspace todos: %w", err)
	}
	return nil
}

type SaveWorkspaceTodosDTOInput struct {
	Workspace string
	Todos     WorkspaceTodosDTO
}

func (uc *SaveWorkspaceTodosUseCase) ExecuteDTO(input SaveWorkspaceTodosDTOInput) error {
	return uc.Execute(SaveWorkspaceTodosInput{
		Workspace: input.Workspace,
		Todos:     mapWorkspaceTodosDTOToDomain(input.Todos),
	})
}
