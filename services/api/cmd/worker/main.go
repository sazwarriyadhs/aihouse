package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-house-generator/api/internal/queue"
	"github.com/ai-house-generator/api/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../../.env", ".env")

	cfg, err := queue.LoadConfig()
	if err != nil {
		log.Fatalf("Redis configuration error: %v", err)
	}

	jobWorker := worker.New(cfg)
	defer jobWorker.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("AI House Generator Worker initializing")

	if err := jobWorker.Run(ctx); err != nil {
		log.Fatalf("Worker stopped: %v", err)
	}

	log.Println("Worker stopped gracefully")
}
