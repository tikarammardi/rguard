package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"rate-limiter-engine/internal/interceptors" // Ensure this path is correct
	"rate-limiter-engine/internal/limiter"
	"syscall"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"rate-limiter-engine/github.com/tikarammardi/rate-limiter-engine/proto"
)

type server struct {
	proto.UnimplementedRateLimiterServer
}

// CheckLimit now only contains business logic because the Interceptor
// handles the Allow/Block logic before this is even reached.
func (s *server) CheckLimit(ctx context.Context, req *proto.LimitRequest) (*proto.LimitResponse, error) {
	return &proto.LimitResponse{Allowed: true}, nil
}

func main() {
	// 1. REDIS SETUP
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// Verify connection
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("❌ Redis unreachable: %v", err)
	}

	// 2. CONFIGURATION & LOGGER
	const (
		port       = ":50051"
		refillRate = 10.0
		capacity   = 50.0
	)

	// Using JSON handler for structured logging (Production Standard)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 3. DEPENDENCY INJECTION
	store := limiter.NewRedisStore(rdb, refillRate, capacity)
	g := limiter.NewGuard(store, logger)

	// 4. gRPC SERVER SETUP with INTERCEPTOR
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	// Register the RateLimit Interceptor here! 🛡️
	s := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryRateLimitInterceptor(g)),
	)

	proto.RegisterRateLimiterServer(s, &server{})

	// Register reflection api
	reflection.Register(s)

	// 5. RUN & GRACEFUL SHUTDOWN
	go func() {
		log.Printf("🛡️ Guard service listening on %s", port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("❌ Failed to serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	s.GracefulStop()
}
