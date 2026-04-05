# Go + Gin Skill Guide

## Project Layout

```
service-name/
├── go.mod
├── go.sum
├── main.go
├── internal/
│   ├── handler/     # HTTP handlers
│   ├── service/     # Business logic
│   ├── repository/  # Data access
│   └── middleware/  # Custom middleware
└── Dockerfile
```

## go.mod Boilerplate

Use the **exact module path and versions from the "Dependency & API Reference" section** of your
task prompt — never hardcode version strings. The example below shows the package names only:

```go
module <module_path from payload>

go <version from Dependency & API Reference>

require (
    github.com/gin-gonic/gin <version from Dependency & API Reference>
    github.com/jackc/pgx/v5 <version from Dependency & API Reference>
)
```

## Server Setup

```go
package main

import (
    "log"
    "os"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(errorMiddleware())

    registerRoutes(r)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(r.Run(":" + port))
}

func registerRoutes(r *gin.Engine) {
    r.GET("/health", healthCheck)

    v1 := r.Group("/api/v1")
    {
        users := v1.Group("/users")
        users.GET("", listUsers)
        users.POST("", createUser)
        users.GET("/:id", getUser)
        users.PUT("/:id", updateUser)
        users.DELETE("/:id", deleteUser)
    }
}
```

## Handler Pattern

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "your-org/service-name/internal/service"
)

type UserHandler struct {
    svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

func (h *UserHandler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    user, err := h.svc.Create(c.Request.Context(), req)
    if err != nil {
        c.Error(err) // passes to error middleware
        return
    }
    c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) Get(c *gin.Context) {
    id := c.Param("id")
    user, err := h.svc.GetByID(c.Request.Context(), id)
    if err != nil {
        c.Error(err)
        return
    }
    c.JSON(http.StatusOK, user)
}
```

## Middleware

```go
// Auth middleware
func AuthMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }
        // validate token...
        c.Set("userID", "parsed-user-id")
        c.Next()
    }
}

// Error middleware — register last
func errorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) == 0 {
            return
        }
        err := c.Errors.Last().Err
        status := http.StatusInternalServerError
        var apiErr *APIError
        if errors.As(err, &apiErr) {
            status = apiErr.Status
        }
        c.JSON(status, gin.H{"error": err.Error()})
    }
}
```

## Health Check Endpoint

```go
func healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
```

## Key gin.Context Methods

```go
// Binding
c.ShouldBindJSON(&req)        // JSON body → struct, returns error
c.ShouldBindQuery(&params)    // Query params → struct
c.ShouldBindUri(&uriParams)   // Path params → struct (use `uri` tag)

// Path / query params
id := c.Param("id")           // :id path segment
page := c.DefaultQuery("page", "1")

// Values set by middleware
userID, _ := c.Get("userID")

// Responses
c.JSON(http.StatusOK, payload)
c.JSON(http.StatusBadRequest, gin.H{"error": "msg"})
c.Status(http.StatusNoContent)
c.Abort()                      // stop handler chain
c.AbortWithStatusJSON(code, obj)
```

## Error Handling

- Use `c.Error(err)` in handlers to collect errors; handle them in a trailing middleware.
- Define a custom `APIError` type with `Status int` for HTTP-aware errors.
- Wrap business logic errors with `fmt.Errorf("context: %w", err)`.
- Never panic in handlers — `gin.Recovery()` catches panics but logs are noisy.

## Key Rules

- Use `gin.New()` not `gin.Default()` so middleware is explicit.
- Always call `c.Abort*` when short-circuiting, never rely on `return` alone.
- Use router groups (`r.Group(...)`) to apply middleware to a subset of routes.
- Read path params with `c.Param()`, query with `c.Query()`/`c.DefaultQuery()`.
- Prefer `ShouldBind*` (returns error) over `MustBind*` (calls Abort internally).
