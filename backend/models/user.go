// models/user.go
package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterRequest 結構體用於處理註冊請求
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 結構體用於登入成功後的響應
type LoginResponse struct {
	Message  string `json:"message"`
	ID       string `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// RegisterResponse 結構體用於註冊成功後的響應
type RegisterResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

// ErrorResponse 結構體用於返回 JSON 格式的錯誤訊息
type ErrorResponse struct {
	Message string `json:"message"`
}

// User 結構體定義了使用者資料的欄位
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"` // MongoDB 的唯一 ID
	Email    string             `bson:"email" json:"email" unique:"true"`  // 使用者 Email
	Username string             `bson:"username" json:"username"`          // 使用者名稱
	Password string             `bson:"password" json:"-"`                 // 儲存哈希後的密碼，JSON 輸出時忽略
}

// PublicUser 結構體用於返回給前端，不包含敏感資訊
type PublicUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	// 可以添加其他非敏感字段
}
