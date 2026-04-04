---
name: multi-tenancy
description: Multi-tenancy implementation patterns — row-per-tenant with PostgreSQL RLS, tenant context propagation in Go, index design, migration strategy, and isolation testing.
origin: vibemenu
---

# Multi-Tenancy Implementation

Patterns for building multi-tenant SaaS applications. Default recommendation: row-per-tenant with PostgreSQL Row Level Security.

## When to Activate

- Manifest describes a SaaS product serving multiple organizations/accounts
- Backend services share a database across tenants
- Frontend has organization-scoped routes or data namespacing
- Compliance requirements mandate data isolation

---

## Strategy Comparison

| Strategy | Isolation | Cost | Complexity | Recommended For |
|----------|-----------|------|------------|----------------|
| Row-per-tenant | Weak (app+DB) | Low | Low | Most SaaS apps |
| Schema-per-tenant | Medium | Medium | Medium | >100 tenants, regulatory |
| Database-per-tenant | Strong | High | High | Finance, healthcare |

**Default choice:** Row-per-tenant with PostgreSQL RLS for most applications. It supports millions of tenants, keeps ops simple, and RLS handles isolation at the DB engine level.

Use schema-per-tenant only if you need per-tenant migration windows or sub-tenant user spaces.
Use database-per-tenant only if a compliance requirement explicitly mandates physical data separation.

---

## Database Schema for Row-Per-Tenant

Add `tenant_id` to every tenant-scoped table. Use UUID for tenant IDs — never expose sequential integers.

```sql
-- Tenants table (the source of truth)
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT UNIQUE NOT NULL,  -- used in subdomains and URLs
    name        TEXT NOT NULL,
    plan        TEXT NOT NULL DEFAULT 'free',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every tenant-scoped table follows this pattern
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, email)  -- unique within tenant, not globally
);

CREATE TABLE orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id),
    total       NUMERIC(12,2) NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## PostgreSQL Row Level Security (RLS)

RLS enforces tenant isolation at the database engine level — a query that forgets `WHERE tenant_id = ?` still only returns the current tenant's rows.

```sql
-- 1. Enable RLS on every tenant-scoped table
ALTER TABLE users  ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

-- FORCE RLS applies to table owners too (critical — prevents bypass by app DB user)
ALTER TABLE users  FORCE ROW LEVEL SECURITY;
ALTER TABLE orders FORCE ROW LEVEL SECURITY;

-- 2. Create isolation policy
-- current_setting('app.tenant_id') is set per-connection by the application
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- 3. Grant the app DB user SELECT/INSERT/UPDATE/DELETE (not BYPASSRLS)
GRANT SELECT, INSERT, UPDATE, DELETE ON users, orders TO app_user;
```

**The `true` parameter in `current_setting('app.tenant_id', true)`** makes it return NULL instead of raising an error when the setting is unset. Your policy should handle NULL:
```sql
-- Explicit NULL guard — reject queries with no tenant context set
CREATE POLICY tenant_isolation ON users
    USING (
        current_setting('app.tenant_id', true) IS NOT NULL
        AND tenant_id = current_setting('app.tenant_id', true)::uuid
    );
```

---

## Tenant Context Propagation in Go

```go
// tenant/context.go
package tenant

import (
    "context"
    "errors"
)

type contextKey string

const tenantIDKey contextKey = "tenant_id"

// WithTenantID stores tenant ID in context. Call this in HTTP middleware.
func WithTenantID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, tenantIDKey, id)
}

// TenantIDFromContext retrieves tenant ID. Returns error if not set.
func TenantIDFromContext(ctx context.Context) (string, error) {
    id, ok := ctx.Value(tenantIDKey).(string)
    if !ok || id == "" {
        return "", errors.New("tenant ID not found in context")
    }
    return id, nil
}

// MustTenantID panics if no tenant is set. Use only in guaranteed-tenant contexts.
func MustTenantID(ctx context.Context) string {
    id, err := TenantIDFromContext(ctx)
    if err != nil {
        panic("tenant ID missing from context — middleware misconfigured")
    }
    return id
}
```

---

## HTTP Middleware — Extract Tenant from Request

Support both subdomain routing (e.g., `acme.myapp.com`) and JWT claim (`claims.tenant_id`).

```go
// middleware/tenant.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/myorg/myapp/tenant"
)

