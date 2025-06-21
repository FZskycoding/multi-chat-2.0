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

	// 初始化訊息集合
	messagesCollection := MongoClient.Database(dbName).Collection("messages")
	messageIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(1800), // 30分鐘後過期
	}
	_, err = messagesCollection.Indexes().CreateOne(ctx, messageIndexModel)

	// 初始化聊天室集合
	roomsCollection := MongoClient.Database(dbName).Collection("chatrooms")
	roomIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: 1}},
	}
	_, err = roomsCollection.Indexes().CreateOne(ctx, roomIndexModel)

	// 初始化邀請集合
	invitationsCollection := MongoClient.Database(dbName).Collection("invitations")
	invitationIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: 1}},
	}
	_, err = invitationsCollection.Indexes().CreateOne(ctx, invitationIndexModel)
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
	log.Printf("Message inserted with ID: %v", result.InsertedID)
	return result, nil
}

// 獲取指定數量的歷史訊息
func GetMessages(limit int64) ([]models.Message, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 設定查詢條件，value:-1代表降序(由新到舊)
	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(limit)

	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		log.Printf("Error finding messages: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// 直接解碼為 models.Message 切片
	var messages []models.Message
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

// CreateChatRoom 創建新的聊天室
func CreateChatRoom(room models.ChatRoom) (*mongo.InsertOneResult, error) {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	room.CreatedAt = time.Now()
	result, err := collection.InsertOne(ctx, room)
	if err != nil {
		log.Printf("Error creating chat room: %v", err)
		return nil, err
	}
	return result, nil
}

// GetChatRoom 獲取聊天室信息
func GetChatRoom(roomID primitive.ObjectID) (*models.ChatRoom, error) {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var room models.ChatRoom
	err := collection.FindOne(ctx, bson.M{"_id": roomID}).Decode(&room)
	if err != nil {
		log.Printf("Error finding chat room: %v", err)
		return nil, err
	}
	return &room, nil
}

// UpdateChatRoomName 更新聊天室名稱
func UpdateChatRoomName(roomID primitive.ObjectID, newName string) error {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"name": newName}}
	_, err := collection.UpdateOne(ctx, bson.M{"_id": roomID}, update)
	if err != nil {
		log.Printf("Error updating chat room name: %v", err)
		return err
	}
	return nil
}

// AddMemberToChatRoom 將新成員加入聊天室
func AddMemberToChatRoom(roomID, memberID primitive.ObjectID) error {
	collection := GetCollection("chatrooms")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$addToSet": bson.M{"members": memberID}}
	_, err := collection.UpdateOne(ctx, bson.M{"_id": roomID}, update)
	if err != nil {
		log.Printf("Error adding member to chat room: %v", err)
		return err
	}
	return nil
}

// CreateInvitation 創建聊天室邀請
func CreateInvitation(invitation models.Invitation) (*mongo.InsertOneResult, error) {
	collection := GetCollection("invitations")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	invitation.CreatedAt = time.Now()
	invitation.Status = "pending"
	result, err := collection.InsertOne(ctx, invitation)
	if err != nil {
		log.Printf("Error creating invitation: %v", err)
		return nil, err
	}
	return result, nil
}

// UpdateInvitationStatus 更新邀請狀態
func UpdateInvitationStatus(invitationID primitive.ObjectID, status string) error {
	collection := GetCollection("invitations")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"status": status}}
	_, err := collection.UpdateOne(ctx, bson.M{"_id": invitationID}, update)
	if err != nil {
		log.Printf("Error updating invitation status: %v", err)
		return err
	}
	return nil
}

// GetPendingInvitations 獲取用戶的待處理邀請
func GetPendingInvitations(inviteeID primitive.ObjectID) ([]models.Invitation, error) {
	collection := GetCollection("invitations")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"inviteeId": inviteeID,
		"status":    "pending",
	}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding pending invitations: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.Invitation
	if err = cursor.All(ctx, &invitations); err != nil {
		log.Printf("Error decoding invitations: %v", err)
		return nil, err
	}
	return invitations, nil
}

// GetRoomMessages 獲取聊天室的訊息歷史
func GetRoomMessages(roomID primitive.ObjectID) ([]models.Message, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"roomId": roomID}
	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Error finding room messages: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		log.Printf("Error decoding messages: %v", err)
		return nil, err
	}
	return messages, nil
}

// GetChatHistory 獲取兩個用戶之間的聊天記錄
func GetChatHistory(user1ID, user2ID primitive.ObjectID) ([]models.Message, error) {
	collection := GetCollection("messages")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 構建查詢條件：找出兩個用戶之間的私聊訊息
	filter := bson.M{
		"$or": []bson.M{
			{
				"senderId":    user1ID,
				"recipientId": user2ID,
				"type":        models.MessageTypeChat,
			},
			{
				"senderId":    user2ID,
				"recipientId": user1ID,
				"type":        models.MessageTypeChat,
			},
		},
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Error finding chat history between users %s and %s: %v",
			user1ID.Hex(), user2ID.Hex(), err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		log.Printf("Error decoding chat history: %v", err)
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
