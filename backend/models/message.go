package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageType 定義消息類型
type MessageType string

const (
	MessageTypeNormal MessageType = "normal" // 普通消息
	MessageTypeSystem MessageType = "system" // 系統消息
	MessageTypeUpdate MessageType = "room_state_update" // 更新消息(需隱藏)
)

// Message 代表一個聊天訊息
type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Type           MessageType        `bson:"type" json:"type"` // 消息類型
	SenderID       primitive.ObjectID `bson:"senderId" json:"senderId"`
	SenderUsername string             `bson:"senderUsername" json:"senderUsername"`
	RoomID         string             `bson:"roomId" json:"roomId"`     // 聊天室ID
	RoomName       string             `bson:"roomName" json:"roomName"` // 聊天室名稱
	Content        string             `bson:"content" json:"content"`
	Timestamp      time.Time          `bson:"timestamp" json:"timestamp"`
	IsRead         bool               `bson:"isRead" json:"isRead"` // 新增已讀狀態
}
