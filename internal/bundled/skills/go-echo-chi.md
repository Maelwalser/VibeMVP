# Go + Echo / Chi Skill Guide

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
    github.com/labstack/echo/v4 <version from Dependency & API Reference>    // Echo
    github.com/go-chi/chi/v5 <version from Dependency & API Reference>       // Chi
    github.com/jackc/pgx/v5 <version from Dependency & API Reference>
)
```

---

## Echo

### Server Setup

```go
package main

import (
    "log"
    "net/http"
    "os"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    e := echo.New()
    e.HideBanner = true

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.RequestID())

    e.HTTPErrorHandler = customErrorHandler

    registerEchoRoutes(e)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(e.Start(":" + port))
}

func customErrorHandler(err error, c echo.Context) {
    code := http.StatusInternalServerError
    msg := err.Error()
    if he, ok := err.(*echo.HTTPError); ok {
        code = he.Code
        msg = fmt.Sprintf("%v", he.Message)
    }
    c.JSON(code, map[string]string{"error": msg})
}

func registerEchoRoutes(e *echo.Echo) {
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    api := e.Group("/api/v1")
    api.Use(authMiddleware)

    users := api.Group("/users")
    users.GET("", listUsers)
    users.POST("", createUser)
    users.GET("/:id", getUser)
}
```

### Echo Handler Pattern

```go
func createUser(c echo.Context) error {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    if err := c.Validate(&req); err != nil {
        return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
    }
    user, err := svc.Create(c.Request().Context(), req)
    if err != nil {
        return fmt.Errorf("create user: %w", err)
    }
    return c.JSON(http.StatusCreated, user)
}

func getUser(c echo.Context) error {
    id := c.Param("id")
    page := c.QueryParam("page")
    _ = page
    user, err := svc.GetByID(c.Request().Context(), id)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "user not found")
    }
    return c.JSON(http.StatusOK, user)
}
```

### Echo Middleware

```go
func authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := c.Request().Header.Get("Authorization")
        if token == "" {
            return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
        }
        c.Set("userID", "parsed-id")
        return next(c)
    }
}
```

---

## Chi

### Server Setup

```go
package main

import (
    "log"
    "net/http"
    "os"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

func main() {
    r := chi.NewRouter()

    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.RequestID)

    registerChiRoutes(r)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(http.ListenAndServe(":"+port, r))
}

func registerChiRoutes(r chi.Router) {
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok"}`))
    })

    r.Route("/api/v1", func(r chi.Router) {
        r.Use(authMiddlewareChi)

        r.Route("/users", func(r chi.Router) {
            r.Get("/", listUsers)
            r.Post("/", createUser)
            r.Route("/{id}", func(r chi.Router) {
                r.Get("/", getUser)
                r.Put("/", updateUser)
                r.Delete("/", deleteUser)
            })
        })
    })
}
```

### Chi Handler Pattern

```go
func createUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid body", http.StatusBadRequest)
        return
    }
    user, err := svc.Create(r.Context(), req)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    user, err := svc.GetByID(r.Context(), id)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

### Chi Middleware

```go
func authMiddlewareChi(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), ctxKeyUserID, "parsed-id")
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## Key Rules

- **Echo**: Use `c.Bind()` for request decoding (supports JSON, form, query). Return `echo.NewHTTPError(code, msg)` from handlers for HTTP errors.
- **Chi**: Uses plain `net/http` — read path params with `chi.URLParam(r, "key")`, write JSON manually or via a helper.
- Both support nested route groups with per-group middleware via `Group`/`Route`.
- Wrap business logic errors with `fmt.Errorf("context: %w", err)` before returning.
- Never swallow errors — always write a response or return the error.
