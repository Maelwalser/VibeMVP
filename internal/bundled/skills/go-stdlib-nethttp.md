# Go + net/http Skill Guide

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
│   └── middleware/  # Middleware wrappers
└── Dockerfile
```

## go.mod Boilerplate

Use the **exact module path and versions from the "Dependency & API Reference" section** of your
task prompt — never hardcode version strings. The example below shows the package names only:

```go
module <module_path from payload>

go <version from Dependency & API Reference>

require (
    github.com/jackc/pgx/v5 <version from Dependency & API Reference>
)
```

No external HTTP framework — standard library only.

## Server Setup

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    mux := http.NewServeMux()
    registerRoutes(mux)

    // Wrap the mux with shared middleware
    handler := loggingMiddleware(recoverMiddleware(mux))

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      handler,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go gracefulShutdown(srv)

    log.Printf("listening on %s", srv.Addr)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server error: %v", err)
    }
}

func gracefulShutdown(srv *http.Server) {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("forced shutdown: %v", err)
    }
}

func registerRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /health", healthCheck)
    mux.HandleFunc("GET /api/v1/users", listUsers)
    mux.HandleFunc("POST /api/v1/users", createUser)
    mux.HandleFunc("GET /api/v1/users/{id}", getUser)   // Go 1.22+ pattern syntax
    mux.HandleFunc("PUT /api/v1/users/{id}", updateUser)
    mux.HandleFunc("DELETE /api/v1/users/{id}", deleteUser)
}
```

## Handler Pattern

```go
package handler

import (
    "encoding/json"
    "fmt"
    "net/http"
    "your-org/service-name/internal/service"
)

type UserHandler struct {
    svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := readJSON(r, &req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    user, err := h.svc.Create(r.Context(), req)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "could not create user")
        return
    }
    writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id") // Go 1.22+
    user, err := h.svc.GetByID(r.Context(), id)
    if err != nil {
        writeError(w, http.StatusNotFound, fmt.Sprintf("user %s not found", id))
        return
    }
    writeJSON(w, http.StatusOK, user)
}
```

## JSON Helpers

```go
func readJSON(r *http.Request, dst any) error {
    r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MB limit
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        log.Printf("writeJSON encode error: %v", err)
    }
}

func writeError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{"error": msg})
}
```

## Middleware

```go
// Middleware wraps http.Handler — use closures for dependencies.
type Middleware func(http.Handler) http.Handler

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}

func recoverMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                log.Printf("panic: %v", rec)
                http.Error(w, "internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// Auth middleware using context values
type contextKey string
const ctxKeyUserID contextKey = "userID"

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            writeError(w, http.StatusUnauthorized, "missing token")
            return
        }
        ctx := context.WithValue(r.Context(), ctxKeyUserID, "parsed-id")
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Chain multiple middleware
func chain(h http.Handler, middlewares ...Middleware) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}
```

## Health Check

```go
func healthCheck(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

## http.Handle vs HandleFunc

```go
// http.Handle — registers any type implementing http.Handler
mux.Handle("GET /users", http.HandlerFunc(listUsers))
mux.Handle("GET /users", myHandlerStruct) // implements ServeHTTP

// http.HandleFunc — convenience wrapper for plain functions
mux.HandleFunc("GET /users", listUsers)

// Go 1.22+ method+path pattern in the mux (preferred):
mux.HandleFunc("GET /api/v1/users/{id}", getUser)
```

## Key Rules

- Use `http.NewServeMux()` — never use `http.DefaultServeMux` (global state, security risk).
- Always set `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` on `http.Server`.
- Use `http.MaxBytesReader` when decoding request bodies to prevent DoS.
- Pass context through `r.Context()` — never store context in structs.
- Use `r.PathValue("key")` (Go 1.22+) for path parameters; fall back to a router for older versions.
- Wrap errors with `fmt.Errorf("context: %w", err)` before logging or propagating.
- Implement graceful shutdown with `srv.Shutdown(ctx)` — always wait for in-flight requests.
