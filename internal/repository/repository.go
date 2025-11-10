package repository

import (
	"Code_executor/internal/domain"
	"context"
	"errors"
)

type ExecutionRepository interface {
	CreateExecution(ctx context.Context, exec *domain.Execution) error
	UpdateExecution(ctx context.Context, exec *domain.Execution) error
	GetExecutionByID(ctx context.Context, id string) (*domain.Execution, error)
}

var (
	ErrExecutionNotFound = errors.New("execution not found")
)
