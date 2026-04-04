---
name: jobs-hangfire
description: Hangfire background job framework for .NET — fire-and-forget, delayed, recurring, and chained jobs with SQL Server or Redis storage.
origin: vibemenu
---

# Hangfire .NET Background Jobs

Hangfire provides persistent background job processing for .NET applications. Jobs survive application restarts because state is stored in SQL Server or Redis.

## When to Activate

- Background processing in ASP.NET Core applications
- Scheduled/recurring tasks (cron-like) without external schedulers
- Delayed execution (send email in 5 minutes, expire trial in 30 days)
- Fire-and-forget side effects that must survive restarts
- Replacing Windows Services or Azure WebJobs for background work

## NuGet Packages

```xml
<!-- Core + SQL Server storage (most common) -->
<PackageReference Include="Hangfire.Core" Version="1.8.*" />
<PackageReference Include="Hangfire.SqlServer" Version="1.8.*" />
<PackageReference Include="Hangfire.AspNetCore" Version="1.8.*" />

<!-- Redis storage (alternative — lower latency, no DB dependency) -->
<PackageReference Include="Hangfire.Redis.StackExchange" Version="1.9.*" />

<!-- Dashboard authorization helper -->
<PackageReference Include="Hangfire.Dashboard.Authorization" Version="3.*" />
```

## Program.cs Setup

### SQL Server Storage

```csharp
using Hangfire;
using Hangfire.SqlServer;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddHangfire(config => config
    .SetDataCompatibilityLevel(CompatibilityLevel.Version_180)
    .UseSimpleAssemblyNameTypeSerializer()
    .UseRecommendedSerializerSettings()
    .UseSqlServerStorage(
        builder.Configuration.GetConnectionString("HangfireDb"),
        new SqlServerStorageOptions
        {
            CommandBatchMaxTimeout = TimeSpan.FromMinutes(5),
            SlidingInvisibilityTimeout = TimeSpan.FromMinutes(5),
            QueuePollInterval = TimeSpan.Zero,
            UseRecommendedIsolationLevel = true,
            DisableGlobalLocks = true,
        }));

// Add the Hangfire server (processes jobs)
builder.Services.AddHangfireServer(options =>
{
    options.Queues = new[] { "critical", "default", "low" };
    options.WorkerCount = Environment.ProcessorCount * 2;
});

var app = builder.Build();

// Dashboard — restrict to admins in production
app.UseHangfireDashboard("/hangfire", new DashboardOptions
{
    Authorization = new[] { new HangfireAuthorizationFilter() }
});
```

### Redis Storage

```csharp
builder.Services.AddHangfire(config => config
    .SetDataCompatibilityLevel(CompatibilityLevel.Version_180)
    .UseSimpleAssemblyNameTypeSerializer()
    .UseRecommendedSerializerSettings()
    .UseRedisStorage(
        builder.Configuration.GetConnectionString("Redis"),
        new RedisStorageOptions
        {
            Prefix = "hangfire:",
            Database = 1,
            ExpiryCheckInterval = TimeSpan.FromHours(1),
        }));
```

## Job Types

### Fire-and-Forget

```csharp
// Enqueue immediately — runs as soon as a worker is free
var jobId = BackgroundJob.Enqueue(() => EmailService.SendWelcomeEmail(userId));

// With a specific queue
var jobId = BackgroundJob.Enqueue<IEmailService>(
    queue: "critical",
    methodCall: svc => svc.SendPasswordReset(userId, token));
```

### Delayed Jobs

```csharp
// Run once after a delay
BackgroundJob.Schedule(
    () => SubscriptionService.ExpireTrial(userId),
    TimeSpan.FromDays(30));

// Run at a specific time
BackgroundJob.Schedule(
    () => ReportService.SendMonthlyReport(orgId),
    new DateTimeOffset(2025, 1, 1, 8, 0, 0, TimeSpan.Zero));
```

### Recurring Jobs

