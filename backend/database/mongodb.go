package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoClient *mongo.Client

// ConnectMongoDB 建立並初始化 MongoDB 連線
func ConnectMongoDB(uri, dbName string) {
	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping the primary to verify connection
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Println("Connected to MongoDB successfully!")
	MongoClient = client
}

// GetCollection 獲取指定資料庫的集合
func GetCollection(collectionName string) *mongo.Collection {
	if MongoClient == nil {
		log.Fatal("MongoDB client is not initialized. Call ConnectMongoDB first.")
	}
	// cfg.DBName 應該從 main.go 傳入或從 config.LoadConfig() 獲取
	// 這裡先寫死 dbName，稍後在 main.go 中會完善
	return MongoClient.Database("chat_app_db").Collection(collectionName) // 替換為你的 DB Name
}

// DisconnectMongoDB 關閉 MongoDB 連線
func DisconnectMongoDB() {
	if MongoClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := MongoClient.Disconnect(ctx); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	} else {
		log.Println("Disconnected from MongoDB.")
	}
}
