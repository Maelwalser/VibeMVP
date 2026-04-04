---
name: jobs-laravel-queues
description: Laravel Queues — PHP native queue system with Redis/database drivers, Supervisor for workers, job chaining, rate limiting, and Horizon monitoring.
origin: vibemenu
---

# Laravel Queues

Laravel provides a unified queue API across multiple backends (Redis, database, SQS, sync). Jobs are PHP classes; workers process them in the background via `artisan queue:work`.

## When to Activate

- Background email/notification sending in Laravel applications
- Any slow operation that shouldn't block an HTTP response (image resize, PDF generation)
- Reliable async processing that survives PHP-FPM restarts
- Rate-limited API calls to third-party services
- Pipelines of dependent jobs (job chaining)

## Queue Connections

```php
// config/queue.php — default connection from QUEUE_CONNECTION env var
'default' => env('QUEUE_CONNECTION', 'redis'),

'connections' => [
    'sync' => ['driver' => 'sync'],          // runs immediately inline (for testing)
    'database' => [
        'driver' => 'database',
        'table' => 'jobs',
        'queue' => 'default',
        'retry_after' => 90,                  // seconds before considering job lost
    ],
    'redis' => [
        'driver' => 'redis',
        'connection' => 'default',            // references config/database.php redis connection
        'queue' => env('REDIS_QUEUE', 'default'),
        'retry_after' => 90,
        'block_for' => null,
    ],
    'sqs' => [
        'driver' => 'sqs',
        'key' => env('AWS_ACCESS_KEY_ID'),
        'secret' => env('AWS_SECRET_ACCESS_KEY'),
        'prefix' => env('SQS_PREFIX', 'https://sqs.us-east-1.amazonaws.com/your-account-id'),
        'queue' => env('SQS_QUEUE', 'default'),
        'suffix' => env('SQS_SUFFIX'),
        'region' => env('AWS_DEFAULT_REGION', 'us-east-1'),
    ],
],
```

### Redis Driver Requirements

```bash
composer require predis/predis
# or install phpredis PHP extension (faster)
```

```env
QUEUE_CONNECTION=redis
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_QUEUE=default
```

## Creating a Job

```bash
php artisan make:job ProcessPodcast
```

```php
<?php

namespace App\Jobs;

use App\Models\Podcast;
use App\Services\AudioTranscoder;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

class ProcessPodcast implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    // Max attempts before marking as failed
    public int $tries = 3;

    // Seconds job can run before being killed
    public int $timeout = 120;

    // Seconds to wait before retrying (array = per-attempt backoff)
    public array $backoff = [10, 60, 300];

    // Delete the job if the related model no longer exists
    public bool $deleteWhenMissingModels = true;

    public function __construct(
        private readonly Podcast $podcast,
        private readonly string $format = 'mp3',
    ) {}

    public function handle(AudioTranscoder $transcoder): void
    {
        // Dependencies are auto-resolved from the service container
        $transcoder->transcode($this->podcast->file_path, $this->format);

        $this->podcast->update(['status' => 'processed']);
    }

    // Called when all retry attempts fail
    public function failed(\Throwable $exception): void
    {
        $this->podcast->update(['status' => 'failed']);
        \Log::error('Podcast processing failed', [
            'podcast_id' => $this->podcast->id,
            'error' => $exception->getMessage(),
        ]);
    }
}
```

## Dispatching Jobs

```php
// Fire-and-forget (immediate, default queue)
ProcessPodcast::dispatch($podcast);

// Alternative syntax
dispatch(new ProcessPodcast($podcast));

// Specific queue
ProcessPodcast::dispatch($podcast)->onQueue('high');

// Delayed dispatch (run after 10 minutes)
ProcessPodcast::dispatch($podcast)->delay(now()->addMinutes(10));

// Specific connection
ProcessPodcast::dispatch($podcast)->onConnection('sqs');

// Conditionally dispatch
ProcessPodcast::dispatchIf($podcast->needs_processing, $podcast);
ProcessPodcast::dispatchUnless($podcast->is_processed, $podcast);

// Dispatch synchronously (useful in tests or for small jobs)
ProcessPodcast::dispatchSync($podcast);
```

## Job Chaining

```php
use Illuminate\Support\Facades\Bus;

// Jobs run sequentially — next job only starts if previous succeeds
Bus::chain([
    new ProcessPodcast($podcast),
    new OptimizeAudio($podcast),
    new PublishPodcast($podcast),
])->dispatch();

// With catch — runs if any job in the chain fails
Bus::chain([
    new ProcessPodcast($podcast),
    new PublishPodcast($podcast),
])->catch(function (\Throwable $e) use ($podcast) {
    $podcast->update(['status' => 'chain_failed']);
})->dispatch();
```

## Job Batching (Laravel 8+)

```php
use Illuminate\Support\Facades\Bus;
use Illuminate\Bus\Batch;

$batch = Bus::batch([
    new ProcessPodcast($podcasts[0]),
    new ProcessPodcast($podcasts[1]),
    new ProcessPodcast($podcasts[2]),
])->then(function (Batch $batch) {
    // All jobs completed successfully
    \Log::info("Batch {$batch->id} complete");
})->catch(function (Batch $batch, \Throwable $e) {
    // First job failure
    \Log::error("Batch failed: {$e->getMessage()}");
})->finally(function (Batch $batch) {
    // Always runs (success or failure)
})->name('Process Podcasts')->dispatch();

// Check batch status
$batch = Bus::findBatch($batchId);
$batch->totalJobs;      // total jobs
$batch->processedJobs(); // completed + failed
$batch->failedJobs;     // count of failures
$batch->progress();     // 0-100 percentage
```

