---
name: jobs-oban
description: Oban — Elixir/Ecto background job processing with PostgreSQL, unique jobs, cron scheduling, telemetry, and testing utilities.
origin: vibemenu
---

# Oban — Elixir Background Jobs

Oban uses PostgreSQL as a durable job queue. Jobs are Elixir structs stored in a `oban_jobs` table; workers are supervised GenServers. Reliable, observable, and testable.

## When to Activate

- Background job processing in Elixir/Phoenix applications
- Scheduled/recurring tasks (cron) without external schedulers
- Guaranteed at-least-once delivery backed by PostgreSQL
- Unique job enforcement (prevent duplicate work)
- High-throughput queue processing with configurable concurrency

## Installation

```elixir
# mix.exs
defp deps do
  [
    {:oban, "~> 2.17"},
    # already have ecto_sql and postgrex if using Phoenix
  ]
end
```

```bash
mix deps.get
mix ecto.gen.migration add_oban_jobs_table
```

```elixir
# Generated migration — replace with Oban's migration helper
defmodule MyApp.Repo.Migrations.AddObanJobsTable do
  use Ecto.Migration

  def up, do: Oban.Migration.up()
  def down, do: Oban.Migration.down(version: 1)
end
```

```bash
mix ecto.migrate
```

## Configuration

```elixir
# config/config.exs — base configuration
config :my_app, Oban,
  repo: MyApp.Repo,
  queues: [
    default: 10,      # 10 concurrent workers
    mailers: 20,      # 20 concurrent workers for email
    media: 5,
    critical: 50,
  ],
  plugins: [
    {Oban.Plugins.Pruner, max_age: 60 * 60 * 24 * 7}, # keep completed jobs 7 days
    {Oban.Plugins.Reindexer, schedule: "@weekly"},      # maintain index health
    {Oban.Plugins.Stager, interval: 1_000},             # move scheduled → available
    {Oban.Plugins.Cron,
      crontab: [
        {"0 * * * *",   MyApp.Workers.HourlySyncWorker},
        {"0 2 * * *",   MyApp.Workers.DailyCleanupWorker},
        {"0 8 * * MON", MyApp.Workers.WeeklyReportWorker},
      ]},
  ]

# config/runtime.exs — disable in test (use Oban.Testing instead)
config :my_app, Oban, testing: :inline  # or :disabled
```

## Supervisor Setup

```elixir
# lib/my_app/application.ex
defmodule MyApp.Application do
  use Application

  def start(_type, _args) do
    children = [
      MyApp.Repo,
      MyAppWeb.Endpoint,
      {Oban, Application.fetch_env!(:my_app, Oban)},  # add after Repo
    ]

    opts = [strategy: :one_for_one, name: MyApp.Supervisor]
    Supervisor.start_link(children, opts)
  end
end
```

## Defining a Worker

```elixir
defmodule MyApp.Workers.WelcomeEmailWorker do
  use Oban.Worker,
    queue: :mailers,
    max_attempts: 3,
    tags: ["email", "onboarding"]

  @impl Oban.Worker
  def perform(%Oban.Job{args: %{"user_id" => user_id} = args}) do
    user = MyApp.Accounts.get_user!(user_id)

    case MyApp.Mailer.send_welcome(user) do
      :ok ->
        :ok

      {:error, reason} ->
        # Returning {:error, reason} marks the job as failed and schedules retry
        {:error, reason}
    end
  end
end
```

### Return Values from `perform/1`

| Return | Behavior |
|--------|----------|
| `:ok` | Job succeeded, marked `completed` |
| `{:ok, value}` | Job succeeded (value ignored) |
| `:discard` | Don't retry, mark `discarded` |
| `{:discard, reason}` | Don't retry, log reason |
| `{:error, reason}` | Retry according to `max_attempts` |
| `{:snooze, seconds}` | Delay retry by N seconds |
| raises exception | Treated as `{:error, exception}` |

## Inserting Jobs

