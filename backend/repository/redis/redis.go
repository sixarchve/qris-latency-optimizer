package redis

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func init() {
	ConnectRedis()
}

func ConnectRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := RedisClient.Ping(ctx).Err()
	if err != nil {
		fmt.Printf("Redis connection failed: %v (will continue without cache)\n", err)
	} else {
		fmt.Println("✓ Redis connected successfully")
	}
}

// Set - simpan data ke Redis dengan TTL
func Set(key string, value string, expiration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return RedisClient.Set(ctx, key, value, expiration).Err()
}

// Get - ambil data dari Redis
func Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return RedisClient.Get(ctx, key).Result()
}

// Delete - hapus data dari Redis
func Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return RedisClient.Del(ctx, key).Err()
}

// Exists - cek apakah key ada
func Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := RedisClient.Exists(ctx, key)
	return result.Val() > 0, result.Err()
}