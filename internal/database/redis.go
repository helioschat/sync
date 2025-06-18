package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(url, password string, db int) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     parseRedisURL(url),
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: rdb,
		ctx:    ctx,
	}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Set(key string, value interface{}, expiration int64) error {
	if expiration > 0 {
		return r.client.Set(r.ctx, key, value, time.Duration(expiration)*time.Second).Err()
	}
	return r.client.Set(r.ctx, key, value, 0).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

func (r *RedisClient) Del(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisClient) HSet(key string, field string, value interface{}) error {
	return r.client.HSet(r.ctx, key, field, value).Err()
}

func (r *RedisClient) HGet(key string, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.client.HGetAll(r.ctx, key).Result()
}

func (r *RedisClient) HDel(key string, fields ...string) error {
	return r.client.HDel(r.ctx, key, fields...).Err()
}

func (r *RedisClient) Keys(pattern string) ([]string, error) {
	return r.client.Keys(r.ctx, pattern).Result()
}

func (r *RedisClient) ZAdd(key string, score float64, member interface{}) error {
	return r.client.ZAdd(r.ctx, key, &redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

func (r *RedisClient) ZRangeByScore(key string, min, max string) ([]string, error) {
	return r.client.ZRangeByScore(r.ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

func (r *RedisClient) ZRem(key string, members ...interface{}) error {
	return r.client.ZRem(r.ctx, key, members...).Err()
}

func parseRedisURL(url string) string {
	// Simple URL parsing for redis://localhost:6379 format
	if url == "" {
		return "localhost:6379"
	}

	// Remove redis:// prefix if present
	if len(url) > 8 && url[:8] == "redis://" {
		return url[8:]
	}

	return url
}
