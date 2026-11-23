package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	StreamKey = "ahg:jobs:stream"
	JobPrefix = "ahg:jobs:"
)

var allowedTypes = map[string]bool{
	"design.generate": true,
	"cad.export":      true,
	"boq.calculate":   true,
	"rab.calculate":   true,
}

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type Client struct {
	redis  *redis.Client
	stream string
}

type Job struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func New(cfg Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Host + ":" + cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10,
		MinIdleConns: 1,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &Client{
		redis:  rdb,
		stream: StreamKey,
	}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.redis.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.redis.Close()
}

func (c *Client) Enqueue(ctx context.Context, jobType string, payload json.RawMessage) (Job, error) {
	if !allowedTypes[jobType] {
		return Job{}, fmt.Errorf("unsupported job type: %s", jobType)
	}

	if !json.Valid(payload) || len(payload) == 0 {
		return Job{}, fmt.Errorf("payload must be valid JSON")
	}

	job := Job{
		ID:        uuid.NewString(),
		Type:      jobType,
		Status:    "queued",
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}

	payloadText := string(payload)
	createdAt := job.CreatedAt.Format(time.RFC3339Nano)

	pipe := c.redis.TxPipeline()

	pipe.HSet(ctx, JobPrefix+job.ID, map[string]interface{}{
		"id":         job.ID,
		"type":       job.Type,
		"status":     job.Status,
		"payload":    payloadText,
		"created_at": createdAt,
	})

	pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: c.stream,
		Values: map[string]interface{}{
			"job_id":  job.ID,
			"type":    job.Type,
			"payload": payloadText,
		},
	})

	if _, err := pipe.Exec(ctx); err != nil {
		return Job{}, fmt.Errorf("enqueue job: %w", err)
	}

	return job, nil
}

func (c *Client) GetJob(ctx context.Context, id string) (Job, error) {
	values, err := c.redis.HGetAll(ctx, JobPrefix+id).Result()
	if err != nil {
		return Job{}, fmt.Errorf("read job: %w", err)
	}

	if len(values) == 0 {
		return Job{}, redis.Nil
	}

	var payload json.RawMessage
	if err := json.Unmarshal([]byte(values["payload"]), &payload); err != nil {
		return Job{}, fmt.Errorf("decode job payload: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, values["created_at"])
	if err != nil {
		return Job{}, fmt.Errorf("decode job timestamp: %w", err)
	}

	return Job{
		ID:        values["id"],
		Type:      values["type"],
		Status:    values["status"],
		Payload:   payload,
		CreatedAt: createdAt,
	}, nil
}
