---
name: api-versioning-impl
description: API versioning implementation patterns — URL path, header-based, content negotiation, deprecation headers (RFC 8594), sunset enforcement, and testing strategies.
origin: vibemenu
---

# API Versioning Implementation

API versioning enables backward-compatible evolution of APIs. This skill covers the three main strategies (URL path, header, content negotiation), how to implement deprecation warnings, and how to enforce sunset retirement.

## When to Activate

- Adding a `/v2` endpoint to an existing API
- Implementing breaking changes without disrupting existing clients
- Setting up deprecation warnings and sunset enforcement
- Designing a new API that will need to evolve over time

## Strategy Comparison

| Strategy | Caching | Client Complexity | Discoverability | Recommendation |
|----------|---------|-------------------|-----------------|----------------|
| URL path (`/v1/users`) | Excellent | Low | High | Default choice |
| Header (`Accept-Version: v2`) | Poor (varies by header) | Medium | Medium | When URL pollution matters |
| Content negotiation (`Accept: application/vnd.myapp.v2+json`) | Poor | High | Low | REST purists, hypermedia APIs |

## URL Path Versioning (Default)

### Go — Fiber

```go
import "github.com/gofiber/fiber/v2"

func SetupRoutes(app *fiber.App) {
    // Shared business logic — versioned handlers call into these
    userSvc := services.NewUserService()

    v1 := app.Group("/v1")
    v1.Get("/users", handlers.ListUsersV1(userSvc))
    v1.Post("/users", handlers.CreateUserV1(userSvc))
    v1.Get("/users/:id", handlers.GetUserV1(userSvc))

    v2 := app.Group("/v2")
    v2.Get("/users", handlers.ListUsersV2(userSvc)) // new shape: cursor pagination
    v2.Post("/users", handlers.CreateUserV2(userSvc))
    v2.Get("/users/:id", handlers.GetUserV2(userSvc))
}
```

### Go — Gin

```go
import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
    v1 := r.Group("/v1")
    {
        v1.GET("/users", handlers.ListUsersV1)
        v1.POST("/users", handlers.CreateUserV1)
        v1.GET("/users/:id", handlers.GetUserV1)
    }

    v2 := r.Group("/v2")
    {
        v2.GET("/users", handlers.ListUsersV2)
        v2.POST("/users", handlers.CreateUserV2)
        v2.GET("/users/:id", handlers.GetUserV2)
    }
}
```

### Go — Echo

```go
import "github.com/labstack/echo/v4"

func SetupRoutes(e *echo.Echo) {
    v1 := e.Group("/v1")
    v1.GET("/users", handlers.ListUsersV1)
    v1.POST("/users", handlers.CreateUserV1)

    v2 := e.Group("/v2")
    v2.GET("/users", handlers.ListUsersV2)
    v2.POST("/users", handlers.CreateUserV2)
}
```

### Node.js — Express

```typescript
import express, { Router } from 'express';
import { v1Router } from './routes/v1';
import { v2Router } from './routes/v2';

const app = express();

app.use('/v1', v1Router);
app.use('/v2', v2Router);

// routes/v1/index.ts
export const v1Router = Router();
v1Router.get('/users', listUsersV1);
v1Router.post('/users', createUserV1);

// routes/v2/index.ts — same structure, different handlers
export const v2Router = Router();
v2Router.get('/users', listUsersV2);
```

### Python — FastAPI

```python
from fastapi import FastAPI
from app.routers.v1 import users as users_v1
from app.routers.v2 import users as users_v2

app = FastAPI()

app.include_router(users_v1.router, prefix="/v1")
app.include_router(users_v2.router, prefix="/v2")

# routers/v1/users.py
from fastapi import APIRouter

router = APIRouter(prefix="/users", tags=["users-v1"])

@router.get("/")
async def list_users_v1():
    ...

# routers/v2/users.py — different response schema, cursor pagination
from fastapi import APIRouter

router = APIRouter(prefix="/users", tags=["users-v2"])

@router.get("/")
async def list_users_v2(cursor: str | None = None, limit: int = 20):
    ...
```

### Monorepo Handler Organization

```
handlers/
├── v1/
│   ├── users.go       # V1 request/response types + handlers
│   └── orders.go
├── v2/
│   ├── users.go       # V2 types + handlers — call same service layer
│   └── orders.go
└── shared/
    └── middleware.go  # Auth, logging — shared across versions
```

```go
// handlers/v2/users.go — only the handler and DTO change; service is shared
package v2

import (
    "myapp/services"
    "myapp/handlers/v2/dto"
)

func ListUsers(userSvc *services.UserService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        cursor := c.Query("cursor")
        limit := c.QueryInt("limit", 20)

        // Same service call — different response serialization
        users, nextCursor, err := userSvc.ListUsers(c.Context(), cursor, limit)
        if err != nil {
            return err
        }
        return c.JSON(dto.PaginatedUsersResponse{
            Data:       dto.MapUsers(users),
            NextCursor: nextCursor,
        })
    }
}
```

