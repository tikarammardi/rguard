package main

import (
	"context"
	"log"
	"net"
	"rate-limiter-engine/internal/limiter"
	"rate-limiter-engine/proto"

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
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterRateLimiterServer(s, &server{
		guard: limiter.NewGuard(100),
	})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