```elixir
# Basic insert
%{user_id: user.id}
|> MyApp.Workers.WelcomeEmailWorker.new()
|> Oban.insert()

# Insert with options
%{report_id: report.id}
|> MyApp.Workers.ReportWorker.new(queue: :media, priority: 0)
|> Oban.insert()

# Insert inside an Ecto transaction (atomically with DB write)
Ecto.Multi.new()
|> Ecto.Multi.insert(:user, User.changeset(%User{}, attrs))
|> Oban.insert(:welcome_job, fn %{user: user} ->
  MyApp.Workers.WelcomeEmailWorker.new(%{user_id: user.id})
end)
|> MyApp.Repo.transaction()
```

## Scheduling

```elixir
# Run 5 minutes from now
%{order_id: order.id}
|> MyApp.Workers.ExpireOrderWorker.new(schedule_in: 300)
|> Oban.insert()

# Run at a specific datetime
scheduled_at = ~U[2025-01-01 08:00:00Z]

%{report_id: report.id}
|> MyApp.Workers.ReportWorker.new(scheduled_at: scheduled_at)
|> Oban.insert()
```

## Unique Jobs (Deduplication)

```elixir
defmodule MyApp.Workers.SyncWorker do
  use Oban.Worker,
    queue: :default,
    unique: [
      period: 60,                          # deduplicate within 60 seconds
      fields: [:args, :queue, :worker],    # uniqueness key fields
      states: [:available, :scheduled, :executing],
    ]

  def perform(%Oban.Job{args: %{"account_id" => account_id}}) do
    MyApp.Sync.run(account_id)
    :ok
  end
end

# Inserting a duplicate returns {:ok, %Oban.Job{conflict?: true}}
{:ok, job} = Oban.insert(SyncWorker.new(%{account_id: 42}))
job.conflict? # => true if a duplicate already existed
```

## Priority

```elixir
# Priority 0 = highest, 3 = lowest (default)
%{user_id: user.id}
|> MyApp.Workers.NotificationWorker.new(priority: 0)  # highest priority
|> Oban.insert()

%{user_id: user.id}
|> MyApp.Workers.BulkExportWorker.new(priority: 3)    # lowest priority
|> Oban.insert()
```

## Cron Scheduling

```elixir
# In Oban config plugins:
{Oban.Plugins.Cron,
  crontab: [
    # Standard cron syntax: minute hour day month weekday
    {"*/15 * * * *",  MyApp.Workers.HeartbeatWorker},
    {"0 * * * *",     MyApp.Workers.HourlySyncWorker},
    {"0 2 * * *",     MyApp.Workers.DailyCleanupWorker},
    {"@daily",        MyApp.Workers.DailyReportWorker},
    {"@weekly",       MyApp.Workers.WeeklyDigestWorker},

    # With extra options
    {"0 8 * * MON", MyApp.Workers.WeeklyReportWorker,
     args: %{format: "pdf"}, queue: :media},
  ]
}
```

## Telemetry Events

Oban emits telemetry events compatible with `Telemetry.Metrics` and `Logger`:

```elixir
# lib/my_app/oban_telemetry.ex
defmodule MyApp.ObanTelemetry do
  def setup do
    events = [
      [:oban, :job, :start],
      [:oban, :job, :stop],
      [:oban, :job, :exception],
    ]

    :telemetry.attach_many("oban-logger", events, &handle_event/4, nil)
  end

  def handle_event([:oban, :job, :start], _measurements, meta, _config) do
    Logger.info("Job started", worker: meta.worker, queue: meta.queue, id: meta.id)
  end

  def handle_event([:oban, :job, :stop], measurements, meta, _config) do
    Logger.info("Job complete",
      worker: meta.worker,
      duration_ms: System.convert_time_unit(measurements.duration, :native, :millisecond),
      state: meta.state
    )
  end

  def handle_event([:oban, :job, :exception], _measurements, meta, _config) do
    Logger.error("Job failed",
      worker: meta.worker,
      error: inspect(meta.reason),
      attempt: meta.attempt,
      max_attempts: meta.max_attempts
    )
  end
end

# Call in application.ex start/2
MyApp.ObanTelemetry.setup()
```

