package redisqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"Code_executor/internal/queue"

	rds "github.com/redis/go-redis/v9"
)

const executionsListKey = "queue:executions"

type producer struct {
	client *rds.Client
	key    string
}

type consumer struct {
	client     *rds.Client
	key        string
	popTimeout time.Duration
}

type jobPayload struct {
	ExecutionID string `json:"execution_id"`
	Language    string `json:"language,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

var (
	errNilRedisClient = errors.New("redis client is nil")
)

func NewProducer(redisClient *rds.Client, key string) (queue.Producer, error) {
	if redisClient == nil {
		return nil, errNilRedisClient
	}

	return &producer{
		client: redisClient,
		key:    normalizeQueueKey(key),
	}, nil
}

func NewConsumer(redisClient *rds.Client, key string, popTimeout time.Duration) (queue.Consumer, error) {
	if redisClient == nil {
		return nil, errNilRedisClient
	}

	if popTimeout <= 0 {
		popTimeout = 5 * time.Second
	}

	return &consumer{
		client:     redisClient,
		key:        normalizeQueueKey(key),
		popTimeout: popTimeout,
	}, nil
}

func (p *producer) Enqueue(ctx context.Context, job queue.Job) error {
	if job.ExecutionID == "" {
		return fmt.Errorf("execution id is required")
	}

	payload := jobPayload{
		ExecutionID: job.ExecutionID,
		Language:    job.Language,
		UserID:      job.UserID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	if err := p.client.LPush(ctx, p.key, data).Err(); err != nil {
		return fmt.Errorf("redis lpush: %w", err)
	}

	return nil
}

func (c *consumer) Consume(ctx context.Context) (<-chan queue.Job, error) {
	out := make(chan queue.Job)

	go func() {
		defer close(out)

		for {
			if ctx.Err() != nil {
				return
			}

			brCtx, cancel := context.WithTimeout(ctx, c.popTimeout)
			result, err := c.client.BRPop(brCtx, c.popTimeout, c.key).Result()
			cancel()

			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}

				if errors.Is(err, rds.Nil) {
					continue
				}

				continue
			}

			if len(result) != 2 {
				continue
			}

			var payload jobPayload
			if err := json.Unmarshal([]byte(result[1]), &payload); err != nil {
				continue
			}

			job := queue.Job{
				ExecutionID: payload.ExecutionID,
				Language:    payload.Language,
				UserID:      payload.UserID,
			}

			select {
			case <-ctx.Done():
				return
			case out <- job:
			}
		}
	}()

	return out, nil
}

func normalizeQueueKey(key string) string {
	if key == "" {
		return executionsListKey
	}

	return key
}
