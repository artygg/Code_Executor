package queue

import "context"

type Job struct {
	ExecutionID string
	Language    string
	UserID      string
}

type Producer interface {
	Enqueue(ctx context.Context, job Job) error
}

type Consumer interface {
	Consume(ctx context.Context) (<-chan Job, error)
}