```csharp
// Register/update recurring job by ID (idempotent — safe to call on startup)
RecurringJob.AddOrUpdate(
    "daily-cleanup",
    () => CleanupService.PurgeExpiredSessions(),
    Cron.Daily());

// Custom cron expression
RecurringJob.AddOrUpdate(
    "hourly-sync",
    () => SyncService.SyncExternalData(),
    "0 * * * *"); // every hour at :00

// Specific queue
RecurringJob.AddOrUpdate(
    "weekly-report",
    () => ReportService.GenerateWeekly(),
    Cron.Weekly(DayOfWeek.Monday, 8, 0),
    new RecurringJobOptions { QueueName = "low" });

// Remove a recurring job
RecurringJob.RemoveIfExists("daily-cleanup");

// Trigger immediately (useful for testing)
RecurringJob.TriggerJob("daily-cleanup");
```

### Continuations (Job Chaining)

```csharp
// Run job B only after job A completes successfully
var jobA = BackgroundJob.Enqueue(() => DataService.ExtractData(reportId));
BackgroundJob.ContinueJobWith(jobA, () => DataService.TransformData(reportId));

// Chain three jobs
var step1 = BackgroundJob.Enqueue(() => Pipeline.Step1(id));
var step2 = BackgroundJob.ContinueJobWith(step1, () => Pipeline.Step2(id));
BackgroundJob.ContinueJobWith(step2, () => Pipeline.Step3(id));
```

## Job Implementation Best Practices

### DI-Injected Job Classes

```csharp
// Prefer class-based jobs over static methods — supports DI
public class EmailService
{
    private readonly IMailSender _mailer;
    private readonly ILogger<EmailService> _logger;

    public EmailService(IMailSender mailer, ILogger<EmailService> logger)
    {
        _mailer = mailer;
        _logger = logger;
    }

    [AutomaticRetry(Attempts = 3, DelaysInSeconds = new[] { 60, 300, 3600 })]
    [Queue("critical")]
    public async Task SendWelcomeEmail(int userId)
    {
        _logger.LogInformation("Sending welcome email to user {UserId}", userId);
        await _mailer.SendAsync(userId, "Welcome!");
    }
}

// Enqueue with DI-resolved instance
BackgroundJob.Enqueue<EmailService>(svc => svc.SendWelcomeEmail(userId));
```

### Retry Policies

```csharp
// Default: 10 automatic retries with increasing delays
// Override per-method:

[AutomaticRetry(Attempts = 5)]
public async Task ProcessPayment(Guid paymentId) { ... }

// Delete the job after all retries fail (don't keep in failed state)
[AutomaticRetry(Attempts = 3, OnAttemptsExceeded = AttemptsExceededAction.Delete)]
public async Task SendNotification(string userId) { ... }

// Fail immediately (no retries)
[AutomaticRetry(Attempts = 0)]
public async Task CriticalWriteOnce(Guid id) { ... }

// Global retry policy (in AddHangfire configuration)
GlobalJobFilters.Filters.Add(new AutomaticRetryAttribute { Attempts = 3 });
```

### Job Cancellation

```csharp
// Accept IJobCancellationToken for graceful shutdown support
public async Task LongRunningExport(
    int reportId,
    IJobCancellationToken cancellationToken)
{
    var items = await _repo.GetReportItems(reportId);

    foreach (var item in items)
    {
        // Check for cancellation (e.g., server shutting down)
        cancellationToken.ThrowIfCancellationRequested();

        await ProcessItem(item);
    }
}
```

### Idempotency (Critical for Retried Jobs)

```csharp
// Jobs can run more than once — make them idempotent
[AutomaticRetry(Attempts = 3)]
public async Task ChargeSubscription(Guid subscriptionId)
{
    var sub = await _repo.GetSubscription(subscriptionId);

    // Guard: check if already charged this period
    if (sub.LastChargedAt >= DateTimeOffset.UtcNow.AddMonths(-1))
    {
        _logger.LogInformation("Subscription {Id} already charged, skipping", subscriptionId);
        return;
    }

    await _paymentGateway.Charge(sub.CustomerId, sub.Amount);
    await _repo.UpdateLastCharged(subscriptionId, DateTimeOffset.UtcNow);
}
```