## Header-Based Versioning

### Go Middleware

```go
// middleware/version.go
package middleware

import (
    "context"
    "net/http"
)

type contextKey string

const apiVersionKey contextKey = "api_version"

func APIVersion(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        version := r.Header.Get("Accept-Version")
        if version == "" {
            version = r.Header.Get("API-Version")
        }
        if version == "" {
            version = "v2" // default to latest stable
        }

        ctx := context.WithValue(r.Context(), apiVersionKey, version)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetAPIVersion(ctx context.Context) string {
    v, _ := ctx.Value(apiVersionKey).(string)
    return v
}
```

```go
// Handler uses version from context to dispatch
func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
    version := middleware.GetAPIVersion(r.Context())

    switch version {
    case "v1":
        renderV1Users(w, r)
    case "v2":
        renderV2Users(w, r)
    default:
        http.Error(w, "unsupported API version", http.StatusBadRequest)
    }
}
```

### Express Middleware

```typescript
import { Request, Response, NextFunction } from 'express';

declare module 'express' {
  interface Request {
    apiVersion: string;
  }
}

export function apiVersionMiddleware(req: Request, _res: Response, next: NextFunction): void {
  req.apiVersion = req.headers['accept-version'] as string
    || req.headers['api-version'] as string
    || 'v2'; // fallback to latest
  next();
}

// Handler
app.get('/users', (req, res) => {
  if (req.apiVersion === 'v1') {
    return res.json(listUsersV1());
  }
  return res.json(listUsersV2());
});
```

## Content Negotiation Versioning

```go
// Accept: application/vnd.myapp.v2+json
func contentTypeMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        accept := r.Header.Get("Accept")
        version := parseVendorVersion(accept) // "v2" from "application/vnd.myapp.v2+json"
        if version == "" {
            version = "v2"
        }
        ctx := context.WithValue(r.Context(), apiVersionKey, version)
        // Echo back the version in Content-Type
        w.Header().Set("Content-Type", fmt.Sprintf("application/vnd.myapp.%s+json", version))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func parseVendorVersion(accept string) string {
    // "application/vnd.myapp.v2+json" -> "v2"
    re := regexp.MustCompile(`application/vnd\.myapp\.(v\d+)\+json`)
    m := re.FindStringSubmatch(accept)
    if len(m) > 1 {
        return m[1]
    }
    return ""
}
```

## Deprecation Headers (RFC 8594)

Add deprecation and sunset headers in middleware — not in individual handlers. This ensures every response for a deprecated version includes the warning:

```go
// middleware/deprecation.go
package middleware

import (
    "net/http"
    "time"
)

type VersionPolicy struct {
    DeprecatedAt time.Time
    SunsetAt     time.Time
    SuccessorURL string
}

var versionPolicies = map[string]VersionPolicy{
    "v1": {
        DeprecatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        SunsetAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
        SuccessorURL: "https://api.example.com/v2",
    },
}

func DeprecationHeaders(version string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if policy, ok := versionPolicies[version]; ok {
                // RFC 8594 Deprecation header
                w.Header().Set("Deprecation",
                    policy.DeprecatedAt.UTC().Format(http.TimeFormat))
                // RFC 8594 Sunset header
                w.Header().Set("Sunset",
                    policy.SunsetAt.UTC().Format(http.TimeFormat))
                // Link to successor
                w.Header().Set("Link",
                    fmt.Sprintf(`<%s>; rel="successor-version"`, policy.SuccessorURL))
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

```go
// Apply to versioned route groups
v1 := app.Group("/v1")
v1.Use(middleware.DeprecationHeaders("v1"))
v1.Get("/users", handlers.ListUsersV1)
```

```typescript
// Express equivalent
function deprecationMiddleware(policy: VersionPolicy) {
  return (_req: Request, res: Response, next: NextFunction) => {
    res.set('Deprecation', policy.deprecatedAt.toUTCString());
    res.set('Sunset', policy.sunsetAt.toUTCString());
    res.set('Link', `<${policy.successorUrl}>; rel="successor-version"`);
    next();
  };
}

