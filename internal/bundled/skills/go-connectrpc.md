# Go + ConnectRPC Skill Guide

## Project Layout

```
service-name/
├── go.mod
├── go.sum
├── buf.yaml
├── buf.gen.yaml
├── proto/
│   └── user/v1/
│       └── user.proto
├── gen/                     # buf generate output (commit this)
│   └── user/v1/
│       ├── user.pb.go
│       └── userv1connect/
│           └── user.connect.go
├── internal/
│   ├── server/              # ConnectRPC handler implementations
│   └── service/             # Business logic
└── main.go
```

## go.mod Boilerplate

Use the **exact module path and versions from the "Dependency & API Reference" section** of your
task prompt — never hardcode version strings. The example below shows the package names only:

```go
module <module_path from payload>

go <version from Dependency & API Reference>

require (
    connectrpc.com/connect <version from Dependency & API Reference>
    google.golang.org/protobuf <version from Dependency & API Reference>
    golang.org/x/net <version from Dependency & API Reference>       // for h2c
    github.com/jackc/pgx/v5 <version from Dependency & API Reference>
)
```

## buf.yaml

```yaml
version: v2
modules:
  - path: proto
deps:
  - buf.build/googleapis/googleapis
```

## buf.gen.yaml

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/connectrpc/go
    out: gen
    opt: paths=source_relative
```

Generate with: `buf generate`

## Proto Definition

```protobuf
syntax = "proto3";
package user.v1;
option go_package = "github.com/your-org/service-name/gen/user/v1;userv1";

service UserService {
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
}

message GetUserRequest  { string id = 1; }
message GetUserResponse { User user = 1; }
message User {
    string id   = 1;
    string name = 2;
    string email = 3;
}
// ... other messages
```

## Handler Implementation

```go
package server

import (
    "context"
    "connectrpc.com/connect"
    userv1 "github.com/your-org/service-name/gen/user/v1"
    "github.com/your-org/service-name/gen/user/v1/userv1connect"
    "github.com/your-org/service-name/internal/service"
)

// UserServer implements the generated userv1connect.UserServiceHandler interface.
type UserServer struct {
    svc *service.UserService
}

func NewUserServer(svc *service.UserService) *UserServer {
    return &UserServer{svc: svc}
}

func (s *UserServer) GetUser(
    ctx context.Context,
    req *connect.Request[userv1.GetUserRequest],
) (*connect.Response[userv1.GetUserResponse], error) {
    user, err := s.svc.GetByID(ctx, req.Msg.Id)
    if err != nil {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }
    return connect.NewResponse(&userv1.GetUserResponse{
        User: &userv1.User{Id: user.ID, Name: user.Name, Email: user.Email},
    }), nil
}

func (s *UserServer) CreateUser(
    ctx context.Context,
    req *connect.Request[userv1.CreateUserRequest],
) (*connect.Response[userv1.CreateUserResponse], error) {
    if req.Msg.Name == "" {
        return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
    }
    user, err := s.svc.Create(ctx, req.Msg.Name, req.Msg.Email)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }
    return connect.NewResponse(&userv1.CreateUserResponse{
        User: &userv1.User{Id: user.ID, Name: user.Name},
    }), nil
}
```

## Server Setup (h2c — HTTP/2 cleartext)

```go
package main

import (
    "log"
    "net/http"
    "os"

    "connectrpc.com/connect"
    "golang.org/x/net/http2"
    "golang.org/x/net/http2/h2c"

    "github.com/your-org/service-name/gen/user/v1/userv1connect"
    "github.com/your-org/service-name/internal/server"
    "github.com/your-org/service-name/internal/service"
)

func main() {
    svc := service.NewUserService()
    userSrv := server.NewUserServer(svc)

    mux := http.NewServeMux()

    // Register handler — supports gRPC, gRPC-Web, and Connect protocols
    path, handler := userv1connect.NewUserServiceHandler(
        userSrv,
        connect.WithInterceptors(loggingInterceptor()),
    )
    mux.Handle(path, handler)

    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"status":"ok"}`))
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Printf("listening on :%s", port)
    log.Fatal(http.ListenAndServe(
        ":"+port,
        h2c.NewHandler(mux, &http2.Server{}), // enables HTTP/2 without TLS
    ))
}
```

## Interceptors

```go
func loggingInterceptor() connect.UnaryInterceptorFunc {
    return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            log.Printf("→ %s", req.Spec().Procedure)
            resp, err := next(ctx, req)
            if err != nil {
                log.Printf("✗ %s: %v", req.Spec().Procedure, err)
            }
            return resp, err
        }
    })
}

// Auth interceptor example
func authInterceptor(secret string) connect.UnaryInterceptorFunc {
    return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            token := req.Header().Get("Authorization")
            if token == "" {
                return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing token"))
            }
            return next(ctx, req)
        }
    })
}
```

## Connect Error Codes

```go
connect.CodeNotFound          // 404 equivalent
connect.CodeInvalidArgument   // 400 equivalent
connect.CodeUnauthenticated   // 401 equivalent
connect.CodePermissionDenied  // 403 equivalent
connect.CodeInternal          // 500 equivalent
connect.CodeAlreadyExists     // 409 equivalent
connect.CodeUnimplemented     // 501 equivalent

// Create errors:
connect.NewError(connect.CodeNotFound, fmt.Errorf("user %s not found", id))
```

## Key Rules

- Run `buf generate` after every `.proto` change — commit generated files.
- Handlers implement the generated `*connect.XxxServiceHandler` interface — do not hand-write signatures.
- Use `h2c.NewHandler` for cleartext HTTP/2 (development/internal); use TLS + standard `http.ListenAndServeTLS` in production.
- Access request metadata via `req.Header()`, set response metadata via `resp.Header()`.
- Wrap business logic errors with `connect.NewError(code, err)` — never return raw errors from handlers.
- Interceptors compose with `connect.WithInterceptors(a, b, c)` — outermost interceptor runs first.
