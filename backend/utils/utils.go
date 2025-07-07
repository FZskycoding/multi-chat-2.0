package utils

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserIDKey 是儲存在 context 中的使用者 ID 的鍵
type contextKey string

const UserIDKey contextKey = "userID"

// GetUserIDFromContext 從 context 中提取使用者 ID
func GetUserIDFromContext(ctx context.Context) (primitive.ObjectID, error) {
	userID, ok := ctx.Value(UserIDKey).(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("user ID not found in context")
	}
	return userID, nil
}

// GetUserIDFromToken 從 JWT token 中提取使用者 ID
func GetUserIDFromToken(tokenString string, jwtSecret string) (primitive.ObjectID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return primitive.NilObjectID, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return primitive.NilObjectID, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["userId"].(string)
	if !ok {
		return primitive.NilObjectID, errors.New("user ID not found in token claims")
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return primitive.NilObjectID, errors.New("invalid user ID format in token")
	}

	return userID, nil
}

// SortObjectIDs 對 primitive.ObjectID 切片進行排序 (按 Hex 字串)
func SortObjectIDs(ids []primitive.ObjectID) {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].Hex() < ids[j].Hex()
	})
}

// GenerateJWT 為用戶生成 JWT Token
func GenerateJWT(userID primitive.ObjectID, username string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"userId":   userID.Hex(), // 將 ObjectID 轉換為 Hex 字串儲存
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token 24 小時後過期
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用您配置的 JWT_SECRET 簽名
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", errors.New("failed to sign token")
	}
	return tokenString, nil
}
