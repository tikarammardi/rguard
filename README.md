# rguard

A high-performance, distributed rate limiting service built in Go, powered by Redis and gRPC. rguard implements the Token Bucket algorithm with microsecond precision and horizontal scalability.

## 🚀 Features

- **Atomic Lua Scripting**: Prevents race conditions across multiple server nodes by executing token logic inside Redis.
- **Microsecond Precision**: High-resolution refill math to eliminate rounding errors and ensure fair throughput.
- **gRPC Interceptors**: Decouples rate-limiting logic from business logic. Simply plug in the middleware.
- **Dynamic Configuration**: Support for per-user limits (tiers) that can be updated in Redis without restarting the service.
- **Observability**: Structured JSON logging and built-in metadata propagation (X-RateLimit-Remaining, X-RateLimit-Reset).
- **Fail-Open Design**: Ensures that if Redis goes down, your API remains available (Resiliency first).

## 🏗️ Architecture

The project is designed with a strict separation of concerns:

- **Interceptor Layer**: Extracts identity (Header/IP) and handles the gRPC lifecycle.
- **Guard**: The orchestrator that fetches user configuration and coordinates with the store.
- **Config Store**: Manages dynamic limits (Rate & Capacity) retrieved from a fast-access cache.

## Prerequisites

- Go 1.24+
- Redis 7.0+
- Docker & Docker Compose (for containerized setup)
- protoc (for regenerating proto files if needed)

---

## 🚀 Quick Start

### Option 1: Using Docker Compose (Recommended)

The easiest way to run the full stack (Redis + rguard):

```bash
# Start all services
make docker-up

# View logs
make docker-logs

# Stop all services
make docker-down
```

Or without Make:

```bash
docker compose up --build -d
docker compose logs -f
docker compose down
```

### Option 2: Run Locally (with external Redis)

1. **Start Redis** (if not already running):

```bash
docker run --name redis -p 6379:6379 -d redis:7-alpine
```

2. **Set environment variables** (optional, defaults shown):

```bash
export REDIS_ADDR=127.0.0.1:6379
export REDIS_PASSWORD=
```

3. **Build and run**:

```bash
make run
```

Or without Make:

```bash
go run cmd/guard/main.go
```

### Option 3: Run Docker Container Standalone

```bash
# Build and run (connects to host Redis)
make docker-run
```

---

## 🧪 Testing the Service

Once the service is running, test it with `grpcurl`:

```bash
# Install grpcurl if needed: brew install grpcurl

# Check rate limit for a user
grpcurl -plaintext \
  -rpc-header "user-id: dev-user-1" \
  -d '{"user_id": "dev-user-1"}' \
  localhost:50051 ratelimiter.RateLimiter/CheckLimit
```

Run the example client:

```bash
go run cmd/client/main.go
```

Run the stress test:

```bash
go run cmd/stress/main.go
```

---

## 📦 Makefile Reference

| Command | Description |
|---------|-------------|
| `make build` | Build the guard binary locally (output: `bin/guard`) |
| `make run` | Build and run guard locally |
| `make test` | Run all tests with race detection and coverage |
| `make lint` | Run golangci-lint (auto-installs if missing) |
| `make generate` | Generate protobuf Go code |
| `make docker-build` | Build Docker image |
| `make docker-run` | Build and run container standalone |
| `make docker-up` | Start all services with docker compose |
| `make docker-down` | Stop and remove docker compose services |
| `make docker-logs` | Tail logs from docker compose services |
| `make clean` | Remove build artifacts |
| `make help` | Show all available targets |

### Examples

```bash
# Build the binary
make build

# Run tests
make test

# Lint the code
make lint

# Regenerate protobuf files
make generate

# Full Docker workflow
make docker-up      # Start Redis + app
make docker-logs    # Watch logs
make docker-down    # Tear down
```

---

## 📁 Project Layout

```
.
├── cmd/
│   ├── guard/main.go        # gRPC server entrypoint
│   ├── client/main.go       # Simple test client
│   └── stress/main.go       # Stress test client
├── internal/
│   ├── interceptors/
│   │   └── ratelimit.go     # gRPC rate limit interceptor
│   └── limiter/
│       ├── config_store.go  # Dynamic per-user config from Redis
│       ├── guard.go         # Rate limit orchestrator
│       ├── redis_store.go   # Redis-backed token bucket
│       └── memory_store.go  # In-memory store for testing
├── proto/
│   └── rate_limit.proto     # Protobuf service definition
├── Dockerfile               # Multi-stage Docker build
├── docker-compose.yaml      # Full stack: Redis + app
├── Makefile                 # Build, test, and Docker commands
└── README.md
```

---

## ⚙️ Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis server address |
| `REDIS_PASSWORD` | *(empty)* | Redis password (if required) |

---

## 📝 Behavior Notes

- **Fail-Open**: If Redis is unavailable or a config key is missing, the service falls back to safe defaults. This preserves availability. To enable fail-closed semantics, modify `internal/limiter/config_store.go`.
- **Dynamic Config**: The `ConfigStore.GetUserConfig` function returns a default config when Redis is unreachable. Ensure `ConfigStore` is wired with a Redis client in `cmd/guard/main.go` for per-user configuration.

---

## 🔌 How to Plug It Into Your Service

The rate limiter runs as a **standalone gRPC service**. Any service can call it before processing requests.

### Go

```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
client := proto.NewRateLimiterClient(conn)

resp, err := client.CheckLimit(ctx, &proto.LimitRequest{UserId: "user-123"})
if err != nil || !resp.Allowed {
    return errors.New("rate limit exceeded!")
}
```

### Python

```python
import grpc
import rate_limit_pb2
import rate_limit_pb2_grpc

channel = grpc.insecure_channel("localhost:50051")
client = rate_limit_pb2_grpc.RateLimiterStub(channel)

response = client.CheckLimit(rate_limit_pb2.LimitRequest(user_id="user-123"))
if not response.allowed:
    raise Exception("Rate limit exceeded!")
```

### Node.js

```javascript
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");

const packageDef = protoLoader.loadSync("rate_limit.proto");
const proto = grpc.loadPackageDefinition(packageDef).ratelimiter;

const client = new proto.RateLimiter(
  "localhost:50051",
  grpc.credentials.createInsecure()
);

client.CheckLimit({ user_id: "user-123" }, (err, response) => {
  if (err || !response.allowed) {
    throw new Error("Rate limit exceeded!");
  }
});
```

### cURL (via grpcurl)

```bash
grpcurl -plaintext localhost:50051 list

grpcurl -plaintext \
  -d '{"user_id": "user-123"}' \
  localhost:50051 ratelimiter.RateLimiter/CheckLimit
```

### Generate Client Stubs

Use `protoc` to generate client code for your language:

```bash
# Python
pip install grpcio grpcio-tools
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/rate_limit.proto

# Node.js (uses proto file directly, no codegen needed)
npm install @grpc/grpc-js @grpc/proto-loader

# Java
protoc --java_out=. --grpc-java_out=. proto/rate_limit.proto
```

### Configure Per-User Limits

```bash
redis-cli HSET "ratelimit:config:user-123" rate 100 capacity 1000
```

---

## 🔧 Development

### Regenerate Protobuf Files

```bash
make generate
```

Or manually:

```bash
protoc --go_out=. --go-grpc_out=. proto/rate_limit.proto
```

### Lint Code

```bash
make lint
```