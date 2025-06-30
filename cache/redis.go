package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr, password string) *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		}),
	}
}
func (r *RedisClient) CacheMessage(ctx context.Context, roomID string, message []byte, expiry time.Duration) error {
	return r.client.Set(ctx, "chat:"+roomID+":"+time.Now().String(), message, expiry).Err()
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}
func WaitRedis(redisClient *RedisClient) {
	for i := 0; i < 10; i++ {
		err := redisClient.Ping(context.Background())
		if err == nil {
			return
		}
		log.Printf("wait redis")
		time.Sleep(1 * time.Second)
	}
	log.Fatal("redis not connected")
}
