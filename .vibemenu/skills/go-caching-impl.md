---
name: go-caching-impl
description: Redis/Valkey caching implementation in Go — client setup, key naming, get-or-set, singleflight, two-level cache, invalidation, bulk SCAN, and testing with miniredis.
origin: vibemenu
---

# Go Caching Implementation

Redis/Valkey caching patterns for production Go services. Covers `go-redis/v9`, cache stampede prevention, invalidation, and testability.

## When to Activate

- Manifest `data.caching` is configured (Redis, Valkey, Memcached, DragonflyDB)
- Adding a read-through or write-through cache layer on top of database repositories
- Implementing session storage or rate-limit counters
- Optimizing hot-path queries that hit the database repeatedly

---

## Client Setup with `go-redis/v9`

```go
// cache/client.go
package cache

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

func NewClient() *redis.Client {
    rdb := redis.NewClient(&redis.Options{
        Addr:         os.Getenv("REDIS_ADDR"),     // "localhost:6379"
        Password:     os.Getenv("REDIS_PASSWORD"), // "" for no auth
        DB:           0,
        PoolSize:     10,               // max open connections per goroutine-group
        MinIdleConns: 2,               // keep warm connections ready
        DialTimeout:  3 * time.Second,
        ReadTimeout:  2 * time.Second,
        WriteTimeout: 2 * time.Second,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := rdb.Ping(ctx).Err(); err != nil {
        slog.Error("redis connection failed", "error", err)
        // Fail fast at startup — misconfigured cache is worse than no cache
        panic(fmt.Sprintf("redis: %v", err))
    }

    slog.Info("redis connected", "addr", os.Getenv("REDIS_ADDR"))
    return rdb
}

// Cluster setup for Redis Cluster / ElastiCache Cluster Mode
func NewClusterClient() *redis.ClusterClient {
    return redis.NewClusterClient(&redis.ClusterOptions{
        Addrs:    strings.Split(os.Getenv("REDIS_CLUSTER_ADDRS"), ","),
        Password: os.Getenv("REDIS_PASSWORD"),
        PoolSize: 10,
    })
}
```

**Anti-patterns:**
- Never use `redis.NewClient` without setting `DialTimeout` — hangs the service if Redis is down
- Never store `*redis.Client` as a package-level variable — inject it as a dependency
- For Valkey (Redis-compatible fork), same driver works; just point `REDIS_ADDR` to the Valkey endpoint

---

## Cache Key Naming Convention

Consistent key naming enables targeted invalidation and prevents collisions between services.

```
Pattern:  {service}:{version}:{entity}:{id}
Examples:
  user-api:v1:user:abc-123
  order-api:v1:order:789
  user-api:v1:user-list:org:xyz (for list queries)
  user-api:v1:session:tok:abcdef (for sessions)
```

```go
// cache/keys.go
package cache

import "fmt"

const (
    servicePrefix = "user-api:v1"
)

func UserKey(id string) string {
    return fmt.Sprintf("%s:user:%s", servicePrefix, id)
}

func UserListKey(orgID string, page int) string {
    return fmt.Sprintf("%s:user-list:org:%s:page:%d", servicePrefix, orgID, page)
}

func SessionKey(token string) string {
    return fmt.Sprintf("%s:session:tok:%s", servicePrefix, token)
}
```

**Version prefix strategy:**
- Bump `v1` → `v2` in the prefix constant to invalidate all cache entries when the schema changes
- Old keys expire naturally (TTL) — no need for mass DEL on deploy
- Only use `SCAN + DEL` for emergency invalidation, never `KEYS *` in production

---

## TTL Constants

Define TTLs once, reference everywhere. Never scatter magic numbers.

```go
// cache/ttl.go
package cache

import "time"

const (
    TTLUser        = 5 * time.Minute
    TTLUserList    = 2 * time.Minute  // shorter: list results change more often
    TTLSession     = 30 * time.Minute
    TTLConfig      = 1 * time.Hour
    TTLPermissions = 15 * time.Minute
    TTLHealthCheck = 10 * time.Second
)
```

---

## Get-or-Set Pattern (Read-Through Cache)

