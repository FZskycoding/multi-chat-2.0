// backend/store/mongo_user_store.go
package store

import (
	"context"
	"errors"
	"go-chat/backend/database"
	"go-chat/backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoUserStore 是 UserStorer 介面的 MongoDB 實作
type MongoUserStore struct {
	collection *mongo.Collection
}

// NewMongoUserStore 是一個工廠函式，用於建立新的 MongoUserStore
func NewMongoUserStore() *MongoUserStore {
	return &MongoUserStore{
		collection: database.GetCollection("users"),
	}
}

// FindUserByEmail 根據 Email 查找使用者
func (s *MongoUserStore) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		// 為了讓 handler 能區分「找不到」和「資料庫真的出錯」，我們回傳 mongo.ErrNoDocuments
		return nil, err
	}
	return &user, nil
}

// FindUserByGoogleID 根據 GoogleID 查找使用者
func (s *MongoUserStore) FindUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	var user models.User
	err := s.collection.FindOne(ctx, bson.M{"googleId": googleID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CheckUserExists 檢查 Email 或 Username 是否已存在
func (s *MongoUserStore) CheckUserExists(ctx context.Context, email, username string) (bool, bool, error) {
	// 檢查 Email
	emailCount, err := s.collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, false, err
	}

	// 檢查 Username
	usernameCount, err := s.collection.CountDocuments(ctx, bson.M{"username": username})
	if err != nil {
		return false, false, err
	}

	return emailCount > 0, usernameCount > 0, nil
}

// CreateUser 創建新使用者
func (s *MongoUserStore) CreateUser(ctx context.Context, user models.User) (primitive.ObjectID, error) {
	result, err := s.collection.InsertOne(ctx, user)
	if err != nil {
		return primitive.NilObjectID, err
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("failed to convert InsertedID to ObjectID")
	}

	return oid, nil
}
