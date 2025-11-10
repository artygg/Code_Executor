package memory

import (
	"Code_executor/internal/domain"
	"Code_executor/internal/repository"
	"context"
	"fmt"
	"sync"
)

type ExecutionRepository struct {
	mu    sync.RWMutex
	store map[string]*domain.Execution
}

func NewExecutionRepository() *ExecutionRepository {
	return &ExecutionRepository{
		store: make(map[string]*domain.Execution),
	}
}

func (r *ExecutionRepository) CreateExecution(_ context.Context, exec *domain.Execution) error {
	if exec == nil {
		return fmt.Errorf("execution is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.store[exec.ID]; exists {
		return fmt.Errorf("execution with id %s already exists", exec.ID)
	}

	r.store[exec.ID] = cloneExecution(exec)
	return nil
}

func (r *ExecutionRepository) UpdateExecution(_ context.Context, exec *domain.Execution) error {
	if exec == nil {
		return fmt.Errorf("execution is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.store[exec.ID]; !exists {
		return repository.ErrExecutionNotFound
	}

	r.store[exec.ID] = cloneExecution(exec)
	return nil
}

func (r *ExecutionRepository) GetExecutionByID(_ context.Context, id string) (*domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exec, exists := r.store[id]
	if !exists {
		return nil, repository.ErrExecutionNotFound
	}

	return cloneExecution(exec), nil
}

func cloneExecution(src *domain.Execution) *domain.Execution {
	if src == nil {
		return nil
	}

	clone := *src

	if src.ExitCode != nil {
		exitCode := *src.ExitCode
		clone.ExitCode = &exitCode
	}

	if src.StartedAt != nil {
		startedAt := *src.StartedAt
		clone.StartedAt = &startedAt
	}

	if src.FinishedAt != nil {
		finishedAt := *src.FinishedAt
		clone.FinishedAt = &finishedAt
	}

	return &clone
}
