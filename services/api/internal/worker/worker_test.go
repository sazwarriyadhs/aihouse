package worker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("AHG_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set AHG_TEST_REDIS_ADDR to run isolated Redis tests")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatalf("connect test Redis: %v", err)
	}

	t.Cleanup(func() { client.Close() })
	return client
}

func testPrefix() string {
	return "ahg:test:" + uuid.NewString() + ":"
}

func TestConcurrentLockAcquisition(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	key := testPrefix() + "lock"

	const workers = 32
	var winners atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := client.SetNX(ctx, key, uuid.NewString(), 10*time.Second).Result()
			if err != nil {
				t.Errorf("SET NX: %v", err)
				return
			}
			if ok {
				winners.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := winners.Load(); got != 1 {
		t.Fatalf("expected exactly one lock owner, got %d", got)
	}
}

func TestLockOwnershipRenewAndRelease(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	key := testPrefix() + "lock"
	owner := uuid.NewString()
	other := uuid.NewString()

	if err := client.Set(ctx, key, owner, 5*time.Second).Err(); err != nil {
		t.Fatal(err)
	}

	renewed, err := renewLockScript.Run(
		ctx, client, []string{key}, owner, 15000,
	).Int64()
	if err != nil || renewed != 1 {
		t.Fatalf("owner renewal: result=%d err=%v", renewed, err)
	}

	rejected, err := renewLockScript.Run(
		ctx, client, []string{key}, other, 15000,
	).Int64()
	if err != nil || rejected != 0 {
		t.Fatalf("non-owner renewal should fail: result=%d err=%v", rejected, err)
	}

	released, err := releaseLockScript.Run(
		ctx, client, []string{key}, other,
	).Int64()
	if err != nil || released != 0 {
		t.Fatalf("non-owner release should fail: result=%d err=%v", released, err)
	}

	released, err = releaseLockScript.Run(
		ctx, client, []string{key}, owner,
	).Int64()
	if err != nil || released != 1 {
		t.Fatalf("owner release: result=%d err=%v", released, err)
	}

	exists, err := client.Exists(ctx, key).Result()
	if err != nil || exists != 0 {
		t.Fatalf("lock should be deleted: exists=%d err=%v", exists, err)
	}
}

func TestStartCompleteAndAcknowledgeAtomically(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	prefix := testPrefix()
	jobKey, lockKey := prefix+"job", prefix+"lock"
	stream := prefix + "stream"
	group := "test-workers"
	owner := uuid.NewString()

	if err := client.HSet(ctx, jobKey, "status", "queued", "attempts", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, lockKey, owner, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	entryID, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream, Values: map[string]interface{}{"job_id": "test-job"},
	}).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.XGroupCreate(ctx, stream, group, "0").Err(); err != nil {
		t.Fatal(err)
	}
	messages, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: "test-consumer",
		Streams: []string{stream, ">"}, Count: 1,
	}).Result()
	if err != nil || len(messages) != 1 {
		t.Fatalf("read test message: streams=%d err=%v", len(messages), err)
	}

	attempts, err := startJobScript.Run(
		ctx, client, []string{jobKey, lockKey},
		owner, "test-consumer", time.Now().UTC().Format(time.RFC3339Nano),
	).Int64()
	if err != nil || attempts != 1 {
		t.Fatalf("start job: attempts=%d err=%v", attempts, err)
	}

	completed, err := completeJobScript.Run(
		ctx, client, []string{jobKey, lockKey, stream},
		owner, group, entryID, time.Now().UTC().Format(time.RFC3339Nano),
	).Int64()
	if err != nil || completed != 1 {
		t.Fatalf("complete job: result=%d err=%v", completed, err)
	}

	status, err := client.HGet(ctx, jobKey, "status").Result()
	if err != nil || status != "completed" {
		t.Fatalf("expected completed status, got %q err=%v", status, err)
	}

	pending, err := client.XPending(ctx, stream, group).Result()
	if err != nil || pending.Count != 0 {
		t.Fatalf("expected zero pending messages, got %d err=%v", pending.Count, err)
	}

	if exists, err := client.Exists(ctx, lockKey).Result(); err != nil || exists != 0 {
		t.Fatalf("expected lock removal, exists=%d err=%v", exists, err)
	}
}

