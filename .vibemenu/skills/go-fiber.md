# Go + Fiber Skill Guide

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

```go
module github.com/your-org/service-name

go 1.22

require (
    github.com/gofiber/fiber/v2 v2.52.0
    github.com/jackc/pgx/v5 v5.5.0
)
```

## Server Setup

```go
package main

import (
    "log"
    "os"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
    app := fiber.New(fiber.Config{
        ErrorHandler: errorHandler,
    })

    app.Use(logger.New())
    app.Use(recover.New())

    registerRoutes(app)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(app.Listen(":" + port))
}

func errorHandler(c *fiber.Ctx, err error) error {
    code := fiber.StatusInternalServerError
    if e, ok := err.(*fiber.Error); ok {
        code = e.Code
    }
    return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}
```

## Handler Pattern

```go
package handler

import (
    "github.com/gofiber/fiber/v2"
    "your-org/service-name/internal/service"
)

type UserHandler struct {
    svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
    var req CreateUserRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
    }
    user, err := h.svc.Create(c.Context(), req)
    if err != nil {
        return err
    }
    return c.Status(fiber.StatusCreated).JSON(user)
}
```

## Health Check Endpoint

```go
app.Get("/health", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{"status": "ok"})
})
app.Get("/ready", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{"status": "ready"})
})
```

## Environment Variables

Always read config from environment variables:
```go
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" {
    log.Fatal("DATABASE_URL not set")
}
```

## Error Handling

- Return `fiber.NewError(statusCode, message)` for HTTP errors.
- Wrap business logic errors with `fmt.Errorf("context: %w", err)`.
- Never panic in handlers — use the recover middleware.
