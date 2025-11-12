package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	Ctx    = context.Background()
)

// InitRedis initializes the Redis client
func InitRedis() {
	host := os.Getenv("REDIS_HOST")
	// if host == "" {
	// 	host = "localhost"
	// }
	port := os.Getenv("REDIS_PORT")
	// if port == "" {
	// 	port = "6379"
	// }

	addr := fmt.Sprintf("%s:%s", host, port)
	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // isi jika pakai password
		DB:       0,
	})

	if err := Client.Ping(Ctx).Err(); err != nil {
		panic(fmt.Sprintf("❌ Failed to connect to Redis: %v", err))
	}

	fmt.Println("✅ Redis connected at", addr)
}
