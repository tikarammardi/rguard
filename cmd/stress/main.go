package main

import (
	"context"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"rate-limiter-engine/proto"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}()
	client := proto.NewRateLimiterClient(conn)

	var wg sync.WaitGroup
	numRequests := 100
	userID := "stress_user"

	log.Printf("Starting stress test with %d concurrent requests...", numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			resp, err := client.CheckLimit(ctx, &proto.LimitRequest{UserId: userID})
			if err != nil {
				log.Printf("Worker %d failed: %v", id, err)
				return
			}

			if resp.Allowed {
				log.Printf("Worker %d: ALLOWED", id)
			} else {
				log.Printf("Worker %d: BLOCKED", id)
			}
		}(i)
	}

	wg.Wait()
	log.Println("Stress test complete.")
}
