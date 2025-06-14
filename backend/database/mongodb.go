package database

import (
	"context"
	"log"
	"time"

	"go-chat/backend/models" // 引入 models 套件

	"go.mongodb.org/mongo-driver/bson" // 引入 bson 套件
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoClient *mongo.Client
var dbName string // 新增：儲存資料庫名稱

// ConnectMongoDB 建立並初始化 MongoDB 連線
func ConnectMongoDB(uri, name string) {
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
	dbName = name // 初始化 dbName

	// 為 messages 集合設定 TTL 索引
	messagesCollection := MongoClient.Database(dbName).Collection("messages")
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}}, // 在 timestamp 欄位上建立升序索引
		Options: options.Index().SetExpireAfterSeconds(1800), // 設定 30 分鐘 (1800 秒) 後過期
	}

	_, err = messagesCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Fatalf("Failed to create TTL index for messages collection: %v", err)
	}
	log.Println("TTL index created for messages collection (30 minutes).")
}

// GetCollection 獲取指定資料庫的集合
func GetCollection(collectionName string) *mongo.Collection {
	if MongoClient == nil {
		log.Fatal("MongoDB client is not initialized. Call ConnectMongoDB first.")
	}
	if dbName == "" { // 額外防護，確保 dbName 已初始化
		log.Fatal("Database name is not set. Call ConnectMongoDB with a valid dbName.")
	}
	return MongoClient.Database(dbName).Collection(collectionName) // 替換為你的 DB Name
}

// InsertMessage 將聊天訊息插入到 MongoDB
func InsertMessage(message models.Message) (*mongo.InsertOneResult, error) {
	collection := GetCollection("messages") // 獲取 messages 集合
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 確保新訊息的 IsRead 預設為 false
	message.IsRead = false

	result, err := collection.InsertOne(ctx, message)
	if err != nil {
		log.Printf("Error inserting message: %v", err)
		return nil, err
	}
	log.Printf("Message inserted with ID: %v", result.InsertedID)
	return result, nil
}

// GetMessages 獲取指定數量的歷史訊息
func GetMessages(limit int64) ([]interface{}, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(limit)
	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		log.Printf("Error finding messages: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []interface{}
	if err = cursor.All(ctx, &messages); err != nil {
		log.Printf("Error decoding messages: %v", err)
		return nil, err
	}
	return messages, nil
}

// GetUnreadMessages 獲取特定使用者所有未讀的訊息
func GetUnreadMessages(recipientID primitive.ObjectID) ([]models.Message, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"recipientId": recipientID, "isRead": false}
	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}) // 按時間戳升序排列

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Error finding unread messages for recipient %s: %v", recipientID.Hex(), err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		log.Printf("Error decoding unread messages: %v", err)
		return nil, err
	}
	return messages, nil
}

// MarkMessagesAsRead 將特定訊息標記為已讀
func MarkMessagesAsRead(messageIDs []primitive.ObjectID) (*mongo.UpdateResult, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": bson.M{"$in": messageIDs}}
	update := bson.M{"$set": bson.M{"isRead": true}}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Printf("Error marking messages as read: %v", err)
		return nil, err
	}
	log.Printf("Marked %d messages as read.", result.ModifiedCount)
	return result, nil
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
