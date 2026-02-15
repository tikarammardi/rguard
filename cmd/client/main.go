package main

import (
	"context"
	"log"

	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"rate-limiter-engine/proto"
)

func main() {
	// 1. Establish a connection to the server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}()
	c := proto.NewRateLimiterClient(conn)

	// 2. Simulate 5 quick requests for "user_123"
	userID := "user_123"
	for i := 1; i <= 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		r, err := c.CheckLimit(ctx, &proto.LimitRequest{UserId: userID})
		if err != nil {
			log.Fatalf("could not check: %v", err)
		}

		log.Printf("Request %d: Allowed = %v", i, r.Allowed)

		// Optional: slight delay to see it happen in real-time
		time.Sleep(100 * time.Millisecond)
	}
}
