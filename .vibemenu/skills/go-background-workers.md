---
name: go-background-workers
description: Goroutine-based background worker pools in Go — pool pattern, graceful shutdown, backoff, dead-letter queues, rate limiting, and health checks.
origin: vibemenu
---

# Go Background Workers

Goroutine-based worker pool patterns for reliable background job processing in Go.

## When to Activate

- Implementing background job queues (`JobQueueDef` in manifest)
- Processing tasks asynchronously (email sending, webhook delivery, data pipelines)
- Building cron job executors (`CronJobDef` in manifest)
- Any producer/consumer pattern with bounded concurrency

---

## Worker Pool Pattern

The canonical pattern for bounded concurrency. Never spin up one goroutine per job in production.

```go
package worker

import (
    "context"
    "log/slog"
    "sync"
    "time"
)

type Job struct {
    ID      string
    Payload []byte
    Attempt int
}

type Pool struct {
    jobs    chan Job
    wg      sync.WaitGroup
    workers int
}

func NewPool(workers, bufSize int) *Pool {
    return &Pool{
        jobs:    make(chan Job, bufSize),
        workers: workers,
    }
}

func (p *Pool) Start(ctx context.Context, process func(context.Context, Job) error) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(ctx, i, process)
    }
}

func (p *Pool) worker(ctx context.Context, id int, process func(context.Context, Job) error) {
    defer p.wg.Done()
    for {
        select {
        case job, ok := <-p.jobs:
            if !ok {
                return // channel closed, drain complete
            }
            start := time.Now()
            slog.Info("job started", "job_id", job.ID, "worker", id)
            if err := process(ctx, job); err != nil {
                slog.Error("job failed", "job_id", job.ID, "worker", id, "error", err)
            } else {
                slog.Info("job completed", "job_id", job.ID, "worker", id,
                    "duration_ms", time.Since(start).Milliseconds())
            }
        case <-ctx.Done():
            return
        }
    }
}

// Submit enqueues a job. Blocks if the buffer is full (back-pressure).
func (p *Pool) Submit(job Job) {
    p.jobs <- job
}

// TrySubmit is a non-blocking submit. Returns false if queue is full.
func (p *Pool) TrySubmit(job Job) bool {
    select {
    case p.jobs <- job:
        return true
    default:
        return false
    }
}

// Shutdown stops accepting jobs and waits for in-flight jobs to complete.
func (p *Pool) Shutdown() {
    close(p.jobs)
    p.wg.Wait()
}
```

**Anti-patterns to avoid:**
- `go processJob(job)` in a loop — unbounded goroutine creation, no back-pressure
- Not calling `wg.Wait()` on shutdown — in-flight jobs are killed mid-execution
- Using `p.wg.Add(1)` inside the goroutine — race condition with `Wait()`

---

## Graceful Shutdown with Signal Handling

Always wire the pool to OS signal context so deployments don't drop jobs.

```go
package main

import (
    "context"
    "os/signal"
    "syscall"
)

func main() {
    // signal.NotifyContext cancels ctx on SIGTERM or SIGINT
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer cancel()

    pool := worker.NewPool(8, 100)
    pool.Start(ctx, processJob)

    // Producer loop: pull jobs from DB/queue and submit
    go producerLoop(ctx, pool)

    // Block until signal received
    <-ctx.Done()
    slog.Info("shutdown signal received, draining pool...")

    // pool.Shutdown() closes the channel and waits for in-flight jobs
    pool.Shutdown()
    slog.Info("all jobs drained, exiting")
}

func producerLoop(ctx context.Context, pool *worker.Pool) {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            jobs, _ := db.FetchPendingJobs(ctx, 20)
            for _, j := range jobs {
                pool.Submit(j)
            }
        }
    }
}
```

**Shutdown order matters:**
1. Signal arrives → `ctx.Done()` fires
2. Producer exits the `producerLoop` (no new submissions)
3. `pool.Shutdown()` closes the jobs channel
4. Workers drain remaining buffered jobs, then return
5. `wg.Wait()` unblocks — process exits cleanly

---

## Buffered vs Unbuffered Channels

```go
// Buffered — producer doesn't block until N items are queued
// Use for: async producers, bursty workloads, database pollers
jobs := make(chan Job, 100)

// Unbuffered — producer blocks until a worker picks up each job
// Use for: synchronous pipelines, when you need strict ordering guarantees
jobs := make(chan Job)
```

**Rules:**
- Buffered: acts as a queue; size determines maximum burst before back-pressure kicks in
- Too-large buffer hides a slow consumer — size should be `workers * 2` to `workers * 10` for most workloads
- Monitor queue depth in production; alert when consistently at >80% capacity
- Use `TrySubmit` with overflow logging to detect saturation without blocking producers