## Rate Limiting

```php
use Illuminate\Queue\Middleware\RateLimited;
use Illuminate\Support\Facades\RateLimiter;

// Define rate limiter in App\Providers\AppServiceProvider::boot()
RateLimiter::for('stripe-api', function ($job) {
    return Limit::perMinute(10); // max 10 Stripe calls/minute
});

// Apply in job class
public function middleware(): array
{
    return [new RateLimited('stripe-api')];
}
```

## Preventing Overlapping Execution

```php
use Illuminate\Queue\Middleware\WithoutOverlapping;

// Only one instance of this job for a given resource ID runs at a time
public function middleware(): array
{
    return [
        (new WithoutOverlapping($this->podcast->id))
            ->releaseAfter(60)      // retry after 60s if locked
            ->expireAfter(300),     // release lock if job takes > 5 min
    ];
}
```

## Failed Jobs

```bash
# Create the failed jobs table
php artisan queue:failed-table
php artisan migrate

# List failed jobs
php artisan queue:failed

# Retry a specific failed job
php artisan queue:retry 5

# Retry all failed jobs
php artisan queue:retry all

# Delete a failed job
php artisan queue:forget 5

# Delete all failed jobs
php artisan queue:flush
```

```php
// config/queue.php — where failed jobs are stored
'failed' => [
    'driver' => env('QUEUE_FAILED_DRIVER', 'database-uuids'),
    'database' => env('DB_CONNECTION', 'mysql'),
    'table' => 'failed_jobs',
],
```

## Running Workers

```bash
# Development — processes one job then exits (good for debugging)
php artisan queue:work --once

# Development — long-running worker, stops on code change
php artisan queue:work

# Production — multiple queues with priority (high first)
php artisan queue:work redis --queue=high,default,low --tries=3 --timeout=90

# Memory limit (restart worker if it exceeds 128MB)
php artisan queue:work --memory=128

# Max time before worker self-restarts (avoids memory leaks)
php artisan queue:work --max-time=3600
```

## Supervisor Config (Production)

```ini
; /etc/supervisor/conf.d/laravel-worker.conf

[program:laravel-worker]
process_name=%(program_name)s_%(process_num)02d
command=php /var/www/app/artisan queue:work redis --queue=high,default --sleep=3 --tries=3 --max-time=3600
directory=/var/www/app
autostart=true
autorestart=true
stopasgroup=true
killasgroup=true
user=www-data
numprocs=8                          ; spawn 8 worker processes
redirect_stderr=true
stdout_logfile=/var/log/laravel-worker.log
stopwaitsecs=3600                   ; wait up to 1 hour for job to finish before killing
```

```bash
# Apply config changes
sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start laravel-worker:*

# Gracefully restart workers after code deployment
php artisan queue:restart
# (workers finish current job then exit; Supervisor restarts them)
```

## Horizon (Redis Queue Monitoring)

```bash
composer require laravel/horizon
php artisan horizon:install
php artisan migrate
```

```php
// config/horizon.php — configure queue workers and balancing
'environments' => [
    'production' => [
        'supervisor-1' => [
            'maxProcesses' => 20,
            'balanceMaxShift' => 1,
            'balanceCooldown' => 3,
        ],
    ],
    'local' => [
        'supervisor-1' => [
            'maxProcesses' => 3,
        ],
    ],
],
```

```bash
# Start Horizon (replaces queue:work in Redis-backed apps)
php artisan horizon

# Dashboard available at /horizon (restrict via HorizonServiceProvider)
```

## Testing

```php
use Illuminate\Support\Facades\Queue;

class PodcastControllerTest extends TestCase
{
    public function test_upload_dispatches_process_job(): void
    {
        Queue::fake(); // intercept all dispatches

        $this->post('/podcasts', ['file' => UploadedFile::fake()->create('ep.mp3')]);

        Queue::assertPushed(ProcessPodcast::class, function ($job) {
            return $job->podcast->title === 'My Podcast';
        });

        Queue::assertPushedOn('high', ProcessPodcast::class);
        Queue::assertNotPushed(PublishPodcast::class);
    }
}
```

## Anti-Patterns

```php
// ❌ BAD: Passing full Eloquent models without SerializesModels trait
// Model may be stale by the time the job runs
class BadJob implements ShouldQueue {
    public function __construct(public Podcast $podcast) {} // no SerializesModels!
}

// ✅ GOOD: SerializesModels re-fetches the model fresh from DB when job runs
class GoodJob implements ShouldQueue {
    use SerializesModels;
    public function __construct(public Podcast $podcast) {}
}

// ❌ BAD: Huge payloads in job constructor (stored in queue)
new ProcessPodcast($largeArrayOfData);

// ✅ GOOD: Pass an ID, load data inside handle()
new ProcessPodcast($podcastId);

// ❌ BAD: No timeout — worker hangs forever on stuck job
public int $timeout = 0;

// ✅ GOOD: Always set a reasonable timeout
public int $timeout = 120;

// ❌ BAD: Using sync driver in production (blocks HTTP response)
QUEUE_CONNECTION=sync

// ✅ GOOD: Redis or database in production
QUEUE_CONNECTION=redis
```
