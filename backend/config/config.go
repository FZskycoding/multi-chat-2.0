package config

import (
	"log"
	"os"

	"github.com/joho/godotenv" // 引入這個庫來讀取 .env 檔案
)

// Config 結構體用於儲存應用程式的配置
type Config struct {
	MongoDBURI string
	DBName     string
	Port       string
}

// LoadConfig 載入配置，優先從環境變數讀取，其次從 .env 檔案讀取
func LoadConfig() *Config {
	// 嘗試載入 .env 檔案，如果不存在也不會報錯
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	cfg := &Config{
		MongoDBURI: getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:     getEnv("DB_NAME", "chat_app_db"),
		Port:       getEnv("PORT", "8080"),
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
