package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InvalidateUserChatRoomsCache 刪除指定使用者的聊天室列表快取
func InvalidateUserChatRoomsCache(userID primitive.ObjectID) {
	cacheKey := fmt.Sprintf("user-chatrooms:%s", userID.Hex())
	ctx := context.Background()

	err := RedisClient.Del(ctx, cacheKey).Err()
	if err != nil {
		// 在快取失效操作中，即使失敗了，只記錄錯誤，不中斷主流程
		log.Printf("Failed to invalidate cache for user %s: %v", userID.Hex(), err)
	} else {
		log.Printf("Cache invalidated for user %s", userID.Hex())
	}
}

// InvalidateMultipleUserChatRoomsCache 刪除多個使用者的聊天室列表快取
func InvalidateMultipleUserChatRoomsCache(userIDs []primitive.ObjectID) {
	// 遍歷所有需要被更新的使用者 ID

	for _, userID := range userIDs {
		InvalidateUserChatRoomsCache(userID)
	}
}
