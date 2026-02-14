package limiter

import (
	"context"
	"fmt"
	"log/slog"
)

type Guard struct {
	store  LimiterStore
	logger *slog.Logger
}

func NewGuard(store LimiterStore, logger *slog.Logger) *Guard {
	return &Guard{store: store, logger: logger}
}

func (g *Guard) Check(ctx context.Context, userId string, rate float64, cap float64) Result {

	key := fmt.Sprintf("user:%s", userId)

	allowed, err := g.store.Take(ctx, key, 1, rate, cap)
	if err != nil {

		g.logger.Error("rate_limit_store_error", "ërror", err, "userId", userId)
		// Fail-open: allow request but return empty metadata
		return Result{
			Allowed: true,
		}
	}

	return allowed
}
