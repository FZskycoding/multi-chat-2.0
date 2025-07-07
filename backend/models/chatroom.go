package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatRoom 代表一個聊天室的元資料
type ChatRoom struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string               `bson:"name" json:"name"`
	CreatorID   primitive.ObjectID   `bson:"creatorId" json:"creatorId"`
	Participants []primitive.ObjectID `bson:"participants" json:"participants"` // 參與者的使用者 ID 列表
	CreatedAt   time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time            `bson:"updatedAt" json:"updatedAt"`
}
