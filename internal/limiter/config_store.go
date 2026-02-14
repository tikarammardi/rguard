package limiter

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type UserConfig struct {
	Rate     float64 `json:"rate"`
	Capacity float64 `json:"capacity"`
}

type ConfigStore struct {
	rdb *redis.Client
}

func NewConfigStore(rdb *redis.Client) *ConfigStore {
	return &ConfigStore{rdb: rdb}
}

func (s *ConfigStore) GetUserConfig(ctx context.Context, userId string) UserConfig {
	val, err := s.rdb.Get(ctx, "config:user:"+userId).Result()
	if err == nil {

		var cfg UserConfig
		if json.Unmarshal([]byte(val), &cfg) == nil {
			return cfg
		}
	}
	return UserConfig{Rate: 10, Capacity: 50}
}
