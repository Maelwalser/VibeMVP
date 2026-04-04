---
name: pagination-impl
description: Pagination implementation patterns — cursor-based (keyset), offset/limit, response envelopes, and multi-language code snippets.
origin: vibemenu
---

# Pagination Implementation Patterns

Two dominant strategies: **cursor-based** (keyset) for large tables and live data, **offset/limit** for small tables and admin UIs. This skill covers correct SQL, cursor encoding, response shape, and multi-language implementations.

## When to Activate

- Implementing list endpoints that return more than ~50 items
- Designing paginated API responses
- Optimizing slow `SELECT ... LIMIT ... OFFSET ...` queries
- Building infinite-scroll or "load more" UIs

## Strategy Selection

| Criteria | Use Cursor (Keyset) | Use Offset/Limit |
|----------|---------------------|------------------|
| Table size | > 10k rows | < 100k rows |
| Live data (items added during pagination) | Yes | No |
| Stable page numbers needed | No | Yes |
| Random page access (jump to page 7) | No | Yes |
| Sorting | By indexed column | Any column |

## Cursor-Based Pagination (Recommended for Large Tables)

### The Core Idea

Instead of `OFFSET n`, the cursor encodes the position of the last item seen. The next query filters rows *after* that position using a `WHERE` clause with the indexed columns.

### Cursor Encoding/Decoding

Never expose raw IDs or timestamps as cursors. Always base64-wrap a JSON object:

**Go**

```go
package pagination

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
)

type Cursor struct {
    LastID        int64  `json:"last_id"`
    LastCreatedAt string `json:"last_created_at"` // RFC3339
}

func EncodeCursor(c Cursor) (string, error) {
    b, err := json.Marshal(c)
    if err != nil {
        return "", fmt.Errorf("encode cursor: %w", err)
    }
    return base64.URLEncoding.EncodeToString(b), nil
}

func DecodeCursor(s string) (Cursor, error) {
    if s == "" {
        return Cursor{}, nil
    }
    b, err := base64.URLEncoding.DecodeString(s)
    if err != nil {
        return Cursor{}, fmt.Errorf("decode cursor: invalid base64: %w", err)
    }
    var c Cursor
    if err := json.Unmarshal(b, &c); err != nil {
        return Cursor{}, fmt.Errorf("decode cursor: invalid json: %w", err)
    }
    return c, nil
}
```

**TypeScript/Node.js**

```typescript
interface Cursor {
  last_id: number;
  last_created_at: string; // ISO-8601
}

export function encodeCursor(cursor: Cursor): string {
  return Buffer.from(JSON.stringify(cursor)).toString('base64url');
}

export function decodeCursor(encoded: string): Cursor {
  try {
    return JSON.parse(Buffer.from(encoded, 'base64url').toString('utf8'));
  } catch {
    throw new Error('Invalid pagination cursor');
  }
}
```

**Python**

```python
import base64
import json
from dataclasses import dataclass
from typing import Optional

@dataclass
class Cursor:
    last_id: int
    last_created_at: str  # ISO-8601

def encode_cursor(cursor: Cursor) -> str:
    data = {"last_id": cursor.last_id, "last_created_at": cursor.last_created_at}
    return base64.urlsafe_b64encode(json.dumps(data).encode()).decode()

def decode_cursor(encoded: str) -> Optional[Cursor]:
    if not encoded:
        return None
    try:
        data = json.loads(base64.urlsafe_b64decode(encoded.encode()))
        return Cursor(last_id=data["last_id"], last_created_at=data["last_created_at"])
    except Exception as e:
        raise ValueError(f"Invalid pagination cursor: {e}")
```

### Keyset SQL Pattern (Correct Cursor SQL)

```sql
-- Composite index — required for performance
CREATE INDEX idx_items_created_id ON items (created_at DESC, id DESC);

-- First page (no cursor)
SELECT id, title, created_at
FROM items
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- Subsequent pages (with cursor)
-- The (created_at, id) tuple comparison is the key to keyset pagination
SELECT id, title, created_at
FROM items
WHERE (created_at, id) < ($1, $2)   -- $1=last_created_at, $2=last_id
ORDER BY created_at DESC, id DESC
LIMIT $3;
```

**Why `(created_at, id) < ($1, $2)` and not two separate WHERE clauses:**

