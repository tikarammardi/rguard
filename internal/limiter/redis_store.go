package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const luaTokenBucket = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])

local bucket = redis.call('HMGET', key, 'tokens', 'last_time')
local tokens = tonumber(bucket[1]) or capacity
local last_time = tonumber(bucket[2]) or now

local delta = math.max(0, now - last_time)
tokens = math.min(capacity, tokens + (delta * rate))

local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

redis.call('HMSET', key, 'tokens', tokens, 'last_time', now)
redis.call('EXPIRE', key, 60)

-- Calculate reset: how many seconds until the bucket is full?
local missing = capacity - tokens
local reset_in = math.ceil(missing / rate)

return {allowed, math.floor(tokens), now + reset_in}
`

type RedisStore struct {
	rdb  *redis.Client
	rate float64
	cap  float64
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func (r *RedisStore) Take(ctx context.Context, key string, amount int, rate float64, cap float64) (Result, error) {
	now := time.Now().Unix()

	// Lua returns an array: [allowed, remaining, reset_timestamp]
	res, err := r.rdb.Eval(ctx, luaTokenBucket, []string{key}, now, rate, cap).Result()
	if err != nil {
		return Result{Allowed: true}, err // Fail-open
	}

	vals := res.([]interface{})
	return Result{
		Allowed:   vals[0].(int64) == 1,
		Remaining: int(vals[1].(int64)),
		Reset:     vals[2].(int64),
	}, nil
}
