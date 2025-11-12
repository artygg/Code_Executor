package main

import (
	"Code_executor/internal/config"
	localhttp "Code_executor/internal/http"
	redisqueue "Code_executor/internal/queue/redis"
	postgresrepo "Code_executor/internal/repository/postgres"
	"Code_executor/internal/service"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"time"
)

func main() {
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
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	producer, err := redisqueue.NewProducer(redisClient, cfg.QueueKey)
	if err != nil {
		log.Fatalf("init redis producer: %v", err)
	}

	serviceDeps := service.ExecutionServiceDeps{
		Repo:     repo,
		Producer: producer,
		IDGenerator: func() (string, error) {
			return uuid.NewString(), nil
		},
		Now: time.Now,
	}

	execService, err := service.NewExecutionService(serviceDeps)
	if err != nil {
		log.Fatalf("init execution service: %v", err)
	}

	handler, err := localhttp.NewExecutionHandler(execService)
	if err != nil {
		log.Fatalf("init execution handler: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger)
	r.Route("/api/v1", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})

	log.Printf("Server started on %s", cfg.APIAddr)
	if err := http.ListenAndServe(cfg.APIAddr, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
