package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"rate-limiter-engine/internal/limiter"
	"rate-limiter-engine/proto"
	"syscall"

	"google.golang.org/grpc"
)

type server struct {
	proto.UnimplementedRateLimiterServer
	guard *limiter.Guard
}

func (s *server) Check(ctx context.Context, req *proto.LimitRequest) (*proto.LimitResponse, error) {
	allowed := s.guard.Allow(req.UserId)

	return &proto.LimitResponse{
		Allowed:   allowed,
		Remaining: 0,
	}, nil
}

func main() {
	g := limiter.NewGuard(100)

	grpcServer := grpc.NewServer()
	proto.RegisterRateLimiterServer(grpcServer, &server{
		guard: g,
	})

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	go func() {
		log.Printf("server listening at %v", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server stopped serving: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
}
