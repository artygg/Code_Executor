package service

import (
	"Code_executor/internal/domain"
	"Code_executor/internal/queue"
	"Code_executor/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidServiceInput = errors.New("invalid execution service input")
)

type ExecutionService interface {
	CreateExecutionAndEnqueue(ctx context.Context, params CreateExecutionParams) (*domain.Execution, error)
	GetExecution(ctx context.Context, id string) (*domain.Execution, error)
	MarkExecutionCompleted(ctx context.Context, id string, result CompleteExecutionResult) (*domain.Execution, error)
	MarkExecutionFailed(ctx context.Context, id string, result FailExecutionResult) (*domain.Execution, error)
	MarkExecutionTimedOut(ctx context.Context, id string, finishedAt time.Time) (*domain.Execution, error)
}

type executionService struct {
	repo        repository.ExecutionRepository
	producer    queue.Producer
	idGenerator func() (string, error)
	now         func() time.Time
}

type ExecutionServiceDeps struct {
	Repo        repository.ExecutionRepository
	Producer    queue.Producer
	IDGenerator func() (string, error)
	Now         func() time.Time
}

func NewExecutionService(deps ExecutionServiceDeps) (ExecutionService, error) {
	if deps.Repo == nil || deps.Producer == nil || deps.IDGenerator == nil {
		return nil, fmt.Errorf("%w: missing dependencies", ErrInvalidServiceInput)
	}

	nowFn := deps.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	return &executionService{
		repo:        deps.Repo,
		producer:    deps.Producer,
		idGenerator: deps.IDGenerator,
		now:         nowFn,
	}, nil
}

type CreateExecutionParams struct {
	Language  string
	Code      string
	Stdin     string
	TimeoutMs int
	UserID    string
}

type CompleteExecutionResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	FinishedAt time.Time
}

type FailExecutionResult struct {
	Stderr     string
	ExitCode   *int
	FinishedAt time.Time
}

func (s *executionService) CreateExecutionAndEnqueue(ctx context.Context, params CreateExecutionParams) (*domain.Execution, error) {
	if params.UserID == "" {
		return nil, fmt.Errorf("%w: user id is required", ErrInvalidServiceInput)
	}

	execID, err := s.idGenerator()
	if err != nil {
		return nil, fmt.Errorf("generate execution id: %w", err)
	}

	exec, err := domain.NewExecution(execID, params.Language, params.Code, params.Stdin, params.TimeoutMs, params.UserID, s.now())
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}

	job := &queue.Job{
		ExecutionID: exec.ID,
		Language:    exec.Language,
		UserID:      exec.UserID,
	}

	if err := s.producer.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	return exec, nil
}

func (s *executionService) GetExecution(ctx context.Context, id string) (*domain.Execution, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: execution id is required", ErrInvalidServiceInput)
	}

	return s.repo.GetExecutionByID(ctx, id)
}

func (s *executionService) MarkExecutionCompleted(ctx context.Context, id string, result CompleteExecutionResult) (*domain.Execution, error) {
	if result.FinishedAt.IsZero() {
		return nil, fmt.Errorf("%w: finished at is required", ErrInvalidServiceInput)
	}

	exec, err := s.repo.GetExecutionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := exec.MarkCompleted(result.Stdout, result.Stderr, result.ExitCode, result.FinishedAt); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateExecution(ctx, exec); err != nil {
		return nil, err
	}

	return exec, nil
}

func (s *executionService) MarkExecutionFailed(ctx context.Context, id string, result FailExecutionResult) (*domain.Execution, error) {
	if result.FinishedAt.IsZero() {
		return nil, fmt.Errorf("%w: finished at is required", ErrInvalidServiceInput)
	}

	exec, err := s.repo.GetExecutionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := exec.MarkFailed(result.Stderr, result.ExitCode, result.FinishedAt); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateExecution(ctx, exec); err != nil {
		return nil, err
	}

	return exec, nil
}

func (s *executionService) MarkExecutionTimedOut(ctx context.Context, id string, finishedAt time.Time) (*domain.Execution, error) {
	if finishedAt.IsZero() {
		return nil, fmt.Errorf("%w: finished at is required", ErrInvalidServiceInput)
	}

	exec, err := s.repo.GetExecutionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := exec.MarkTimedOut(finishedAt); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateExecution(ctx, exec); err != nil {
		return nil, err
	}

	return exec, nil
}
