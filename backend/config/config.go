package config

import (
	"github.com/joho/godotenv" // 引入這個庫來讀取 .env 檔案
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 結構體用於儲存應用程式的配置
type Config struct {
	MongoDBURI           string
	DBName               string
	Port                 string
	JWTSecret            string
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	RedisAddr            string
	RedisCacheExpiration time.Duration
	LoadtestMode         bool
}

// LoadConfig 載入配置，優先從環境變數讀取，其次從 .env 檔案讀取
func LoadConfig() *Config {
	// 嘗試載入 .env 檔案，如果不存在也不會報錯
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	// 預設為10分鐘的快取
	expMinutesStr := getEnv("REDIS_CACHE_EXPIRATION_MINUTES", "10")
	expMinutes, err := strconv.Atoi(expMinutesStr)
	if err != nil {
		log.Printf("Invalid cache expiration value, defaulting to 10 minutes: %v", err)
		expMinutes = 10
	}
	cacheExpiration := time.Duration(expMinutes) * time.Minute

	cfg := &Config{
		MongoDBURI:           getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:               getEnv("DB_NAME", "chat_app_db"),
		Port:                 getEnv("PORT", "8080"),
		JWTSecret:            getEnv("JWT_SECRET", "your_super_secret_jwt_key_please_change_this_in_production"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getEnv("GOOGLE_REDIRECT_URL", ""),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisCacheExpiration: cacheExpiration,
		LoadtestMode:         getEnvBool("LOADTEST_MODE", false),
	}
	return cfg
}

// getEnv 輔助函數，用於從環境變數獲取值，如果不存在則使用預設值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Printf("Invalid boolean env %s=%q, defaulting to %v", key, value, defaultValue)
		return defaultValue
	}
}
