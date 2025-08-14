// backend/store/user_store.go
package store

import (
	"context"
	"go-chat/backend/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserStorer 定義了所有對使用者資料的必要操作
// 這是我們 handler 將要依賴的抽象層
type UserStorer interface {
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	FindUserByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	CheckUserExists(ctx context.Context, email, username string) (bool, bool, error)
	CreateUser(ctx context.Context, user models.User) (primitive.ObjectID, error)
}
