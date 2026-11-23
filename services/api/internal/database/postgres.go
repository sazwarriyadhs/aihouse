package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ai-house-generator/api/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func LoadRuntimeConfig() (Config, error) {
	return loadConfig("POSTGRES_RUNTIME_USER", "POSTGRES_RUNTIME_PASSWORD")
}

func LoadMigratorConfig() (Config, error) {
	return loadConfig("POSTGRES_MIGRATOR_USER", "POSTGRES_MIGRATOR_PASSWORD")
}

func loadConfig(userKey, passwordKey string) (Config, error) {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	cfg := Config{
		Host:     envOr("POSTGRES_HOST", "127.0.0.1"),
		Port:     envOr("POSTGRES_PORT", "55432"),
		User:     os.Getenv(userKey),
		Password: os.Getenv(passwordKey),
		Database: os.Getenv("POSTGRES_DB"),
	}

	if cfg.User == "" || cfg.Password == "" || cfg.Database == "" {
		return Config{}, fmt.Errorf("%s, %s and POSTGRES_DB must be configured", userKey, passwordKey)
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
	}

	return cfg, nil
}
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (c Config) DSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   c.Database,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL config: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SET ROLE ahg_owner"); err != nil {
		return fmt.Errorf("assume migration owner role: %w", err)
	}
	defer func() {
		resetCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.Exec(resetCtx, "RESET ROLE"); err != nil {
			log.Printf("warning: reset migration role: %v", err)
		}
	}()

	// Serialize migration execution across application instances.
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(724190261)"); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock(724190261)")
	}()

	if _, err := conn.Exec(ctx, `
        CREATE SCHEMA IF NOT EXISTS app;
        CREATE TABLE IF NOT EXISTS app.schema_migrations (
            version TEXT PRIMARY KEY,
            filename TEXT NOT NULL,
            checksum TEXT NOT NULL,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            execution_ms BIGINT NOT NULL DEFAULT 0
        )
    `); err != nil {
		return fmt.Errorf("prepare migration ledger: %w", err)
	}

	entries, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)

	for _, filename := range entries {
		contents, err := migrations.Files.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read %s: %w", filename, err)
		}

		version := strings.TrimSuffix(filepath.Base(filename), ".sql")
		sum := sha256.Sum256(contents)
		checksum := hex.EncodeToString(sum[:])

		var existing string
		err = conn.QueryRow(ctx,
			"SELECT checksum FROM app.schema_migrations WHERE version = $1",
			version,
		).Scan(&existing)

		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration checksum mismatch: %s; do not edit an already-applied migration", filename)
			}
			log.Printf("migration already applied: %s", filename)
			continue
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", filename, err)
		}

		started := time.Now()
		if _, err = tx.Exec(ctx, string(contents)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", filename, err)
		}

		duration := time.Since(started).Milliseconds()
		_, err = tx.Exec(ctx, `
            INSERT INTO app.schema_migrations
                (version, filename, checksum, execution_ms)
            VALUES ($1, $2, $3, $4)
        `, version, filename, checksum, duration)

		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", filename, err)
		}

		log.Printf("migration applied: %s (%d ms)", filename, duration)
	}

	return nil
}