## Failed Job Management

```csharp
// Re-enqueue a failed job programmatically
BackgroundJob.Requeue(jobId);

// Delete a job
BackgroundJob.Delete(jobId);

// Get job state
var job = JobStorage.Current.GetMonitoringApi().JobDetails(jobId);

// Batch-requeue all failed jobs (run from admin endpoint)
using var connection = JobStorage.Current.GetConnection();
var failedJobs = JobStorage.Current.GetMonitoringApi().FailedJobs(0, 1000);
foreach (var (id, _) in failedJobs)
{
    BackgroundJob.Requeue(id);
}
```

## Dashboard Authorization Filter

```csharp
// Restrict dashboard to authenticated admins
public class HangfireAuthorizationFilter : IDashboardAuthorizationFilter
{
    public bool Authorize(DashboardContext context)
    {
        var httpContext = context.GetHttpContext();

        // Only allow authenticated users with Admin role
        return httpContext.User.Identity?.IsAuthenticated == true
            && httpContext.User.IsInRole("Admin");
    }
}
```

## SQL Server Schema

Hangfire creates its own schema on first run. Point it at a dedicated database or schema:

```csharp
// Dedicated schema (recommended — keeps Hangfire tables separate)
.UseSqlServerStorage(connectionString, new SqlServerStorageOptions
{
    SchemaName = "hangfire", // default is "HangFire"
})
```

```sql
-- Hangfire creates these tables automatically:
-- HangFire.Job, HangFire.State, HangFire.Counter
-- HangFire.AggregatedCounter, HangFire.Hash
-- HangFire.List, HangFire.Set, HangFire.Schema
-- HangFire.Server, HangFire.JobQueue
```

## Logging

Hangfire integrates with `Microsoft.Extensions.Logging` automatically when using `Hangfire.AspNetCore`. No extra configuration needed:

```csharp
// In appsettings.json — control Hangfire log verbosity
{
  "Logging": {
    "LogLevel": {
      "Hangfire": "Warning"
    }
  }
}
```

## Anti-Patterns

```csharp
// ❌ BAD: Capturing EF DbContext or scoped services in closure
var ctx = serviceProvider.GetRequiredService<AppDbContext>();
BackgroundJob.Enqueue(() => ctx.Users.ToListAsync()); // ctx is disposed before job runs

// ✅ GOOD: Pass only IDs — resolve services fresh inside job method
BackgroundJob.Enqueue<UserService>(svc => svc.ProcessUser(userId));

// ❌ BAD: Large objects in job arguments (serialized to DB)
BackgroundJob.Enqueue(() => ProcessReport(hugeReportObject));

// ✅ GOOD: Pass identifier, load inside job
BackgroundJob.Enqueue<ReportService>(svc => svc.ProcessReport(reportId));

// ❌ BAD: Fire-and-forget without Hangfire (lost on crash)
_ = Task.Run(() => SendEmail(userId));

// ✅ GOOD: Persistent job
BackgroundJob.Enqueue<EmailService>(svc => svc.SendWelcomeEmail(userId));

// ❌ BAD: Non-idempotent job (charges twice on retry)
public async Task ChargeCard(string cardId, decimal amount)
{
    await _stripe.Charge(cardId, amount); // no duplicate check
}
```

## Testing Hangfire Jobs

```csharp
// Test the job method directly — no Hangfire infrastructure needed
public class EmailServiceTests
{
    [Fact]
    public async Task SendWelcomeEmail_SendsToCorrectUser()
    {
        var mailer = Substitute.For<IMailSender>();
        var logger = NullLogger<EmailService>.Instance;
        var svc = new EmailService(mailer, logger);

        await svc.SendWelcomeEmail(42);

        await mailer.Received(1).SendAsync(42, Arg.Any<string>());
    }
}
```
