package database

import (
    "context"
    "log"
    "time"

    "go-chat/backend/models"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoClient *mongo.Client
var dbName string // 儲存資料庫名稱

// GetUserByID 根據用戶ID獲取用戶信息
func GetUserByID(userID primitive.ObjectID) (*models.User, error) {
    collection := GetCollection("users")
    var user models.User
    err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// UpdateChatRoom 更新聊天室信息
func UpdateChatRoom(roomID primitive.ObjectID, participants []primitive.ObjectID, newName string) (*models.ChatRoom, error) {
    collection := GetCollection("chatrooms")
    update := bson.M{
        "$set": bson.M{
            "name":        newName,
            "participants": participants,
            "updatedAt":   time.Now(),
        },
    }

    var updatedRoom models.ChatRoom
    err := collection.FindOneAndUpdate(
        context.Background(),
        bson.M{"_id": roomID},
        update,
        options.FindOneAndUpdate().SetReturnDocument(options.After),
    ).Decode(&updatedRoom)

    if err != nil {
        return nil, err
    }

    return &updatedRoom, nil
}

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
    return MongoClient.Database(dbName).Collection(collectionName)
}

// InsertMessage 將新的聊天訊息插入到 MongoDB
func InsertMessage(message models.Message) (*mongo.InsertOneResult, error) {
    collection := GetCollection("messages")
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
    findOptions := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}).SetLimit(50)

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
    collection := GetCollection("chatrooms")
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
            return nil, nil
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

    filter := bson.M{
        "participants": bson.M{
            "$size": len(participantIDs),
            "$all":  participantIDs,
        },
    }

    var room models.ChatRoom
    err := collection.FindOne(ctx, filter).Decode(&room)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
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

    filter := bson.M{"participants": userID}
    findOptions := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})

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
