// backend/cmd/seeder/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-chat/backend/config"
	"go-chat/backend/database"
	"go-chat/backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 連接到 .env 中設定的開發資料庫
	cfg := config.LoadConfig()
	database.ConnectMongoDB(cfg.MongoDBURI, cfg.DBName)

	usersCollection := database.GetCollection("users")
	chatroomsCollection := database.GetCollection("chatrooms")
	ctx := context.Background()

	log.Println("--- Starting Database Seeding on local DB ---")

	// 1. 為了安全，我們先刪除可能存在的舊測試使用者
	log.Println("Step 1: Cleaning up previous performance test user...")
	usersCollection.DeleteOne(ctx, bson.M{"email": "perf-test@example.com"})

	// 2. 建立主要被測試的使用者
	log.Println("Step 2: Creating main performance test user...")
	mainUserPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(mainUserPassword), bcrypt.DefaultCost)
	mainUser := models.User{
		ID:       primitive.NewObjectID(),
		Email:    "perf-test@example.com",
		Username: "perf_test_user",
		Password: string(hashedPassword),
	}
	_, err := usersCollection.InsertOne(ctx, mainUser)
	if err != nil {
		log.Fatalf("Failed to create main user: %v", err)
	}
	log.Printf("-> Main user created. Email: %s, Password: %s", mainUser.Email, mainUserPassword)

	// 3. 為主要使用者建立 100 個聊天室
	log.Println("Step 3: Creating 100 chat rooms for the main user...")
	// 為了確保聊天室的唯一性，我們只用 mainUser 自己跟自己建立聊天室
	// 在真實世界中不常見，但對於測試查詢效能是完美的
	var chatRoomsToInsert []interface{}
	for i := 0; i < 100; i++ {
		participants := []primitive.ObjectID{mainUser.ID}
		roomName := fmt.Sprintf("Performance Test Room %d", i+1)
		chatRoom := models.ChatRoom{
			Name:         roomName,
			CreatorID:    mainUser.ID,
			Participants: participants,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		chatRoomsToInsert = append(chatRoomsToInsert, chatRoom)
	}
	_, err = chatroomsCollection.InsertMany(ctx, chatRoomsToInsert)
	if err != nil {
		log.Fatalf("Failed to insert chat rooms: %v", err)
	}

	log.Println("--- Seeding Completed Successfully! ---")
}
