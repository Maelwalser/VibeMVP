# PostgreSQL Skill Guide

## Connection with pgx

```go
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
    pool, err := pgxpool.New(ctx, dsn)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("ping database: %w", err)
    }
    return pool, nil
}
```

## DSN Format

```
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=disable
```

## Migration Files (golang-migrate format)

Use sequential numbered migrations:
```
data/migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_orders.up.sql
└── 000002_create_orders.down.sql
```

Example migration:
```sql
-- 000001_create_users.up.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

## Query Pattern

```go
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    row := r.pool.QueryRow(ctx,
        `SELECT id, email, created_at FROM users WHERE id = $1`, id)

    var u User
    if err := row.Scan(&u.ID, &u.Email, &u.CreatedAt); err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("query user %s: %w", id, err)
    }
    return &u, nil
}
```

## UUID Primary Keys

Always use `gen_random_uuid()` (requires PostgreSQL 13+) or `uuid_generate_v4()` for primary keys. Include the `pgcrypto` extension if needed:
```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

## Indexes

- Add indexes on all foreign keys.
- Add indexes on frequently queried columns (email, status, created_at).
- Use partial indexes for soft-deleted records: `WHERE deleted_at IS NULL`.