```go
// cache/user_cache.go
package cache

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"

    "github.com/redis/go-redis/v9"
)

type UserCache struct {
    rdb  *redis.Client
    repo UserRepository // downstream data source
}

func (c *UserCache) GetUser(ctx context.Context, id string) (*User, error) {
    key := UserKey(id)

    // 1. Try cache
    data, err := c.rdb.Get(ctx, key).Bytes()
    if err == nil {
        var user User
        if jsonErr := json.Unmarshal(data, &user); jsonErr == nil {
            return &user, nil
        }
        // Corrupted cache entry — treat as miss, re-fetch
    }

    if !errors.Is(err, redis.Nil) {
        // Real Redis error (not just a cache miss) — log but don't fail
        slog.Warn("cache get error, falling back to DB", "key", key, "error", err)
    }

    // 2. Cache miss — fetch from DB
    user, err := c.repo.GetUser(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("repo.GetUser: %w", err)
    }

    // 3. Populate cache (best-effort; don't fail if Redis is down)
    if b, marshalErr := json.Marshal(user); marshalErr == nil {
        if setErr := c.rdb.Set(ctx, key, b, TTLUser).Err(); setErr != nil {
            slog.Warn("cache set failed", "key", key, "error", setErr)
        }
    }

    return user, nil
}
```

**Key rule:** Never fail a request because the cache is unavailable. Cache errors should degrade gracefully to a direct DB hit.

---

## Cache Stampede Prevention with `singleflight`

When many requests arrive simultaneously for an expired key, `singleflight` collapses them into a single DB call.

```go
import "golang.org/x/sync/singleflight"

type UserCache struct {
    rdb  *redis.Client
    repo UserRepository
    sf   singleflight.Group
}

func (c *UserCache) GetUser(ctx context.Context, id string) (*User, error) {
    key := UserKey(id)

    // Try cache first (no singleflight needed here)
    if cached, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
        var user User
        json.Unmarshal(cached, &user)
        return &user, nil
    }

    // Collapse concurrent misses for the same key
    v, err, _ := c.sf.Do(key, func() (interface{}, error) {
        user, err := c.repo.GetUser(ctx, id)
        if err != nil {
            return nil, err
        }
        if b, e := json.Marshal(user); e == nil {
            c.rdb.Set(ctx, key, b, TTLUser)
        }
        return user, nil
    })

    if err != nil {
        return nil, err
    }
    return v.(*User), nil
}
```

**When to use:** High-traffic services where a single popular entity (e.g., a trending product) expires and hundreds of requests hit the DB simultaneously. For low-traffic services, simple get-or-set is sufficient.

---

## Two-Level Cache (L1 In-Process + L2 Redis)

For extremely hot reads, reduce Redis round-trips with a short-lived in-process cache.

```go
// cache/two_level.go
package cache

import (
    "sync"
    "time"
)

type l1Entry struct {
    value     []byte
    expiresAt time.Time
}

type L1Cache struct {
    mu      sync.RWMutex
    entries map[string]l1Entry
}

func NewL1Cache() *L1Cache {
    c := &L1Cache{entries: make(map[string]l1Entry)}
    // Background sweep: remove expired entries every 30 seconds
    go func() {
        t := time.NewTicker(30 * time.Second)
        for range t.C {
            c.sweep()
        }
    }()
    return c
}

func (c *L1Cache) Get(key string) ([]byte, bool) {
    c.mu.RLock()
    e, ok := c.entries[key]
    c.mu.RUnlock()
    if !ok || time.Now().After(e.expiresAt) {
        return nil, false
    }
    return e.value, true
}

func (c *L1Cache) Set(key string, value []byte, ttl time.Duration) {
    c.mu.Lock()
    c.entries[key] = l1Entry{value: value, expiresAt: time.Now().Add(ttl)}
    c.mu.Unlock()
}

func (c *L1Cache) sweep() {
    now := time.Now()
    c.mu.Lock()
    for k, e := range c.entries {
        if now.After(e.expiresAt) {
            delete(c.entries, k)
        }
    }
    c.mu.Unlock()
}

// TwoLevelCache wraps L1 (in-process) and L2 (Redis)
type TwoLevelCache struct {
    l1   *L1Cache
    l2   *redis.Client
    repo UserRepository
}

const l1TTL = 30 * time.Second  // much shorter than L2
const l2TTL = TTLUser            // 5 minutes

func (c *TwoLevelCache) GetUser(ctx context.Context, id string) (*User, error) {
    key := UserKey(id)

    // L1 hit
    if b, ok := c.l1.Get(key); ok {
        var user User
        json.Unmarshal(b, &user)
        return &user, nil
    }

    // L2 hit
    if b, err := c.l2.Get(ctx, key).Bytes(); err == nil {
        var user User
        if json.Unmarshal(b, &user) == nil {
            c.l1.Set(key, b, l1TTL) // promote to L1
            return &user, nil
        }
    }

    // DB fallback
    user, err := c.repo.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    if b, e := json.Marshal(user); e == nil {
        c.l2.Set(ctx, key, b, l2TTL)
        c.l1.Set(key, b, l1TTL)
    }
    return user, nil
}
```

