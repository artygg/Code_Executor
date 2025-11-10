package domain

import (
	"errors"
	"fmt"
	"time"
)

type ExecutionStatus string

const (
	ExecutionStatusQueued    ExecutionStatus = "queued"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimedOut  ExecutionStatus = "timed_out"
)

var (
	ErrInvalidExecution        = errors.New("invalid execution")
	ErrInvalidStatusTransition = errors.New("invalid execution status transition")
)

var finalStatuses = map[ExecutionStatus]struct{}{
	ExecutionStatusCompleted: {},
	ExecutionStatusFailed:    {},
	ExecutionStatusTimedOut:  {},
}
var statusTransitions = map[ExecutionStatus]map[ExecutionStatus]struct{}{
	ExecutionStatusQueued: {
		ExecutionStatusRunning: {},
	},

	ExecutionStatusRunning: {
		ExecutionStatusCompleted: {},
		ExecutionStatusFailed:    {},
		ExecutionStatusTimedOut:  {},
	},
}

type Execution struct {
	ID         string
	Language   string
	Code       string
	Stdin      string
	TimeoutMs  int
	Status     ExecutionStatus
	Stdout     string
	Stderr     string
	ExitCode   *int
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	UserID     string
}

func NewExecution(id, language, code, stdin string, timeoutMs int, userID string, createdAt time.Time) (*Execution, error) {
	if id == "" || language == "" || code == "" || userID == "" || createdAt.IsZero() {
		return nil, fmt.Errorf("%w: missing required fields", ErrInvalidExecution)
	}
	if timeoutMs <= 0 {
		return nil, fmt.Errorf("%w: timeout must be positive", ErrInvalidExecution)
	}

	execution := &Execution{
		ID:        id,
		Language:  language,
		Code:      code,
		Stdin:     stdin,
		TimeoutMs: timeoutMs,
		Status:    ExecutionStatusQueued,
		CreatedAt: createdAt.UTC(),
		UserID:    userID,
	}

	return execution, nil
}

func (e *Execution) MarkRunning(startedAt time.Time) error {
	if startedAt.IsZero() {
		return fmt.Errorf("%w: started at time is zero", ErrInvalidExecution)
	}

	if err := e.transition(ExecutionStatusRunning); err != nil {
		return err
	}

	e.StartedAt = timePtr(startedAt)
	return nil
}

func (e *Execution) MarkCompleted(stdout, stderr string, exitCode int, finishedAt time.Time) error {
	if finishedAt.IsZero() {
		return fmt.Errorf("%w: finished at time is zero", ErrInvalidExecution)
	}

	if err := e.transition(ExecutionStatusCompleted); err != nil {
		return err
	}

	e.Stdout = stdout
	e.Stderr = stderr
	e.ExitCode = intPtr(exitCode)
	e.FinishedAt = timePtr(finishedAt)
	return nil
}

func (e *Execution) MarkFailed(stderr string, exitCode *int, finishedAt time.Time) error {
	if finishedAt.IsZero() {
		return fmt.Errorf("%w: finished at time is zero", ErrInvalidExecution)
	}

	if err := e.transition(ExecutionStatusFailed); err != nil {
		return err
	}

	e.Stderr = stderr
	e.ExitCode = exitCode
	e.FinishedAt = timePtr(finishedAt)
	return nil
}

func (e *Execution) MarkTimedOut(finishedAt time.Time) error {
	if finishedAt.IsZero() {
		return fmt.Errorf("%w: finished at time is zero", ErrInvalidExecution)
	}

	if err := e.transition(ExecutionStatusTimedOut); err != nil {
		return err
	}

	e.FinishedAt = timePtr(finishedAt)
	return nil
}

func (e *Execution) transition(newStatus ExecutionStatus) error {
	if _, isFinal := finalStatuses[e.Status]; isFinal {
		return fmt.Errorf("%w: current status %s is final", ErrInvalidStatusTransition, e.Status)
	}

	allowedNext, ok := statusTransitions[e.Status]
	if !ok {
		return fmt.Errorf("%w: no transitions defined for status %s", ErrInvalidStatusTransition, e.Status)
	}

	if _, ok := allowedNext[newStatus]; !ok {
		return fmt.Errorf("%w: %s -> %s not allowed", ErrInvalidStatusTransition, e.Status, newStatus)
	}

	e.Status = newStatus
	return nil
}

func timePtr(t time.Time) *time.Time {
	tt := t.UTC()
	return &tt
}

func intPtr(v int) *int {
	val := v
	return &val
}
