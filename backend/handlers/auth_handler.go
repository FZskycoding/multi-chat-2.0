package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go-chat/backend/database"
	"go-chat/backend/models"
	"go-chat/backend/utils" 
	"go-chat/backend/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt" // 用於密碼哈希
)

// sendJSONError 統一發送 JSON 格式錯誤響應
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	var errorResponse models.ErrorResponse
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse.Message = message
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

// RegisterUser 處理使用者註冊請求
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var registerReq models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		log.Printf("JSON decode error: %v", err)
		sendJSONError(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 基本的輸入驗證
	if registerReq.Email == "" || registerReq.Username == "" || registerReq.Password == "" {
		sendJSONError(w, "Email, username, and password are required", http.StatusBadRequest)
		return
	}

	usersCollection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 檢查 Email 是否已存在
	count, err := usersCollection.CountDocuments(ctx, bson.M{"email": registerReq.Email})
	if err != nil {
		log.Printf("Error checking existing email: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		sendJSONError(w, "Email already registered", http.StatusConflict)
		return
	}

	// 檢查 Username 是否已存在
	count, err = usersCollection.CountDocuments(ctx, bson.M{"username": registerReq.Username})
	if err != nil {
		log.Printf("Error checking existing username: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		sendJSONError(w, "Username already taken", http.StatusConflict)
		return
	}

	// 哈希密碼
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerReq.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 創建新使用者
	user := models.User{
		Email:    registerReq.Email,
		Username: registerReq.Username,
		Password: string(hashedPassword),
	}

	// 插入新使用者到資料庫
	result, err := usersCollection.InsertOne(ctx, user)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		sendJSONError(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	log.Printf("User registered successfully: %v", result.InsertedID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	json.NewEncoder(w).Encode(models.RegisterResponse{
		Message: "User registered successfully",
		ID:      result.InsertedID.(primitive.ObjectID).Hex(),
	})
}

// LoginUser 處理使用者登入請求
func LoginUser(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		log.Printf("JSON decode error: %v", err)
		sendJSONError(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 基本的輸入驗證
	if credentials.Email == "" || credentials.Password == "" {
		sendJSONError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	usersCollection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 透過 Email 尋找使用者
	var user models.User
	err := usersCollection.FindOne(ctx, bson.M{"email": credentials.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			sendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			log.Printf("Error finding user by email: %v", err)
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// 比較哈希後的密碼
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		sendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	cfg := config.LoadConfig() // 獲取 JWT Secret
	token, err := utils.GenerateJWT(user.ID, user.Username, cfg.JWTSecret) // 調用 utils 中的 GenerateJWT 函數
	if err != nil {
		log.Printf("Error generating JWT token for user %s: %v", user.ID.Hex(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 登入成功
	log.Printf("User logged in successfully: %s", user.Email)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK
	// 回傳使用者 ID 和 Username 給前端
	json.NewEncoder(w).Encode(models.LoginResponse{
		Message:  "Login successful",
		ID:       user.ID.Hex(), // 將 ObjectID 轉換為 Hex 字串
		Username: user.Username,
		Token:    token,
	})
}

// GetAllUsers 處理獲取所有使用者列表的請求
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	usersCollection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 查找所有使用者
	cursor, err := usersCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Error finding all users: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		log.Printf("Error decoding users: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 創建一個只包含公開資訊的 PublicUser 列表
	var publicUsers []models.PublicUser
	for _, user := range users {
		publicUsers = append(publicUsers, models.PublicUser{
			ID:       user.ID.Hex(),
			Username: user.Username,
			Email:    user.Email, // 考慮是否要暴露 Email，如果不需要可以移除
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(publicUsers); err != nil {
		log.Printf("Error encoding public users: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
	}
}