**Tradeoffs:**
- L1 TTL must be short enough that stale reads are tolerable (30s is typical for user data)
- Invalidation only reaches L2 — L1 entries will serve stale data until L1 TTL expires
- Only add L1 if Redis latency (typically 1–5ms) is measurably impacting throughput

---

## Invalidation on Write

Always invalidate cache immediately after a successful DB write.

```go
func (r *UserRepository) UpdateUser(ctx context.Context, u *User) error {
    if err := r.db.Update(ctx, u); err != nil {
        return fmt.Errorf("db.Update: %w", err)
    }

    // Invalidate immediately — next read will re-populate from DB
    key := UserKey(u.ID)
    if err := r.cache.Del(ctx, key).Err(); err != nil {
        // Log but don't fail — stale cache is better than a failed update
        slog.Warn("cache invalidation failed", "key", key, "error", err)
    }

    // Also invalidate any list caches that include this user
    // Use a consistent key prefix or tag for related keys
    listKey := fmt.Sprintf("user-api:v1:user-list:org:%s:*", u.OrgID)
    _ = scanAndDelete(ctx, r.cache, listKey)

    return nil
}
```

**Patterns:**
- Write-through: update DB and cache simultaneously (risks cache/DB drift if DB update fails)
- Write-behind: update cache first, async DB update (risky, only for non-critical data)
- **Cache-aside** (recommended): update DB, delete cache key, let next read repopulate. Simplest and safest.

---

## Bulk Invalidation via SCAN

**Never use `KEYS *` in production** — it blocks the Redis event loop.

```go
// scanAndDelete deletes all keys matching a pattern using non-blocking SCAN.
func scanAndDelete(ctx context.Context, rdb *redis.Client, pattern string) error {
    var cursor uint64
    for {
        keys, nextCursor, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            return fmt.Errorf("scan: %w", err)
        }
        if len(keys) > 0 {
            if err := rdb.Del(ctx, keys...).Err(); err != nil {
                return fmt.Errorf("del: %w", err)
            }
        }
        cursor = nextCursor
        if cursor == 0 {
            break // scan complete
        }
    }
    return nil
}
```

**Cluster note:** In Redis Cluster, `SCAN` only iterates keys on the node you're connected to. Use `ForEachMaster` from `go-redis` to scan all shards:
```go
rdb.(*redis.ClusterClient).ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
    return scanAndDelete(ctx, node, pattern)
})
```

---

## Testing with `miniredis`

Use `miniredis` for unit tests — no Docker required, runs in-process.

```go
import (
    "testing"
    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

func TestGetUser_CacheHit(t *testing.T) {
    mr := miniredis.RunT(t) // auto-cleanup on test end
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

    cache := &UserCache{rdb: rdb, repo: &mockRepo{}}

    // Pre-populate cache
    user := &User{ID: "123", Name: "Alice"}
    b, _ := json.Marshal(user)
    mr.Set(UserKey("123"), string(b))
    mr.SetTTL(UserKey("123"), TTLUser)

    result, err := cache.GetUser(context.Background(), "123")
    require.NoError(t, err)
    assert.Equal(t, "Alice", result.Name)
}

func TestGetUser_TTLExpiry(t *testing.T) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

    b, _ := json.Marshal(&User{ID: "123", Name: "Alice"})
    mr.Set(UserKey("123"), string(b))
    mr.SetTTL(UserKey("123"), 1*time.Second)

    // Simulate TTL expiry without sleeping
    mr.FastForward(2 * time.Second)

    // Next Get should miss cache and call repo
    repo := &mockRepo{user: &User{ID: "123", Name: "Alice-updated"}}
    cache := &UserCache{rdb: rdb, repo: repo}

    result, _ := cache.GetUser(context.Background(), "123")
    assert.Equal(t, "Alice-updated", result.Name)
    assert.True(t, repo.called, "expected repo to be called on cache miss")
}
```

**Test coverage checklist:**
- [ ] Cache hit returns correct data without calling repo
- [ ] Cache miss calls repo and populates cache
- [ ] Redis error falls back to repo (test by closing `mr` before the call)
- [ ] TTL expiry triggers re-fetch (`mr.FastForward`)
- [ ] Invalidation removes the key (`mr.Exists`)
- [ ] Concurrent requests don't cause stampede (use `singleflight` test with goroutines)
