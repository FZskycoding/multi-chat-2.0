package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatRoom 代表一個聊天室
type ChatRoom struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string              `bson:"name" json:"name"`
	Members   []primitive.ObjectID `bson:"members" json:"members"`
	CreatedAt time.Time           `bson:"createdAt" json:"createdAt"`
}

// Invitation 代表一個聊天室邀請
type Invitation struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	RoomID    primitive.ObjectID `bson:"roomId" json:"roomId"`
	InviterID primitive.ObjectID `bson:"inviterId" json:"inviterId"`
	InviteeID primitive.ObjectID `bson:"inviteeId" json:"inviteeId"`
	Status    string            `bson:"status" json:"status"` // "pending", "accepted", "rejected"
	CreatedAt time.Time         `bson:"createdAt" json:"createdAt"`
}
