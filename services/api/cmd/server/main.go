package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-house-generator/api/internal/database"
	"github.com/ai-house-generator/api/internal/projects"
	"github.com/ai-house-generator/api/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Service:   "ai-house-generator-api",
		Version:   "0.1.0",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func readyHandler(pool *pgxpool.Pool, redisClient *queue.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		databaseStatus := "connected"
		redisStatus := "connected"

		if err := pool.Ping(ctx); err != nil {
			databaseStatus = "unavailable"
		}

		if err := redisClient.Ping(ctx); err != nil {
			redisStatus = "unavailable"
		}

		status := http.StatusOK
		overall := "ready"

		if databaseStatus != "connected" || redisStatus != "connected" {
			status = http.StatusServiceUnavailable
			overall = "not_ready"
		}

		writeJSON(w, status, map[string]string{
			"status":   overall,
			"database": databaseStatus,
			"redis":    redisStatus,
		})
	}
}

func main() {
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startupCancel()

	migrationConfig, err := database.LoadMigratorConfig()
	if err != nil {
		log.Fatalf("migration database configuration error: %v", err)
	}

	migrationPool, err := database.Connect(startupCtx, migrationConfig)
	if err != nil {
		log.Fatalf("migration database connection failed: %v", err)
	}

	if err := database.RunMigrations(startupCtx, migrationPool); err != nil {
		migrationPool.Close()
		log.Fatalf("database migration failed: %v", err)
	}
	migrationPool.Close()

	dbConfig, err := database.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("runtime database configuration error: %v", err)
	}

	pool, err := database.Connect(startupCtx, dbConfig)
	if err != nil {
		log.Fatalf("runtime database connection failed: %v", err)
	}
	defer pool.Close()
	redisConfig, err := queue.LoadConfig()
	if err != nil {
		log.Fatalf("Redis configuration error: %v", err)
	}

	redisClient := queue.New(redisConfig)
	if err := redisClient.Ping(startupCtx); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer redisClient.Close()

	jobHandler := queue.NewHandler(redisClient)

	mux := http.NewServeMux()
	projectsHandler := projects.NewHandler(pool)
	mux.Handle("/api/v1/projects", projectsHandler)
	mux.Handle("/api/v1/projects/", projectsHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/v1/health", healthHandler)
	mux.HandleFunc("/api/v1/ready", readyHandler(pool, redisClient))
	mux.HandleFunc("/api/v1/jobs", jobHandler.CreateJob)
	mux.HandleFunc("/api/v1/jobs/{id}", jobHandler.GetJob)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8097"
	}

	host := os.Getenv("API_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	server := &http.Server{
		Addr:              host + ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("AI House Generator API listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
			stop <- os.Interrupt
		}
	}()

	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}

	log.Println("API server stopped")
}
