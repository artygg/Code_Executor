package queuememory

import (
	"Code_executor/internal/queue"
	"context"
	"errors"
	"fmt"
	"log"
)

type InMemoryQueue struct {
	ch chan queue.Job
}

var (
	ErrInvalidMemoryBufferSize = errors.New("invalid memory buffer size")
)

func NewInMemoryQueue(buffer int) (*InMemoryQueue, error) {
	if buffer <= 0 {
		return nil, ErrInvalidMemoryBufferSize
	}
	return &InMemoryQueue{ch: make(chan queue.Job, buffer)}, nil
}

func (q *InMemoryQueue) Enqueue(ctx context.Context, job queue.Job) error {

	if job.ExecutionID == "" {
		return fmt.Errorf("execution id is empty")
	}

	select {
	case <-ctx.Done():
		log.Printf("enqueue cancelled: %v", ctx.Err())
		return ctx.Err()
	case q.ch <- job:
		return nil
	}
}

func (q *InMemoryQueue) Consume(ctx context.Context) (<-chan queue.Job, error) {
	out := make(chan queue.Job)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-q.ch:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- job:
				}
			}
		}
	}()
	return out, nil
}
