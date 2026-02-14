package interceptors

import (
	"context"
	"rate-limiter-engine/internal/limiter"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryRateLimitInterceptor(g *limiter.Guard, configStore *limiter.ConfigStore) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		// 1. Extract Identity (In production, get this from JWT/Metadata)
		// For this example, we assume the user ID is passed in context or a specific header
		userID := "anonymous"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("user-id"); len(ids) > 0 {
				userID = ids[0]
			}
		}

		userCfg := configStore.GetUserConfig(ctx, userID)

		// 2. Perform the Rate Limit Check
		res := g.Check(ctx, userID, userCfg.Rate, userCfg.Capacity)

		// 3. Attach standard headers to the response
		header := metadata.Pairs(
			"x-ratelimit-remaining", strconv.Itoa(res.Remaining),
			"x-ratelimit-reset", strconv.FormatInt(res.Reset, 10),
		)
		grpc.SetHeader(ctx, header)

		// 4. Block if not allowed
		if !res.Allowed {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded, retry after %d", res.Reset)
		}

		// 5. Proceed to the actual service logic
		return handler(ctx, req)
	}
}