---

## Exponential Backoff with Jitter

Use for retries inside the process function. Jitter prevents thundering herd on shared resources.

```go
import (
    "math/rand"
    "time"
)

// Backoff returns wait duration for attempt N (0-indexed).
// Sequence: ~1s, ~2s, ~4s, ~8s, capped at 60s.
func Backoff(attempt int) time.Duration {
    base := time.Second * time.Duration(1<<attempt) // 1, 2, 4, 8, 16...
    if base > 60*time.Second {
        base = 60 * time.Second
    }
    // Add up to 50% jitter to spread retries
    jitter := time.Duration(rand.Int63n(int64(base / 2)))
    return base + jitter
}

// ProcessWithRetry wraps a job processor with retry logic.
func ProcessWithRetry(ctx context.Context, job Job, maxAttempts int,
    process func(context.Context, Job) error,
    deadLetter func(context.Context, Job, error),
) error {
    var lastErr error
    for attempt := 0; attempt < maxAttempts; attempt++ {
        if attempt > 0 {
            wait := Backoff(attempt - 1)
            slog.Info("retrying job", "job_id", job.ID, "attempt", attempt,
                "wait_ms", wait.Milliseconds())
            select {
            case <-time.After(wait):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        job.Attempt = attempt + 1
        if err := process(ctx, job); err != nil {
            lastErr = err
            slog.Error("job attempt failed", "job_id", job.ID, "attempt", attempt+1, "error", err)
            continue
        }
        return nil // success
    }
    // All attempts exhausted
    deadLetter(ctx, job, lastErr)
    return fmt.Errorf("job %s exhausted %d attempts: %w", job.ID, maxAttempts, lastErr)
}
```

---

## Dead-Letter Queue Integration

Failed jobs after max retries must never be silently dropped.

```go
// dead_letter.go

type DeadLetterJob struct {
    ID           string    `db:"id"`
    OriginalID   string    `db:"original_id"`
    Payload      []byte    `db:"payload"`
    LastError    string    `db:"last_error"`
    AttemptCount int       `db:"attempt_count"`
    FailedAt     time.Time `db:"failed_at"`
    QueueName    string    `db:"queue_name"`
}

type DeadLetterStore interface {
    Insert(ctx context.Context, job DeadLetterJob) error
    GetByID(ctx context.Context, id string) (*DeadLetterJob, error)
    List(ctx context.Context, queueName string, limit int) ([]DeadLetterJob, error)
    Delete(ctx context.Context, id string) error
}

// SendToDeadLetter writes a failed job to the dead-letter table.
func SendToDeadLetter(ctx context.Context, store DeadLetterStore, job Job, lastErr error) {
    dlj := DeadLetterJob{
        ID:           uuid.NewString(),
        OriginalID:   job.ID,
        Payload:      job.Payload,
        LastError:    lastErr.Error(),
        AttemptCount: job.Attempt,
        FailedAt:     time.Now().UTC(),
        QueueName:    "default",
    }
    if err := store.Insert(ctx, dlj); err != nil {
        // Log but don't fail — dead-letter failure must not crash the worker
        slog.Error("failed to write dead-letter job", "job_id", job.ID, "error", err)
    }
    slog.Warn("job moved to dead-letter queue",
        "job_id", job.ID, "attempts", job.Attempt, "last_error", lastErr)
}

// RequeueDeadLetter resubmits a dead-letter job for reprocessing.
// Call from an admin HTTP handler or CLI command.
func RequeueDeadLetter(ctx context.Context, store DeadLetterStore, pool *Pool, dlJobID string) error {
    dlj, err := store.GetByID(ctx, dlJobID)
    if err != nil {
        return fmt.Errorf("dead-letter lookup: %w", err)
    }
    job := Job{
        ID:      dlj.OriginalID,
        Payload: dlj.Payload,
        Attempt: 0, // reset attempt counter
    }
    pool.Submit(job)
    return store.Delete(ctx, dlJobID) // remove from dead-letter after requeue
}
```

**Schema:**
```sql
CREATE TABLE dead_letter_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_id  TEXT NOT NULL,
    payload      BYTEA NOT NULL,
    last_error   TEXT NOT NULL,
    attempt_count INT NOT NULL,
    failed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    queue_name   TEXT NOT NULL DEFAULT 'default'
);

CREATE INDEX ON dead_letter_jobs (queue_name, failed_at DESC);
```

---

## Structured Logging Per Job with `slog`