```sql
-- ❌ WRONG: Misses rows with same created_at
WHERE created_at < $1 AND id < $2

-- ✅ CORRECT: Tuple comparison matches PostgreSQL's composite index ordering
WHERE (created_at, id) < ($1, $2)
```

### Go Repository Method

```go
type PaginatedResult[T any] struct {
    Data       []T    `json:"data"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}

func (r *ItemRepository) ListItems(
    ctx context.Context,
    cursor string,
    limit int,
) (PaginatedResult[Item], error) {
    if limit <= 0 || limit > 100 {
        limit = 20
    }

    // Fetch limit+1 to detect if more pages exist
    fetchLimit := limit + 1

    var rows []Item
    var err error

    if cursor == "" {
        // First page
        rows, err = r.db.QueryContext(ctx, `
            SELECT id, title, created_at FROM items
            ORDER BY created_at DESC, id DESC
            LIMIT $1
        `, fetchLimit)
    } else {
        c, decErr := DecodeCursor(cursor)
        if decErr != nil {
            return PaginatedResult[Item]{}, fmt.Errorf("invalid cursor: %w", decErr)
        }
        rows, err = r.db.QueryContext(ctx, `
            SELECT id, title, created_at FROM items
            WHERE (created_at, id) < ($1, $2)
            ORDER BY created_at DESC, id DESC
            LIMIT $3
        `, c.LastCreatedAt, c.LastID, fetchLimit)
    }
    if err != nil {
        return PaginatedResult[Item]{}, fmt.Errorf("list items: %w", err)
    }

    hasMore := len(rows) > limit
    if hasMore {
        rows = rows[:limit]
    }

    var nextCursor string
    if hasMore && len(rows) > 0 {
        last := rows[len(rows)-1]
        nextCursor, err = EncodeCursor(Cursor{
            LastID:        last.ID,
            LastCreatedAt: last.CreatedAt.Format(time.RFC3339Nano),
        })
        if err != nil {
            return PaginatedResult[Item]{}, err
        }
    }

    return PaginatedResult[Item]{
        Data:       rows,
        NextCursor: nextCursor,
        HasMore:    hasMore,
    }, nil
}
```

### TypeScript/Node.js (with pg or Prisma)

```typescript
interface PaginatedResult<T> {
  data: T[];
  next_cursor: string | null;
  has_more: boolean;
}

async function listItems(
  cursor: string | undefined,
  limit: number = 20
): Promise<PaginatedResult<Item>> {
  const fetchLimit = limit + 1;

  let rows: Item[];

  if (!cursor) {
    rows = await db.query<Item>(
      'SELECT id, title, created_at FROM items ORDER BY created_at DESC, id DESC LIMIT $1',
      [fetchLimit]
    );
  } else {
    const c = decodeCursor(cursor);
    rows = await db.query<Item>(
      `SELECT id, title, created_at FROM items
       WHERE (created_at, id) < ($1, $2)
       ORDER BY created_at DESC, id DESC
       LIMIT $3`,
      [c.last_created_at, c.last_id, fetchLimit]
    );
  }

  const hasMore = rows.length > limit;
  if (hasMore) rows.splice(limit);

  const last = rows[rows.length - 1];
  const nextCursor = hasMore && last
    ? encodeCursor({ last_id: last.id, last_created_at: last.created_at.toISOString() })
    : null;

  return { data: rows, next_cursor: nextCursor, has_more: hasMore };
}
```

### Python/FastAPI

```python
from fastapi import Query
from pydantic import BaseModel
from typing import Generic, TypeVar, Optional, List

T = TypeVar("T")

class PaginatedResponse(BaseModel, Generic[T]):
    data: List[T]
    next_cursor: Optional[str] = None
    has_more: bool

@router.get("/items", response_model=PaginatedResponse[ItemSchema])
async def list_items(
    cursor: Optional[str] = Query(None),
    limit: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
):
    fetch_limit = limit + 1

    if not cursor:
        stmt = (
            select(Item)
            .order_by(Item.created_at.desc(), Item.id.desc())
            .limit(fetch_limit)
        )
    else:
        c = decode_cursor(cursor)
        stmt = (
            select(Item)
            .where(
                tuple_(Item.created_at, Item.id) < (c.last_created_at, c.last_id)
            )
            .order_by(Item.created_at.desc(), Item.id.desc())
            .limit(fetch_limit)
        )

    rows = (await db.execute(stmt)).scalars().all()

    has_more = len(rows) > limit
    if has_more:
        rows = rows[:limit]

    next_cursor = None
    if has_more and rows:
        last = rows[-1]
        next_cursor = encode_cursor(Cursor(
            last_id=last.id,
            last_created_at=last.created_at.isoformat()
        ))

    return PaginatedResponse(data=rows, next_cursor=next_cursor, has_more=has_more)
