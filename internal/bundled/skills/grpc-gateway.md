---
name: grpc-gateway
description: gRPC-Gateway for HTTP/JSON transcoding of gRPC services in Go — proto annotations, buf codegen, server wiring, status mapping, auth, and Envoy alternative.
origin: vibemenu
---

# gRPC-Gateway

HTTP/JSON transcoding for gRPC services using `grpc-ecosystem/grpc-gateway/v2`. Lets REST clients consume gRPC services without a proxy rewrite.

## When to Activate

- Backend services use gRPC protocol (manifest `contracts.endpoints[].protocol = "gRPC"`)
- Exposing gRPC services to REST/browser clients
- Building API gateways that unify HTTP and gRPC traffic
- Generating OpenAPI specs from proto definitions

---

## Package Setup

Use the **exact versions from the "Dependency & API Reference" section** of your task prompt —
never hardcode version strings. The example below shows the package names only:

```go
// go.mod — required dependencies (versions from Dependency & API Reference)
require (
    google.golang.org/grpc              <version from Dependency & API Reference>
    google.golang.org/protobuf          <version from Dependency & API Reference>
    github.com/grpc-ecosystem/grpc-gateway/v2 <version from Dependency & API Reference>
    google.golang.org/genproto/googleapis/api <version from Dependency & API Reference>
)
```

Install codegen tools:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

---

## Proto Annotation Syntax

Annotate each RPC with `google.api.http` to define its HTTP binding.

```proto
syntax = "proto3";
package myapp.v1;
option go_package = "github.com/myorg/myapp/gen/go/myapp/v1;myappv1";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";

service UserService {
  // GET with path parameter
  rpc GetUser(GetUserRequest) returns (User) {
    option (google.api.http) = {
      get: "/v1/users/{id}"
    };
  }

  // POST with full body mapping
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (google.api.http) = {
      post: "/v1/users"
      body: "*"
    };
  }

  // PATCH with partial body mapping (body maps only the update_mask field)
  rpc UpdateUser(UpdateUserRequest) returns (User) {
    option (google.api.http) = {
      patch: "/v1/users/{id}"
      body: "user"  // maps request.user to the HTTP body
    };
  }

  // DELETE with no body
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/v1/users/{id}"
    };
  }

  // Additional HTTP bindings (aliases)
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {
    option (google.api.http) = {
      get: "/v1/users"
      additional_bindings {
        get: "/v1/orgs/{org_id}/users"
      }
    };
  }
}

message GetUserRequest  { string id = 1; }
message DeleteUserRequest { string id = 1; }
message CreateUserRequest {
  string name  = 1;
  string email = 2;
}
message UpdateUserRequest {
  string id   = 1;
  User   user = 2;
}
message User {
  string id    = 1;
  string name  = 2;
  string email = 3;
}
```

---

## Code Generation with buf

Prefer `buf` over raw `protoc` — it handles imports automatically.

```yaml
# buf.yaml — proto module config
version: v2
deps:
  - buf.build/googleapis/googleapis  # provides google/api/annotations.proto
```

```yaml
# buf.gen.yaml — code generation config
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative

  - remote: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false

  - remote: buf.build/grpc-ecosystem/gateway
    out: gen/go
    opt:
      - paths=source_relative
      - generate_unbound_methods=true  # generate HTTP handlers for all RPCs

  - remote: buf.build/grpc-ecosystem/openapiv2
    out: gen/openapi
    opt:
      - logtostderr=true
```

```bash
# Generate all protos
buf generate
# Lint protos
buf lint
# Check for breaking changes vs main branch
buf breaking --against '.git#branch=main'
```

---

## Server Wiring

Run gRPC and HTTP on separate ports. This is simpler and more compatible than port-sharing via `cmux`.

```go
package main

import (
    "context"
    "net"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    pb "github.com/myorg/myapp/gen/go/myapp/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    ctx := context.Background()

    // 1. Start gRPC server
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            authInterceptor,
            loggingInterceptor,
        ),
    )
    pb.RegisterUserServiceServer(grpcServer, &UserServiceImpl{})
    reflection.Register(grpcServer) // enable grpcurl / Postman

    grpcLis, _ := net.Listen("tcp", ":9090")
    go grpcServer.Serve(grpcLis)

    // 2. Start HTTP/JSON gateway
    mux := runtime.NewServeMux(
        runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
        runtime.WithErrorHandler(customErrorHandler),
    )

    opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
    pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:9090", opts)

    httpServer := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }
    httpServer.ListenAndServe()
}
```

**Single-port via `cmux`** (use only when a load balancer requires it):
```go
import "github.com/soheilhy/cmux"

l, _ := net.Listen("tcp", ":443")
m := cmux.New(l)
grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
httpL := m.Match(cmux.HTTP1Fast())
go grpcServer.Serve(grpcL)
go httpServer.Serve(httpL)
m.Serve()
```

---

## gRPC Reflection

Register reflection so `grpcurl`, Postman, and Evans can discover services without proto files.

```go
import "google.golang.org/grpc/reflection"

// After registering all services:
reflection.Register(grpcServer)
```

```bash
# List services
grpcurl -plaintext localhost:9090 list
# Describe service
grpcurl -plaintext localhost:9090 describe myapp.v1.UserService
# Call RPC
grpcurl -plaintext -d '{"id":"123"}' localhost:9090 myapp.v1.UserService/GetUser
```

---

## Status Code Mapping

gRPC status codes translate to HTTP status codes automatically. Know the mapping to avoid surprises.