func TestRetryAndDeadLetter(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	prefix := testPrefix()
	jobKey, lockKey := prefix+"job", prefix+"lock"
	stream, deadStream := prefix+"stream", prefix+"dead"
	group := "test-workers"
	jobID := uuid.NewString()
	owner := uuid.NewString()
	payload := `{"test":true}`

	if err := client.HSet(ctx, jobKey,
		"type", "test.invalid", "payload", payload,
		"status", "processing", "attempts", 1,
	).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, lockKey, owner, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	entryID, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream, Values: map[string]interface{}{"job_id": jobID},
	}).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.XGroupCreate(ctx, stream, group, "0").Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group, Consumer: "test-consumer",
		Streams: []string{stream, ">"}, Count: 1,
	}).Result(); err != nil {
		t.Fatal(err)
	}

	retried, err := retryJobScript.Run(
		ctx, client, []string{jobKey, lockKey},
		owner, 1, "simulated transient error",
	).Int64()
	if err != nil || retried != 1 {
		t.Fatalf("retry transition: result=%d err=%v", retried, err)
	}

	status, _ := client.HGet(ctx, jobKey, "status").Result()
	if status != "queued" {
		t.Fatalf("expected queued after retry, got %q", status)
	}

	// Reacquire the lock to simulate the final attempt.
	owner = uuid.NewString()
	if err := client.Set(ctx, lockKey, owner, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	markerKey := "ahg:jobs:dlq:" + jobID
	failArgs := []interface{}{
		owner, group, entryID, jobID, "test.invalid", payload,
		"unsupported job type", 3, time.Now().UTC().Format(time.RFC3339Nano),
	}
	keys := []string{jobKey, lockKey, stream, deadStream, markerKey}

	failed, err := failJobScript.Run(ctx, client, keys, failArgs...).Int64()
	if err != nil || failed != 1 {
		t.Fatalf("DLQ transition: result=%d err=%v", failed, err)
	}

	status, _ = client.HGet(ctx, jobKey, "status").Result()
	if status != "failed" {
		t.Fatalf("expected failed status, got %q", status)
	}

	dlqCount, err := client.XLen(ctx, deadStream).Result()
	if err != nil || dlqCount != 1 {
		t.Fatalf("expected one DLQ entry, got %d err=%v", dlqCount, err)
	}

	marker, err := client.Get(ctx, markerKey).Result()
	if err != nil || marker == "" {
		t.Fatalf("expected DLQ idempotency marker, got %q err=%v", marker, err)
	}

	// A repeated finalization with a valid new lock must not add another DLQ entry.
	owner = uuid.NewString()
	if err := client.Set(ctx, lockKey, owner, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	failArgs[0] = owner
	_, err = failJobScript.Run(ctx, client, keys, failArgs...).Result()
	if err != nil {
		t.Fatalf("repeat DLQ finalization: %v", err)
	}

	dlqCount, err = client.XLen(ctx, deadStream).Result()
	if err != nil || dlqCount != 1 {
		t.Fatalf("DLQ duplicated: count=%d err=%v", dlqCount, err)
	}

	_ = fmt.Sprintf("%s", marker)
}

func newProcessTestJob(t *testing.T, client *redis.Client, duplicate bool) (*Worker, []redis.XMessage, string) {
	t.Helper()
	ctx := context.Background()

	jobID := uuid.NewString()
	jobKey := JobPrefix + jobID
	lockKey := lockPrefix + jobID
	consumer := "test-" + uuid.NewString()

	if err := client.XGroupCreateMkStream(ctx, StreamKey, GroupName, "0").Err(); err != nil && !isBusyGroup(err) {
		t.Fatalf("create group: %v", err)
	}

	if err := client.HSet(ctx, jobKey,
		"type", "design.generate",
		"payload", `{"project_id":"integration-test","style":"modern_minimalist"}`,
		"status", "queued",
		"attempts", 0,
	).Err(); err != nil {
		t.Fatal(err)
	}

	count := int64(1)
	if duplicate {
		count = 2
	}

	for i := int64(0); i < count; i++ {
		if _, err := client.XAdd(ctx, &redis.XAddArgs{
			Stream: StreamKey,
			Values: map[string]interface{}{"job_id": jobID},
		}).Result(); err != nil {
			t.Fatalf("add stream entry: %v", err)
		}
	}

	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    groupNameForTest(),
		Consumer: consumer,
		Streams:  []string{StreamKey, ">"},
		Count:    count,
	}).Result()
	if err != nil {
		t.Fatalf("read group: %v", err)
	}

	var messages []redis.XMessage
	for _, stream := range streams {
		messages = append(messages, stream.Messages...)
	}
	if len(messages) != int(count) {
		t.Fatalf("expected %d messages, got %d", count, len(messages))
	}

	t.Cleanup(func() {
		_ = client.Del(context.Background(), jobKey, lockKey).Err()
		for _, message := range messages {
			_ = client.XDel(context.Background(), StreamKey, message.ID).Err()
		}
	})

	return &Worker{
		redis:       client,
		consumer:    consumer,
		maxAttempts: 3,
		minIdle:     0,
	}, messages, jobID
}

func groupNameForTest() string {
	return GroupName
}

func TestProcessCompletesRealJob(t *testing.T) {
	client := testRedis(t)
	w, messages, jobID := newProcessTestJob(t, client, false)

	if err := w.process(context.Background(), messages[0]); err != nil {
		t.Fatalf("process job: %v", err)
	}

	status, err := client.HGet(context.Background(), JobPrefix+jobID, "status").Result()
	if err != nil || status != "completed" {
		t.Fatalf("expected completed job, got status=%q err=%v", status, err)
	}

	attempts, err := client.HGet(context.Background(), JobPrefix+jobID, "attempts").Int()
	if err != nil || attempts != 1 {
		t.Fatalf("expected one attempt, got %d err=%v", attempts, err)
	}
}

