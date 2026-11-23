package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ai-house-generator/api/internal/queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	StreamKey  = queue.StreamKey
	GroupName  = "ahg-workers"
	DeadStream = "ahg:jobs:dead"
	JobPrefix  = "ahg:jobs:"

	lockPrefix = "ahg:jobs:lock:"
	lockTTL    = 60 * time.Second
	lockRenew  = 20 * time.Second
)

type Worker struct {
	redis       *redis.Client
	consumer    string
	maxAttempts int
	minIdle     time.Duration

	// Configurable per worker instance for deterministic lease tests.
	lockTTL    time.Duration
	lockRenew  time.Duration
	processJob func(context.Context, string, string) error
}

var startJobScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then
    return -1
end
local status = redis.call('HGET', KEYS[1], 'status')
if status == 'completed' or status == 'failed' then
    return 0
end
local attempts = redis.call('HINCRBY', KEYS[1], 'attempts', 1)
redis.call('HSET', KEYS[1],
    'status', 'processing',
    'worker', ARGV[2],
    'attempts', attempts,
    'started_at', ARGV[3])
return attempts
`)

var renewLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 0
`)

var releaseLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
end
return 0
`)

var completeJobScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then
    return -1
end
redis.call('HSET', KEYS[1],
    'status', 'completed',
    'completed_at', ARGV[4])
redis.call('XACK', KEYS[3], ARGV[2], ARGV[3])
redis.call('DEL', KEYS[2])
return 1
`)

var retryJobScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then
    return -1
end
redis.call('HSET', KEYS[1],
    'status', 'queued',
    'attempts', ARGV[2],
    'last_error', ARGV[3])
redis.call('DEL', KEYS[2])
return 1
`)

var failJobScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1] then
    return -1
end
local existing = redis.call('GET', KEYS[5])
if not existing then
    local dlqID = redis.call('XADD', KEYS[4], '*',
        'job_id', ARGV[4],
        'type', ARGV[5],
        'payload', ARGV[6],
        'error', ARGV[7],
        'attempts', ARGV[8],
        'original_id', ARGV[3],
        'failed_at', ARGV[9])
    redis.call('SET', KEYS[5], dlqID)
end
redis.call('HSET', KEYS[1],
    'status', 'failed',
    'attempts', ARGV[8],
    'last_error', ARGV[7],
    'failed_at', ARGV[9])
redis.call('XACK', KEYS[3], ARGV[2], ARGV[3])
redis.call('DEL', KEYS[2])
return 1
`)

func New(cfg queue.Config) *Worker {
	consumer := os.Getenv("REDIS_WORKER_CONSUMER")
	if consumer == "" {
		host, _ := os.Hostname()
		consumer = fmt.Sprintf("%s-%d", host, os.Getpid())
	}

	maxAttempts := 3
	if value := os.Getenv("REDIS_WORKER_MAX_ATTEMPTS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			maxAttempts = parsed
		}
	}

	minIdle := 30 * time.Second
	if value := os.Getenv("REDIS_WORKER_MIN_IDLE_MS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			minIdle = time.Duration(parsed) * time.Millisecond
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Host + ":" + cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10,
		MinIdleConns: 1,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	return &Worker{
		redis:       client,
		consumer:    consumer,
		maxAttempts: maxAttempts,
		minIdle:     minIdle,
		lockTTL:     lockTTL,
		lockRenew:   lockRenew,
		processJob: func(_ context.Context, jobType, payload string) error {
			return validateJob(jobType, payload)
		},
	}
}

func (w *Worker) Close() error {
	return w.redis.Close()
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis ping: %w", err)
	}

	err := w.redis.XGroupCreateMkStream(
		ctx, StreamKey, GroupName, "0",
	).Err()
	if err != nil && !isBusyGroup(err) {
		return fmt.Errorf("create consumer group: %w", err)
	}

	log.Printf(
		"Worker started: consumer=%s group=%s max_attempts=%d lock_ttl=%s",
		w.consumer, GroupName, w.maxAttempts, lockTTL,
	)

	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := w.recoverPending(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("pending recovery error: %v", err)
		}

		streams, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: GroupName, Consumer: w.consumer,
			Streams: []string{StreamKey, ">"},
			Count:   1, Block: 3 * time.Second,
		}).Result()

		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("stream read error: %v", err)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := w.process(ctx, message); err != nil {
					log.Printf("job processing error: %v", err)
				}
			}
		}
	}
}

func isBusyGroup(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "BUSYGROUP")
}

func (w *Worker) recoverPending(ctx context.Context) error {
	messages, _, err := w.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream: StreamKey, Group: GroupName,
		Consumer: w.consumer, MinIdle: w.minIdle,
		Start: "0-0", Count: 10,
	}).Result()

	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, message := range messages {
		if err := w.process(ctx, message); err != nil {
			log.Printf("recovered job error: %v", err)
		}
	}
	return nil
}