// TenantFromSubdomain extracts tenant slug from subdomain and resolves tenant ID.
// acme.myapp.com → slug "acme" → tenant ID lookup
func TenantFromSubdomain(tenantRepo TenantRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            host := r.Host
            parts := strings.SplitN(host, ".", 2)
            if len(parts) < 2 {
                http.Error(w, "invalid host", http.StatusBadRequest)
                return
            }
            slug := parts[0]
            t, err := tenantRepo.GetBySlug(r.Context(), slug)
            if err != nil {
                http.Error(w, "tenant not found", http.StatusNotFound)
                return
            }
            ctx := tenant.WithTenantID(r.Context(), t.ID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// TenantFromJWT extracts tenant_id from a validated JWT claim.
func TenantFromJWT(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := r.Context().Value(jwtClaimsKey{}).(JWTClaims)
        if !ok || claims.TenantID == "" {
            http.Error(w, "missing tenant claim", http.StatusUnauthorized)
            return
        }
        ctx := tenant.WithTenantID(r.Context(), claims.TenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## pgx Connection Middleware — Set Tenant Context

Set `app.tenant_id` on every DB connection before executing queries. Use `pgx` transaction-scoped settings with `SET LOCAL`.

```go
// repo/base.go
package repo

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/myorg/myapp/tenant"
)

// withTenantContext sets the PostgreSQL session variable for RLS.
// Must be called inside a transaction; use SET LOCAL so it applies only to the transaction.
func withTenantContext(ctx context.Context, tx pgx.Tx) error {
    tenantID, err := tenant.TenantIDFromContext(ctx)
    if err != nil {
        return fmt.Errorf("withTenantContext: %w", err)
    }
    _, err = tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
    return err
}

// BeginTenantTx starts a transaction and sets tenant context for RLS.
func BeginTenantTx(ctx context.Context, db *pgx.Conn) (pgx.Tx, error) {
    tx, err := db.Begin(ctx)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }
    if err := withTenantContext(ctx, tx); err != nil {
        tx.Rollback(ctx)
        return nil, err
    }
    return tx, nil
}
```

**Usage in repository:**
```go
func (r *OrderRepo) ListOrders(ctx context.Context) ([]Order, error) {
    tx, err := BeginTenantTx(ctx, r.db)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    // RLS is now active — query will only return rows for the current tenant
    rows, err := tx.Query(ctx, "SELECT id, status, total FROM orders ORDER BY created_at DESC")
    if err != nil {
        return nil, fmt.Errorf("query orders: %w", err)
    }
    defer rows.Close()

    orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[Order])
    if err != nil {
        return nil, err
    }

    return orders, tx.Commit(ctx)
}
```

---

## Index Design

Always prefix indexes with `tenant_id`. Queries will always filter by it — an index without it is useless for tenant-scoped queries.

```sql
-- Correct: tenant_id first in all indexes
CREATE INDEX ON users (tenant_id, email);
CREATE INDEX ON orders (tenant_id, created_at DESC);
CREATE INDEX ON orders (tenant_id, status) WHERE status != 'completed';
CREATE INDEX ON orders (tenant_id, user_id, created_at DESC);

-- WRONG: missing tenant_id prefix — full table scans for tenant queries
-- CREATE INDEX ON orders (created_at DESC);
-- CREATE INDEX ON orders (status);

-- Partial indexes for active records (reduces index size significantly for large tenants)
CREATE INDEX ON users (tenant_id, email) WHERE deleted_at IS NULL;
```

**Query planner note:** With RLS active, the query planner sees the `tenant_id = ?` filter and uses the `(tenant_id, ...)` indexes. Run `EXPLAIN ANALYZE` on your most frequent tenant-scoped queries to verify index hits.

---

## Non-Destructive RLS Rollout for Existing Tables

Enabling RLS on a production table with existing data requires careful steps to avoid downtime.

```sql
-- Step 1: Add tenant_id column (nullable initially to allow backfill)
ALTER TABLE orders ADD COLUMN tenant_id UUID REFERENCES tenants(id);

-- Step 2: Backfill from existing data (do this in batches for large tables)
UPDATE orders SET tenant_id = (
    SELECT tenant_id FROM users WHERE users.id = orders.user_id
) WHERE tenant_id IS NULL;

-- Step 3: Verify no NULLs remain before adding constraint
SELECT COUNT(*) FROM orders WHERE tenant_id IS NULL;

-- Step 4: Add NOT NULL constraint
ALTER TABLE orders ALTER COLUMN tenant_id SET NOT NULL;

-- Step 5: Add index
CREATE INDEX CONCURRENTLY ON orders (tenant_id, created_at DESC);

-- Step 6: Create policy (does not activate yet)
CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- Step 7: Dry-run with a read-only check — verify no unintended data leakage
-- (Test in staging: set app.tenant_id, run queries, verify row counts)

-- Step 8: Enable RLS
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders FORCE ROW LEVEL SECURITY;
```

**Rollback plan:** RLS can be disabled without data loss: `ALTER TABLE orders DISABLE ROW LEVEL SECURITY;`

---

## Bypass RLS for Migrations and Admin

Grant `BYPASSRLS` only to the migration user. Never to the application user.

```sql
-- Migration role bypasses RLS for schema changes and backfills
ALTER ROLE migration_user BYPASSRLS;

-- Application role must never bypass RLS
-- REVOKE BYPASSRLS FROM app_user;  -- verify this is not set

-- For superuser connections, RLS is always bypassed — never use superuser in app code
```

In Go, use separate connection strings:
```go
// Application connection — limited privileges, RLS enforced
appDB := connectDB(os.Getenv("DATABASE_URL"))

// Migration connection — BYPASSRLS, only used by migration tooling
migrationDB := connectDB(os.Getenv("MIGRATION_DATABASE_URL"))
```

---

## Tenant Isolation Testing

Write tests that verify tenant A cannot read tenant B's data through the application layer.

```go
// repo/order_repo_test.go
func TestTenantIsolation(t *testing.T) {
    db := testDB(t) // uses real PostgreSQL with RLS enabled

    tenantA := createTenant(t, db, "tenant-a")
    tenantB := createTenant(t, db, "tenant-b")

    // Create orders for both tenants (using migration connection to bypass RLS)
    orderA := createOrder(t, db, tenantA.ID, "order-A")
    orderB := createOrder(t, db, tenantB.ID, "order-B")

    repo := NewOrderRepo(db)

    // Tenant A context — should only see tenant A's orders
    ctxA := tenant.WithTenantID(context.Background(), tenantA.ID)
    ordersA, err := repo.ListOrders(ctxA)
    require.NoError(t, err)
    ids := orderIDs(ordersA)
    assert.Contains(t, ids, orderA.ID, "tenant A should see its own order")
    assert.NotContains(t, ids, orderB.ID, "tenant A must NOT see tenant B's order")

    // Tenant B context — should only see tenant B's orders
    ctxB := tenant.WithTenantID(context.Background(), tenantB.ID)
    ordersB, err := repo.ListOrders(ctxB)
    require.NoError(t, err)
    ids = orderIDs(ordersB)
    assert.Contains(t, ids, orderB.ID)
    assert.NotContains(t, ids, orderA.ID)
}

func TestMissingTenantContext(t *testing.T) {
    repo := NewOrderRepo(testDB(t))

    // A request with no tenant context must fail, not return all rows
    _, err := repo.ListOrders(context.Background())
    require.Error(t, err, "missing tenant context must return an error, not all rows")
}
```

---

## Anti-Patterns to Avoid

- **Forgetting `FORCE ROW LEVEL SECURITY`**: Without it, table owners and superusers bypass RLS silently.
- **`SET app.tenant_id` without `LOCAL`**: `SET` is session-scoped; `SET LOCAL` is transaction-scoped. With connection pools, session-level settings leak across requests.
- **Exposing sequential tenant IDs**: Use UUID. An attacker can enumerate tenants if IDs are `1, 2, 3...`.
- **Global `tenant_id` variable**: Never store tenant ID in a package-level variable — it's not goroutine-safe and causes cross-request data leakage.
- **Filtering in application code instead of DB**: `SELECT * FROM orders` then filtering in Go is dangerous — a bug exposes all tenants' data. Let RLS enforce it at the DB level.
- **Missing tenant_id indexes**: Without them, PostgreSQL does full table scans on every query, killing performance at scale.