func TestConcurrentDuplicateDeliveryIsSafe(t *testing.T) {
	client := testRedis(t)
	w, messages, jobID := newProcessTestJob(t, client, true)

	var wg sync.WaitGroup
	errs := make(chan error, len(messages))

	for _, message := range messages {
		wg.Add(1)
		go func(message redis.XMessage) {
			defer wg.Done()
			errs <- w.process(context.Background(), message)
		}(message)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("process duplicate: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		if err := w.recoverPending(context.Background()); err != nil {
			t.Fatalf("recover pending: %v", err)
		}

		pending, err := client.XPending(context.Background(), StreamKey, GroupName).Result()
		if err != nil {
			t.Fatal(err)
		}
		if pending.Count == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	status, err := client.HGet(context.Background(), JobPrefix+jobID, "status").Result()
	if err != nil || status != "completed" {
		t.Fatalf("expected completed status, got %q err=%v", status, err)
	}

	attempts, err := client.HGet(context.Background(), JobPrefix+jobID, "attempts").Int()
	if err != nil || attempts != 1 {
		t.Fatalf("duplicate delivery must not execute twice; attempts=%d err=%v", attempts, err)
	}

	pending, err := client.XPending(context.Background(), StreamKey, GroupName).Result()
	if err != nil || pending.Count != 0 {
		t.Fatalf("expected no pending entries, got %d err=%v", pending.Count, err)
	}
}

// TestHeartbeatRenewsLongRunningJob verifies that heartbeat renewals
// preserve a lock beyond its original TTL during long-running work.
func TestHeartbeatRenewsLongRunningJob(t *testing.T) {
	client := testRedis(t)
	w, messages, jobID := newProcessTestJob(t, client, false)
	jobKey := JobPrefix + jobID

	w.lockTTL = 300 * time.Millisecond
	w.lockRenew = 60 * time.Millisecond

	started := make(chan struct{})
	release := make(chan struct{})

	w.processJob = func(ctx context.Context, jobType, payload string) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	message := messages[0]
	// jobID was returned by newProcessTestJob.
	lockKey := lockPrefix + jobID

	result := make(chan error, 1)
	go func() {
		result <- w.process(context.Background(), message)
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not start")
	}

	// Longer than the original TTL.
	time.Sleep(450 * time.Millisecond)

	ttl, err := client.PTTL(context.Background(), lockKey).Result()
	if err != nil {
		t.Fatalf("read lock TTL: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("heartbeat failed to preserve lock, TTL=%v", ttl)
	}

	close(release)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("process long-running job: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not finish")
	}

	values, err := client.HGetAll(context.Background(), jobKey).Result()
	if err != nil {
		t.Fatal(err)
	}
	if values["status"] != "completed" {
		t.Fatalf("expected completed status, got %q", values["status"])
	}
	if values["attempts"] != "1" {
		t.Fatalf("expected one attempt, got %q", values["attempts"])
	}
}

// TestLeaseLossCancelsHandler verifies that loss of lock ownership
// cancels processing and prevents job completion.
func TestLeaseLossCancelsHandler(t *testing.T) {
	client := testRedis(t)
	w, messages, jobID := newProcessTestJob(t, client, false)
	jobKey := JobPrefix + jobID

	w.lockTTL = 2 * time.Second
	w.lockRenew = 60 * time.Millisecond

	started := make(chan struct{})
	cancelled := make(chan struct{})

	w.processJob = func(ctx context.Context, jobType, payload string) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}

	message := messages[0]
	// jobID was returned by newProcessTestJob.
	lockKey := lockPrefix + jobID

	result := make(chan error, 1)
	go func() {
		result <- w.process(context.Background(), message)
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not start")
	}

	// Simulate another actor removing the current lease.
	if err := client.Del(context.Background(), lockKey).Err(); err != nil {
		t.Fatalf("simulate lease loss: %v", err)
	}

	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("handler was not cancelled after lease loss")
	}

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected process error after lease loss")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not return after lease loss")
	}

	values, err := client.HGetAll(context.Background(), jobKey).Result()
	if err != nil {
		t.Fatal(err)
	}
	if values["status"] == "completed" {
		t.Fatal("job must not be completed after lease loss")
	}

	exists, err := client.Exists(context.Background(), lockKey).Result()
	if err != nil {
		t.Fatal(err)
	}
	if exists != 0 {
		t.Fatal("lost lock should remain absent")
	}

	if err := client.XAck(
		context.Background(), StreamKey, GroupName, message.ID,
	).Err(); err != nil {
		t.Fatalf("cleanup pending message: %v", err)
	}
}
