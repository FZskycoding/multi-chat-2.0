package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageType 定義訊息類型
type MessageType string

const (
	MessageTypeChat   MessageType = "chat"   // 一般聊天訊息
	MessageTypeSystem MessageType = "system" // 系統通知，如成員加入通知
)

// Message 代表一個聊天訊息
type Message struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Type          MessageType       `bson:"type" json:"type"`                                 // 訊息類型
	RoomID        primitive.ObjectID `bson:"roomId" json:"roomId"`                           // 聊天室ID
	SenderID      primitive.ObjectID `bson:"senderId" json:"senderId"`                       // 發送者ID
	SenderUsername string            `bson:"senderUsername" json:"senderUsername"`            // 發送者名稱
	RecipientID   primitive.ObjectID `bson:"recipientId,omitempty" json:"recipientId,omitempty"` // 接收者ID（用於私聊）
	Content       string            `bson:"content" json:"content"`                          // 訊息內容
	Timestamp     time.Time         `bson:"timestamp" json:"timestamp"`                      // 時間戳
	IsRead        bool              `bson:"isRead" json:"isRead"`                            // 已讀狀態
}