```

## Offset/Limit Pagination

Acceptable for admin panels and datasets under ~100k rows where users need page numbers:

```sql
-- Simple — works fine for small tables
SELECT id, title, created_at
FROM items
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
-- $1 = page_size, $2 = (page - 1) * page_size
```

```go
// Go handler
func ListItemsWithOffset(page, pageSize int) ([]Item, error) {
    if pageSize > 100 {
        pageSize = 100
    }
    offset := (page - 1) * pageSize
    return r.db.Query(`
        SELECT id, title, created_at FROM items
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `, pageSize, offset)
}
```

## Response Envelope

All paginated endpoints must use this exact envelope shape:

```json
{
  "data": [
    { "id": 1, "title": "First Item", "created_at": "2025-01-01T10:00:00Z" }
  ],
  "next_cursor": "eyJsYXN0X2lkIjoxLCJsYXN0X2NyZWF0ZWRfYXQiOiIyMDI1LTAxLTAxVDEwOjAwOjAwWiJ9",
  "has_more": true
}
```

For offset pagination:

```json
{
  "data": [...],
  "page": 2,
  "page_size": 20,
  "has_more": true,
  "total": 450
}
```

## Optional Total Count

Never compute `COUNT(*)` by default — it's expensive on large tables. Only compute when explicitly requested:

```go
// Only count when ?include_total=true
if r.URL.Query().Get("include_total") == "true" {
    var total int
    r.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&total)
    resp.Total = &total
}
```

```python
# FastAPI
@router.get("/items")
async def list_items(
    include_total: bool = Query(False),
    ...
):
    ...
    total = None
    if include_total:
        total = await db.scalar(select(func.count()).select_from(Item))
    return PaginatedResponse(..., total=total)
```

## OpenAPI Schema

```yaml
components:
  schemas:
    PaginatedItemsResponse:
      type: object
      required: [data, has_more]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Item'
        next_cursor:
          type: string
          nullable: true
          description: Opaque cursor for the next page. Null when has_more is false.
          example: "eyJsYXN0X2lkIjoxfQ=="
        has_more:
          type: boolean
          description: True if there are more results beyond this page.
        total:
          type: integer
          nullable: true
          description: Total count. Only present when include_total=true is requested.
```

## Anti-Patterns

```sql
-- ❌ BAD: OFFSET on large tables — PostgreSQL must scan and discard all preceding rows
SELECT * FROM items ORDER BY created_at DESC LIMIT 20 OFFSET 100000;
-- O(offset) scan; page 5000 is 5000x slower than page 1

-- ✅ GOOD: Keyset — always O(log N) via index
SELECT * FROM items WHERE (created_at, id) < ($1, $2) ORDER BY created_at DESC, id DESC LIMIT 20;
```

```typescript
// ❌ BAD: Returning raw database ID as cursor (exposes internal ID sequence)
return { next_cursor: String(lastItem.id) };

// ✅ GOOD: Opaque base64-wrapped cursor
return { next_cursor: encodeCursor({ last_id: lastItem.id, last_created_at: lastItem.createdAt }) };
```

```go
// ❌ BAD: Always computing COUNT(*) (expensive on large tables)
total, _ := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&total)

// ✅ GOOD: Opt-in only
if req.URL.Query().Get("include_total") == "true" {
    db.QueryRow("SELECT COUNT(*) FROM items").Scan(&total)
}

// ❌ BAD: No index on the cursor columns
// SELECT ... WHERE (created_at, id) < ... ORDER BY created_at DESC, id DESC
// Full table scan every time

// ✅ GOOD: Composite index matching the ORDER BY
// CREATE INDEX idx_items_created_id ON items (created_at DESC, id DESC);

// ❌ BAD: Page numbers for cursor-paginated APIs (breaks if items added between requests)
GET /items?page=5
// page 5 returns different items after new items are inserted at the front

// ✅ GOOD: Cursor pins position relative to the data
GET /items?cursor=eyJsYXN0X2lkIjo0MH0=
```
