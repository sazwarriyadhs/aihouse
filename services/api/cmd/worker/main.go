package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-house-generator/api/internal/cad"
	"github.com/ai-house-generator/api/internal/database"
	"github.com/ai-house-generator/api/internal/queue"
	"github.com/ai-house-generator/api/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../../.env", ".env")

	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	redisConfig, err := queue.LoadConfig()
	if err != nil {
		log.Fatalf("Redis configuration error: %v", err)
	}

	runtimeConfig, err := database.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("PostgreSQL runtime configuration error: %v", err)
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, 10*time.Second)
	pool, err := database.Connect(connectCtx, runtimeConfig)
	cancelConnect()
	if err != nil {
		log.Fatalf("PostgreSQL connection error: %v", err)
	}
	defer pool.Close()

	script := cad.EmbeddedExporter()

	repository := cad.NewRepository(pool)
	executor, err := cad.NewExecutor(
		repository,
		os.Getenv("CAD_STORAGE_ROOT"),
		os.Getenv("FREECAD_PYTHON"),
		script,
	)
	if err != nil {
		log.Fatalf("CAD executor configuration error: %v", err)
	}

	handler := func(
		jobCtx context.Context,
		jobID, jobType, payload string,
	) error {
		if jobType != "cad.export" {
			return worker.ValidateJob(jobType, payload)
		}

		err := executor.Process(jobCtx, jobID, jobType, payload)
		var validationErr *cad.ValidationError
		if errors.As(err, &validationErr) {
			return worker.Permanent(err)
		}
		return err
	}

	jobWorker := worker.NewWithJobHandler(redisConfig, handler)
	defer jobWorker.Close()

	log.Println("AI House Generator Worker initializing")
	log.Println("CAD export executor enabled")

	if err := jobWorker.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("Worker stopped: %v", err)
	}

	log.Println("Worker stopped gracefully")
}
