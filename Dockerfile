FROM golang:1.24-alpine AS builder
WORKDIR /app

# Install protoc and build tools
RUN apk add --no-cache protobuf protobuf-dev make

# Cache dependencies by copying only go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Install protobuf Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy the rest of the source
COPY . .

# Generate protobuf files
RUN make generate

# Build a static, stripped binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /guard ./cmd/guard/main.go


FROM alpine:3.18
# Install CA certs for TLS and create an unprivileged user in one layer
RUN apk --no-cache add ca-certificates \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app
COPY --from=builder /guard /app/guard

# Run as non-root user
USER app

EXPOSE 50051

ENTRYPOINT ["/app/guard"]