| gRPC Code | HTTP Status | When to use |
|-----------|-------------|-------------|
| `codes.OK` | `200` | Success |
| `codes.InvalidArgument` | `400` | Bad request / validation error |
| `codes.Unauthenticated` | `401` | Missing or invalid credentials |
| `codes.PermissionDenied` | `403` | Valid credentials, insufficient permissions |
| `codes.NotFound` | `404` | Resource does not exist |
| `codes.AlreadyExists` | `409` | Conflict / duplicate resource |
| `codes.ResourceExhausted` | `429` | Rate limit exceeded |
| `codes.Internal` | `500` | Unexpected server error |
| `codes.Unimplemented` | `501` | RPC not implemented |
| `codes.Unavailable` | `503` | Service temporarily down |
| `codes.DeadlineExceeded` | `504` | Request timed out |

**Return rich errors from your service implementation:**
```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/genproto/googleapis/rpc/errdetails"
)

func (s *UserServiceImpl) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    if req.Id == "" {
        return nil, status.Errorf(codes.InvalidArgument, "id is required")
    }
    user, err := s.repo.GetByID(ctx, req.Id)
    if errors.Is(err, ErrNotFound) {
        return nil, status.Errorf(codes.NotFound, "user %q not found", req.Id)
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to fetch user: %v", err)
    }
    return toProtoUser(user), nil
}
```

**Custom error handler for HTTP gateway:**
```go
func customErrorHandler(
    ctx context.Context,
    mux *runtime.ServeMux,
    marshaler runtime.Marshaler,
    w http.ResponseWriter,
    r *http.Request,
    err error,
) {
    s, _ := status.FromError(err)
    // Add request-id to error response
    w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
    runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
    _ = s // use for custom logging if needed
}
```

---

## Authentication: HTTP Header → gRPC Metadata

Pass JWT from the HTTP `Authorization` header into gRPC metadata so interceptors can read it.

```go
// Custom header matcher — passes Authorization and X-Request-ID to gRPC metadata
func customHeaderMatcher(key string) (string, bool) {
    switch strings.ToLower(key) {
    case "authorization":
        return "authorization", true
    case "x-request-id":
        return "x-request-id", true
    default:
        return runtime.DefaultHeaderMatcher(key)
    }
}

// Register with ServeMux:
mux := runtime.NewServeMux(
    runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
)

// In gRPC server interceptor — read from metadata
func authInterceptor(ctx context.Context, req interface{},
    info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }
    authHeader := md.Get("authorization")
    if len(authHeader) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing authorization header")
    }
    token := strings.TrimPrefix(authHeader[0], "Bearer ")
    claims, err := validateJWT(token)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }
    ctx = context.WithValue(ctx, userClaimsKey{}, claims)
    return handler(ctx, req)
}
```

---

## Streaming Limitations

**Server-side streaming** — supported via grpc-gateway:
- HTTP response is a newline-delimited JSON stream
- Each message is one JSON object followed by `\n`
- Clients read the stream incrementally

**Client-side streaming** — NOT supported:
- grpc-gateway cannot buffer the full HTTP body and convert it to a stream
- Workaround: use a single POST with a repeated field instead

**Bidirectional streaming** — NOT supported:
- WebSocket is needed for bidirectional; grpc-gateway is HTTP/1.1 only
- Use WebSocket alongside gRPC, not as a replacement

```proto
// Server streaming — works with grpc-gateway
rpc WatchEvents(WatchRequest) returns (stream Event) {
    option (google.api.http) = { get: "/v1/events/watch" };
}
// Client streaming — avoid; use a batch RPC instead
// rpc BatchCreate(stream CreateRequest) returns (BatchResponse) {}
```

---

## Envoy gRPC-JSON Transcoding (Alternative)

Use Envoy transcoding instead of grpc-gateway when you already run Envoy as a sidecar or gateway (e.g., Istio service mesh).

```yaml
# envoy.yaml — filter config
http_filters:
  - name: envoy.filters.http.grpc_json_transcoder
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder
      proto_descriptor: /etc/envoy/api.pb   # compiled binary descriptor
      services:
        - myapp.v1.UserService
      print_options:
        add_whitespace: true
        always_print_primitive_fields: true
      request_validation_options:
        reject_unknown_query_parameters: true
```

Generate the binary descriptor:
```bash
buf build --as-file-descriptor-set -o api.pb
```

**grpc-gateway vs Envoy transcoding:**
| | grpc-gateway | Envoy transcoding |
|---|---|---|
| Setup | Go library, in-process | Sidecar config |
| Proto descriptor | Embedded in binary | File on disk |
| Custom middleware | Go interceptors | Envoy filters |
| Good for | Simple setups, no Envoy | Kubernetes/Istio deployments |

---

## Anti-Patterns to Avoid

- **Generating `*.pb.go` without `buf`**: Using raw `protoc` without `buf.yaml` leads to import path mismatches. Always use `buf generate`.
- **Returning bare `error` from service methods**: Always wrap with `status.Errorf` — bare errors become `codes.Unknown` (HTTP 500).
- **Registering reflection in production without auth**: `grpcurl` can enumerate all endpoints. Add an auth check or disable reflection behind a build tag.
- **Ignoring `codes.DeadlineExceeded`**: Propagate deadline from incoming context: `ctx, cancel := context.WithTimeout(ctx, 5*time.Second); defer cancel()`.
- **Bidirectional streaming over grpc-gateway**: It silently fails. Document clearly which RPCs are gateway-compatible.
