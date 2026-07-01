package challenge

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore adapts *redis.Client to the Store interface. In cmd/engine,
// wire this to the existing `rdb` package variable — don't open a second
// connection pool:
//
//	challengeStore = &challenge.RedisStore{Client: rdb}
type RedisStore struct {
	Client *redis.Client
}

func (s *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.Client.Set(ctx, key, value, ttl).Err()
}

func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	val, err := s.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (s *RedisStore) Del(ctx context.Context, key string) error {
	return s.Client.Del(ctx, key).Err()
}