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

// errorResponse 結構體用於返回 JSON 格式的錯誤訊息
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

// 註：`Password` 欄位在儲存到資料庫前會被哈希，`json:"-"` 表示在 JSON 序列化時忽略此欄位，避免將密碼暴露出去。
// `unique:"true"` 是一個示意，實際的唯一索引會在 MongoDB 操作時建立。