Use the standard library `log/slog` package (Go 1.21+). Never use `fmt.Println` in worker code.

```go
import "log/slog"

// At process start — log job context
slog.Info("job started",
    "job_id", job.ID,
    "queue", queueName,
    "worker", workerID,
    "attempt", job.Attempt,
)

// On failure — include full error and context
slog.Error("job failed",
    "job_id", job.ID,
    "attempt", job.Attempt,
    "error", err,
    "worker", workerID,
)

// On success — include timing
slog.Info("job completed",
    "job_id", job.ID,
    "duration_ms", elapsed.Milliseconds(),
    "worker", workerID,
)

// Add structured context via WithGroup or With
logger := slog.With("service", "payment-worker", "env", os.Getenv("APP_ENV"))
// Reuse logger throughout the worker lifecycle
```

**JSON output for production** (configure at startup):
```go
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})))
```

---

## Rate Limiting Workers

Use `golang.org/x/time/rate` to cap throughput and protect downstream services.

```go
import "golang.org/x/time/rate"

// Allow 10 jobs/second, burst of up to 20
limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 20)

func (p *Pool) worker(ctx context.Context, id int, limiter *rate.Limiter,
    process func(context.Context, Job) error,
) {
    defer p.wg.Done()
    for {
        select {
        case job, ok := <-p.jobs:
            if !ok {
                return
            }
            // Wait for rate limit token before processing
            if err := limiter.Wait(ctx); err != nil {
                // ctx cancelled
                return
            }
            _ = process(ctx, job)
        case <-ctx.Done():
            return
        }
    }
}
```

**Choosing limits:**
- Set limit at 70% of the downstream service's capacity — leaves headroom for retries
- Per-worker limit: `rate.NewLimiter(rate.Every(time.Second/jobsPerWorker), burst)`
- Global limit across all workers: pass a single `*rate.Limiter` shared by all goroutines (it's safe for concurrent use)

---

## Health Check Endpoint

Expose worker pool metrics for Kubernetes probes and observability.

```go
import (
    "encoding/json"
    "net/http"
    "sync/atomic"
)

type PoolMetrics struct {
    pool         *Pool
    processed    atomic.Int64
    failed       atomic.Int64
}

type HealthResponse struct {
    Status         string `json:"status"`
    Workers        int    `json:"workers"`
    QueueDepth     int    `json:"queue_depth"`
    QueueCapacity  int    `json:"queue_capacity"`
    JobsProcessed  int64  `json:"jobs_processed"`
    JobsFailed     int64  `json:"jobs_failed"`
}

func (m *PoolMetrics) HealthHandler(w http.ResponseWriter, r *http.Request) {
    depth := len(m.pool.jobs)
    capacity := cap(m.pool.jobs)

    status := "healthy"
    httpStatus := http.StatusOK

    // Alert if queue is >90% full — likely a consumer bottleneck
    if capacity > 0 && float64(depth)/float64(capacity) > 0.9 {
        status = "degraded"
        httpStatus = http.StatusServiceUnavailable
    }

    resp := HealthResponse{
        Status:        status,
        Workers:       m.pool.workers,
        QueueDepth:    depth,
        QueueCapacity: capacity,
        JobsProcessed: m.processed.Load(),
        JobsFailed:    m.failed.Load(),
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(resp)
}

// Register in your HTTP mux:
// mux.HandleFunc("/health/workers", metrics.HealthHandler)
```

---

## Complete Wiring Example

```go
func Run() error {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer cancel()

    db := mustConnectDB()
    dlStore := postgres.NewDeadLetterStore(db)
    limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 20) // 10 rps

    pool := worker.NewPool(8, 200)
    metrics := &worker.PoolMetrics{Pool: pool}

    process := func(ctx context.Context, job worker.Job) error {
        if err := limiter.Wait(ctx); err != nil {
            return err
        }
        return handleJob(ctx, job)
    }

    pool.Start(ctx, func(ctx context.Context, job worker.Job) error {
        return worker.ProcessWithRetry(ctx, job, 3, process,
            func(ctx context.Context, j worker.Job, err error) {
                worker.SendToDeadLetter(ctx, dlStore, j, err)
                metrics.Failed.Add(1)
            },
        )
    })

    // HTTP health server
    mux := http.NewServeMux()
    mux.HandleFunc("/health/workers", metrics.HealthHandler)
    srv := &http.Server{Addr: ":8081", Handler: mux}
    go srv.ListenAndServe()

    go producerLoop(ctx, db, pool)

    <-ctx.Done()
    pool.Shutdown()
    srv.Shutdown(context.Background())
    return nil
}
```
