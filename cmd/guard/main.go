package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"rate-limiter-engine/internal/limiter"
	"syscall"

	"google.golang.org/grpc"
	"rate-limiter-engine/github.com/tikarammardi/rate-limiter-engine/proto"
)

// server wraps our business logic to satisfy the gRPC interface
type server struct {
	proto.UnimplementedRateLimiterServer
	guard *limiter.Guard
}

// Check is the gRPC method called by clients
func (s *server) Check(ctx context.Context, req *proto.LimitRequest) (*proto.LimitResponse, error) {
	// Our Guard logic remains the same regardless of the storage backend
	allowed := s.guard.Allow(ctx, req.UserId)
	return &proto.LimitResponse{Allowed: allowed}, nil
}

func main() {
	// --- 1. CONFIGURATION ---
	// In a real app, these might come from environment variables
	const (
		port       = ":50051"
		refillRate = 10.0 // tokens per second (600 per minute)
		capacity   = 50.0 // allow bursts of 50 requests
	)

	// --- 2. DEPENDENCY INJECTION ---
	// To switch to Redis, you would replace this line with:
	// store := limiter.NewRedisStore(redisClient, refillRate, capacity)
	store := limiter.NewMemoryStore(refillRate, capacity)

	g := limiter.NewGuard(store)

	// --- 3. gRPC SERVER SETUP ---
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterRateLimiterServer(s, &server{guard: g})

	// --- 4. GRACEFUL SHUTDOWN ---
	// This ensures we don't drop requests when stopping the service
	go func() {
		log.Printf("🛡️ Guard service listening on %s", port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("❌ Failed to serve: %v", err)
		}
	}()

	// Wait for control-c or terminate signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	s.GracefulStop()
}
