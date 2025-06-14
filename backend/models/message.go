package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message 代表一個聊天訊息
type Message struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SenderID     primitive.ObjectID `bson:"senderId" json:"senderId"`
	SenderUsername string             `bson:"senderUsername" json:"senderUsername"`
	RecipientID  primitive.ObjectID `bson:"recipientId,omitempty" json:"recipientId,omitempty"` // 新增接收者ID
	Content      string             `bson:"content" json:"content"`
	Timestamp    time.Time          `bson:"timestamp" json:"timestamp"`
	IsRead       bool               `bson:"isRead" json:"isRead"` // 新增已讀狀態
}
