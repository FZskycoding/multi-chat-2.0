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
	dbName = name

	messagesCollection := MongoClient.Database(dbName).Collection("messages")
	// 設定規則:自動清理超過30分鐘的訊息
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},        // value:1代表升序(由舊到新)
		Options: options.Index().SetExpireAfterSeconds(1800), // 設定 30 分鐘 (1800 秒) 後過期
	}

	// 套用規則
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

// 將新的聊天訊息插入到 MongoDB
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
	return result, nil
}

// GetChatHistory 獲取指定聊天室的歷史訊息
func GetChatHistory(roomID string) ([]models.Message, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"roomId": roomID}
	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}).SetLimit(50) // 獲取最近 50 條訊息

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Error finding chat history for room %s: %v", roomID, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		log.Printf("Error decoding chat history for room %s: %v", roomID, err)
		return nil, err
	}
	return messages, nil
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

// InsertChatRoom 將新的聊天室插入到 MongoDB
func InsertChatRoom(room models.ChatRoom) (*mongo.InsertOneResult, error) {
	collection := GetCollection("chatrooms") // 獲取 chatrooms 集合
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	room.CreatedAt = time.Now()
	room.UpdatedAt = time.Now()

	result, err := collection.InsertOne(ctx, room)
	if err != nil {
		log.Printf("Error inserting chatroom: %v", err)
		return nil, err
	}
	return result, nil
}

// FindChatRoomByID 根據 ID 查找聊天室
func FindChatRoomByID(roomID primitive.ObjectID) (*models.ChatRoom, error) {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var room models.ChatRoom
	err := collection.FindOne(ctx, bson.M{"_id": roomID}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // 未找到文件
		}
		log.Printf("Error finding chatroom by ID %s: %v", roomID.Hex(), err)
		return nil, err
	}
	return &room, nil
}

// FindChatRoomByParticipants 根據參與者 ID 查找聊天室 (用於兩人聊天室)
func FindChatRoomByParticipants(participantIDs []primitive.ObjectID) (*models.ChatRoom, error) {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 確保參與者 ID 列表已排序，以便查詢一致性
	// 注意：Go 的 primitive.ObjectID 不直接支援比較，需要轉換為字串或使用自定義比較
	// 這裡假設傳入的 participantIDs 已經是排序好的，或者我們在查詢時不依賴順序
	// 更穩健的做法是確保查詢條件能匹配任何順序的參與者列表
	filter := bson.M{
		"participants": bson.M{
			"$size": len(participantIDs), // 確保參與者數量一致
			"$all":  participantIDs,      // 確保包含所有指定的參與者
		},
	}

	var room models.ChatRoom
	err := collection.FindOne(ctx, filter).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // 未找到文件
		}
		log.Printf("Error finding chatroom by participants %v: %v", participantIDs, err)
		return nil, err
	}
	return &room, nil
}

// GetUserChatRooms 獲取使用者所參與的所有聊天室
func GetUserChatRooms(userID primitive.ObjectID) ([]models.ChatRoom, error) {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"participants": userID}                                     // 查找 participants 陣列中包含該 userID 的聊天室
	findOptions := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}) // 按最新更新時間排序

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Error finding chatrooms for user %s: %v", userID.Hex(), err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var chatRooms []models.ChatRoom
	if err = cursor.All(ctx, &chatRooms); err != nil {
		log.Printf("Error decoding chatrooms for user %s: %v", userID.Hex(), err)
		return nil, err
	}
	return chatRooms, nil
}
