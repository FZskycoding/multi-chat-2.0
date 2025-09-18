// backend/database/redis.go
package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// RedisClient 是一個全域變數，用於在整個應用程式中存取 Redis 連線
var RedisClient *redis.Client

// ConnectRedis 建立並初始化 Redis 連線
func ConnectRedis(addr string) {
	// 建立一個新的 Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr, // Redis 伺服器位址，例如 "localhost:6379"
		Password: "",   // 如果你的 Redis 沒有密碼，就留空
		DB:       0,    // 使用預設的資料庫 0
	})

	// 使用 Ping 來檢查連線是否成功
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis successfully!")
	RedisClient = rdb // 將建立好的連線存到全域變數中
}