## Cancelling and Retrying Jobs

```elixir
# Cancel a specific job (sets state to :cancelled)
Oban.cancel_job(job_id)

# Retry a failed/discarded job immediately
Oban.retry_job(job_id)

# Drain a queue in tests (runs all available jobs synchronously)
Oban.drain_queue(queue: :mailers)
```

## Error Handling Patterns

```elixir
defmodule MyApp.Workers.ResilientWorker do
  use Oban.Worker, queue: :default, max_attempts: 5

  @impl Oban.Worker
  def perform(%Oban.Job{args: args, attempt: attempt}) do
    case do_work(args) do
      :ok ->
        :ok

      {:error, :rate_limited} ->
        # Snooze for exponential backoff (not counting as failed attempt)
        backoff = :math.pow(2, attempt) |> round()
        {:snooze, backoff}

      {:error, :permanently_unavailable} ->
        # Don't retry — resource is gone
        {:discard, "Resource permanently unavailable"}

      {:error, reason} ->
        # Retry up to max_attempts
        {:error, reason}
    end
  end
end
```

## Testing with `Oban.Testing`

```elixir
# test/test_helper.exs — configure inline mode so jobs run synchronously
# Already set in config/runtime.exs: config :my_app, Oban, testing: :inline

# In your test module
defmodule MyApp.AccountsTest do
  use MyApp.DataCase, async: true
  use Oban.Testing, repo: MyApp.Repo

  test "creating a user enqueues a welcome email" do
    {:ok, user} = Accounts.create_user(%{email: "alice@example.com"})

    assert_enqueued(
      worker: MyApp.Workers.WelcomeEmailWorker,
      args: %{user_id: user.id}
    )
  end

  test "welcome email job sends the email" do
    user = insert(:user)

    # Run the job directly and assert on its return value
    assert :ok = perform_job(MyApp.Workers.WelcomeEmailWorker, %{user_id: user.id})
  end

  test "no jobs enqueued for banned users" do
    insert(:user, status: :banned)
    refute_enqueued(worker: MyApp.Workers.WelcomeEmailWorker)
  end
end
```

## Pruning Old Jobs

```elixir
# Keep only last 7 days of completed jobs
{Oban.Plugins.Pruner,
  max_age: 60 * 60 * 24 * 7,   # 7 days in seconds
  interval: :timer.minutes(30)  # check every 30 minutes
}

# Keep up to 1000 completed jobs (alternative)
{Oban.Plugins.Pruner, max_len: 1000}
```

## Anti-Patterns

```elixir
# ❌ BAD: Storing large data in job args (stored in PostgreSQL jsonb column)
%{full_document: very_large_map}
|> MyWorker.new()
|> Oban.insert()

# ✅ GOOD: Store only the ID, load inside perform/1
%{document_id: doc.id}
|> MyWorker.new()
|> Oban.insert()

# ❌ BAD: Catching all errors and returning :ok (hides failures)
def perform(%Oban.Job{args: args}) do
  try do
    do_work(args)
    :ok
  rescue
    _ -> :ok  # swallows failures silently
  end
end

# ✅ GOOD: Let Oban handle retries — return {:error, reason}
def perform(%Oban.Job{args: args}) do
  case do_work(args) do
    :ok -> :ok
    {:error, reason} -> {:error, reason}
  end
end

# ❌ BAD: Long-polling or sleeping inside a job (blocks the worker)
def perform(%Oban.Job{args: %{"id" => id}}) do
  Process.sleep(30_000)  # blocks one worker slot for 30 seconds
  :ok
end

# ✅ GOOD: Use {:snooze, seconds} to release the worker and retry later
def perform(%Oban.Job{args: %{"id" => id}}) do
  if MyApp.Service.ready?(id) do
    :ok
  else
    {:snooze, 30}  # worker is free; Oban retries in 30s
  end
end
```
