package main

import (
	"Code_executor/internal/config"
	redisqueue "Code_executor/internal/queue/redis"
	postgresrepo "Code_executor/internal/repository/postgres"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

func main() {
	fmt.Println("Starting worker")
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	repo, err := postgresrepo.NewExecutionRepository(pool)
	if err != nil {
		log.Fatalf("init postgres repo: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	queue, err := redisqueue.NewConsumer(redisClient, cfg.QueueKey, cfg.PopTimeout)
	if err != nil {
		log.Fatalf("Redis cannot create new consumer: %v", err)
	}

	jobs, err := queue.Consume(ctx)
	if err != nil {
		panic(err)
	}

	for job := range jobs {
		exec, err := repo.GetExecutionByID(ctx, job.ExecutionID)
		if err != nil {
			panic(err)
		}
		err = exec.MarkRunning(time.Now())
		if err != nil {
			panic(err)
		}
		err = repo.UpdateExecution(ctx, exec)
		if err != nil {
			panic(err)
		}

		fmt.Printf("⚙️ Processing job %s\n", job.ExecutionID)
		time.Sleep(2 * time.Second) // simulate "work"

		stdout := fmt.Sprintf("Fake output for %s code", exec.Language)
		err = exec.MarkCompleted(stdout, "", 0, time.Now())
		if err != nil {
			panic(err)
		}
		err = repo.UpdateExecution(ctx, exec)
		if err != nil {
			panic(err)
		}

		fmt.Printf("✅ Completed job %s\n", job.ExecutionID)
	}
}