func (w *Worker) process(ctx context.Context, message redis.XMessage) error {
	jobID, ok := message.Values["job_id"].(string)
	if !ok || jobID == "" {
		log.Printf("invalid stream message: %s", message.ID)
		return w.ack(ctx, message.ID)
	}

	jobKey := JobPrefix + jobID
	lockKey := lockPrefix + jobID
	token := uuid.NewString()

	ttl := w.lockTTL
	if ttl <= 0 {
		ttl = lockTTL
	}

	renewInterval := w.lockRenew
	if renewInterval <= 0 {
		renewInterval = lockRenew
	}

	acquired, err := w.redis.SetNX(ctx, lockKey, token, ttl).Result()
	if err != nil {
		return fmt.Errorf("acquire job lock: %w", err)
	}
	if !acquired {
		log.Printf("job already locked by another worker: id=%s", jobID)
		return nil
	}

	// Release only if this worker still owns the lock.
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = releaseLockScript.Run(
			releaseCtx, w.redis, []string{lockKey}, token,
		).Err()
	}()

	workCtx, cancelWork := context.WithCancel(ctx)
	stopRenew := make(chan struct{})
	renewDone := make(chan struct{})

	go func() {
		defer close(renewDone)
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stopRenew:
				return
			case <-workCtx.Done():
				return
			case <-ticker.C:
				renewCtx, cancel := context.WithTimeout(
					context.Background(), 3*time.Second,
				)
				result, renewErr := renewLockScript.Run(
					renewCtx, w.redis, []string{lockKey},
					token, ttl.Milliseconds(),
				).Int64()
				cancel()

				if renewErr != nil || result != 1 {
					log.Printf("job lock renewal failed: id=%s err=%v", jobID, renewErr)
					cancelWork()
					return
				}
			}
		}
	}()

	defer func() {
		close(stopRenew)
		<-renewDone
		cancelWork()
	}()

	values, err := w.redis.HGetAll(workCtx, jobKey).Result()
	if err != nil {
		return fmt.Errorf("read job: %w", err)
	}
	if len(values) == 0 {
		log.Printf("job hash missing: %s", jobID)
		return w.ack(workCtx, message.ID)
	}

	if values["status"] == "completed" || values["status"] == "failed" {
		return w.ack(workCtx, message.ID)
	}

	jobType := values["type"]
	payload := values["payload"]

	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	attempts, err := startJobScript.Run(
		workCtx, w.redis, []string{jobKey, lockKey},
		token, w.consumer, startedAt,
	).Int64()
	if err != nil {
		return fmt.Errorf("start job atomically: %w", err)
	}
	if attempts == -1 {
		return fmt.Errorf("job lock ownership lost: %s", jobID)
	}
	if attempts == 0 {
		return w.ack(workCtx, message.ID)
	}

	handler := w.processJob
	if handler == nil {
		handler = func(_ context.Context, jobType, payload string) error {
			return validateJob(jobType, payload)
		}
	}

	processErr := handler(workCtx, jobType, payload)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if workCtx.Err() != nil {
		return fmt.Errorf("job lease lost during processing: %s", jobID)
	}

	if processErr != nil {
		if attempts >= int64(w.maxAttempts) {
			failedAt := time.Now().UTC().Format(time.RFC3339Nano)
			markerKey := "ahg:jobs:dlq:" + jobID

			result, err := failJobScript.Run(
				workCtx, w.redis,
				[]string{jobKey, lockKey, StreamKey, DeadStream, markerKey},
				token, GroupName, message.ID,
				jobID, jobType, payload, processErr.Error(),
				attempts, failedAt,
			).Int64()
			if err != nil {
				return fmt.Errorf("finalize failed job: %w", err)
			}
			if result != 1 {
				return fmt.Errorf("failed job finalization rejected: %s", jobID)
			}

			log.Printf("job moved to DLQ: id=%s attempts=%d", jobID, attempts)
			return nil
		}

		result, err := retryJobScript.Run(
			workCtx, w.redis, []string{jobKey, lockKey},
			token, attempts, processErr.Error(),
		).Int64()
		if err != nil {
			return fmt.Errorf("schedule job retry: %w", err)
		}
		if result != 1 {
			return fmt.Errorf("retry transition rejected: %s", jobID)
		}

		log.Printf(
			"job will retry: id=%s attempt=%d/%d error=%v",
			jobID, attempts, w.maxAttempts, processErr,
		)
		return nil
	}

	completedAt := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := completeJobScript.Run(
		workCtx, w.redis,
		[]string{jobKey, lockKey, StreamKey},
		token, GroupName, message.ID, completedAt,
	).Int64()
	if err != nil {
		return fmt.Errorf("complete job atomically: %w", err)
	}
	if result != 1 {
		return fmt.Errorf("job completion rejected: %s", jobID)
	}

	log.Printf(
		"job completed: id=%s type=%s attempts=%d",
		jobID, jobType, attempts,
	)
	return nil
}

func validateJob(jobType string, payload string) error {
	allowed := map[string]bool{
		"design.generate": true,
		"cad.export":      true,
		"boq.calculate":   true,
		"rab.calculate":   true,
	}

	if !allowed[jobType] {
		return fmt.Errorf("unsupported job type: %s", jobType)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	if data == nil {
		return fmt.Errorf("payload must be a JSON object")
	}
	return nil
}

func (w *Worker) ack(ctx context.Context, messageID string) error {
	return w.redis.XAck(ctx, StreamKey, GroupName, messageID).Err()
}