app.use('/v1', deprecationMiddleware({
  deprecatedAt: new Date('2025-01-01'),
  sunsetAt: new Date('2026-01-01'),
  successorUrl: 'https://api.example.com/v2',
}));
```

## Sunset Enforcement (Return 410 Gone)

```go
func SunsetEnforcement(version string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            policy, ok := versionPolicies[version]
            if ok && time.Now().UTC().After(policy.SunsetAt) {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusGone)
                json.NewEncoder(w).Encode(map[string]string{
                    "error": fmt.Sprintf(
                        "API version %s was retired on %s. Please upgrade to %s.",
                        version,
                        policy.SunsetAt.Format("2006-01-02"),
                        policy.SuccessorURL,
                    ),
                })
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

```python
# FastAPI middleware
from fastapi import Request
from fastapi.responses import JSONResponse
from datetime import datetime, timezone

VERSION_POLICIES = {
    "v1": {
        "sunset_at": datetime(2026, 1, 1, tzinfo=timezone.utc),
        "successor_url": "https://api.example.com/v2",
    }
}

@app.middleware("http")
async def sunset_enforcement(request: Request, call_next):
    # Extract version from URL path
    path_parts = request.url.path.split("/")
    version = next((p for p in path_parts if p.startswith("v") and p[1:].isdigit()), None)

    if version and version in VERSION_POLICIES:
        policy = VERSION_POLICIES[version]
        if datetime.now(timezone.utc) > policy["sunset_at"]:
            return JSONResponse(
                status_code=410,
                content={"error": f"API {version} retired. Upgrade to {policy['successor_url']}."}
            )
        # Add deprecation headers for active-but-deprecated versions
        response = await call_next(request)
        response.headers["Sunset"] = policy["sunset_at"].strftime("%a, %d %b %Y %H:%M:%S GMT")
        return response

    return await call_next(request)
```

## Version Routing Utility

```go
// Semver-comparable version parsing for programmatic comparisons
type APIVersion struct {
    Major int
}

func ParseVersion(s string) (APIVersion, error) {
    // Accepts "v1", "v2", "1", "2"
    s = strings.TrimPrefix(s, "v")
    major, err := strconv.Atoi(s)
    if err != nil {
        return APIVersion{}, fmt.Errorf("invalid api version %q: %w", s, err)
    }
    return APIVersion{Major: major}, nil
}

func (v APIVersion) AtLeast(other APIVersion) bool {
    return v.Major >= other.Major
}

func (v APIVersion) String() string {
    return fmt.Sprintf("v%d", v.Major)
}
```

## Testing Versioned APIs

```go
func TestAPIVersions(t *testing.T) {
    app := setupTestApp()

    t.Run("v1 and v2 coexist", func(t *testing.T) {
        // v1 returns flat pagination
        resp1 := makeRequest(app, "GET", "/v1/users?page=1", nil)
        assert.Equal(t, 200, resp1.Code)
        var v1Body map[string]any
        json.Unmarshal(resp1.Body.Bytes(), &v1Body)
        assert.NotNil(t, v1Body["items"])

        // v2 returns cursor pagination
        resp2 := makeRequest(app, "GET", "/v2/users", nil)
        assert.Equal(t, 200, resp2.Code)
        var v2Body map[string]any
        json.Unmarshal(resp2.Body.Bytes(), &v2Body)
        assert.Contains(t, v2Body, "next_cursor")
    })

    t.Run("v1 deprecation headers present", func(t *testing.T) {
        resp := makeRequest(app, "GET", "/v1/users", nil)
        assert.NotEmpty(t, resp.Header().Get("Deprecation"))
        assert.NotEmpty(t, resp.Header().Get("Sunset"))
        assert.Contains(t, resp.Header().Get("Link"), "successor-version")
    })

    t.Run("v1 returns 410 after sunset", func(t *testing.T) {
        // Temporarily override sunset date to the past
        versionPolicies["v1"] = VersionPolicy{
            SunsetAt: time.Now().Add(-time.Hour),
        }
        defer func() { versionPolicies["v1"] = originalV1Policy }()

        resp := makeRequest(app, "GET", "/v1/users", nil)
        assert.Equal(t, 410, resp.Code)
    })
}
```

## Anti-Patterns

```go
// ❌ BAD: Versioning via query parameter (hard to cache, breaks REST conventions)
GET /users?version=2

// ✅ GOOD: Versioned URL path
GET /v2/users

// ❌ BAD: Duplicating business logic in V2 handler instead of sharing the service layer
func ListUsersV2(c *fiber.Ctx) error {
    // copy-paste of V1 logic with minor changes
    users, _ := db.Query("SELECT ...")
    ...
}

// ✅ GOOD: V1 and V2 call the same service; only the DTO and serialization differ
func ListUsersV2(userSvc *services.UserService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        users, cursor, _ := userSvc.ListWithCursor(c.Context(), c.Query("cursor"), 20)
        return c.JSON(v2dto.MapUsers(users, cursor))
    }
}

// ❌ BAD: Breaking a deployed v1 endpoint (renaming fields, changing types)
// Existing clients will break immediately

// ✅ GOOD: Add new fields to v1 (backward compatible), breaking changes go to v2 only

// ❌ BAD: No deprecation notice — clients are surprised when v1 disappears
// ✅ GOOD: At least 6 months of Deprecation + Sunset headers before removing
```
