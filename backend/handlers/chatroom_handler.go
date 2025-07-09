package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"strings"

	"go-chat/backend/database"
	"go-chat/backend/models"
	"go-chat/backend/utils" // 引入 utils 套件

	
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateChatRoomRequest 定義創建聊天室的請求體
type CreateChatRoomRequest struct {
    ParticipantIDs []string `json:"participantIds"` // 參與者的使用者 ID 字串列表
}

// UpdateChatRoomRequest 定義更新聊天室的請求體
type UpdateChatRoomRequest struct {
    ParticipantIDs []string `json:"participantIds"` // 新的參與者列表
}

// 根據用戶ID獲取用戶名稱列表
func getUsernames(userIDs []primitive.ObjectID) ([]string, error) {
    usernames := make([]string, 0, len(userIDs))
    for _, id := range userIDs {
        user, err := database.GetUserByID(id)
        if err != nil {
            return nil, err
        }
        usernames = append(usernames, user.Username)
    }
    return usernames, nil
}

// generateRoomName 根據參與者生成聊天室名稱
func generateRoomName(participantUsernames []string) string {
    return strings.Join(participantUsernames, "、") + "的聊天室"
}

// CreateChatRoom 處理創建聊天室的請求
func CreateChatRoom(w http.ResponseWriter, r *http.Request) {
    var req CreateChatRoomRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // 驗證請求參數
    if len(req.ParticipantIDs) < 2 {
        http.Error(w, "At least two participants are required", http.StatusBadRequest)
        return
    }

	// 從 JWT token 中獲取創建者 ID
	creatorID, err := utils.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized: Creator ID not found in context", http.StatusUnauthorized)
		return
	}

	// 將參與者 ID 字串轉換為 primitive.ObjectID
	var participantObjectIDs []primitive.ObjectID
	for _, idStr := range req.ParticipantIDs {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid participant ID format", http.StatusBadRequest)
			return
		}
		participantObjectIDs = append(participantObjectIDs, objID)
	}

	// 確保創建者也在參與者列表中
	foundCreator := false
	for _, pID := range participantObjectIDs {
		if pID == creatorID {
			foundCreator = true
			break
		}
	}
	if !foundCreator {
		participantObjectIDs = append(participantObjectIDs, creatorID)
	}

	// 檢查是否已存在相同參與者的聊天室 (針對兩人聊天室的特殊處理)
	if len(participantObjectIDs) == 2 {
		// 為了確保順序一致性，對參與者 ID 進行排序
		sortedParticipantIDs := make([]primitive.ObjectID, len(participantObjectIDs))
		copy(sortedParticipantIDs, participantObjectIDs)
		// 這裡需要一個穩健的排序方法，例如轉換為字串後排序
		// 簡單起見，假設 primitive.ObjectID 可以直接比較或使用其 Hex() 字串比較
		// 實際應用中，可能需要自定義排序邏輯
		// For now, we'll rely on the database's $all operator which doesn't care about order.
		// However, for FindChatRoomByParticipants to work consistently, the input array should be sorted.
		// Let's sort by Hex string for consistency.
		utils.SortObjectIDs(sortedParticipantIDs) // 假設 utils.SortObjectIDs 存在並能正確排序

		existingRoom, err := database.FindChatRoomByParticipants(sortedParticipantIDs)
		if err != nil {
			log.Printf("Error checking for existing chatroom: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if existingRoom != nil {
			// 如果已存在，則直接返回該聊天室資訊
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(existingRoom)
			return
		}
	}

// 獲取所有參與者的用戶名
usernames, err := getUsernames(participantObjectIDs)
if err != nil {
    log.Printf("Error getting usernames: %v", err)
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    return
}

// 創建新的聊天室物件，使用生成的名稱
newChatRoom := models.ChatRoom{
Name:        generateRoomName(usernames),
		CreatorID:   creatorID,
		Participants: participantObjectIDs,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 插入到資料庫
	result, err := database.InsertChatRoom(newChatRoom)
	if err != nil {
		log.Printf("Error inserting new chatroom: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newChatRoom.ID = result.InsertedID.(primitive.ObjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newChatRoom)
}

// GetUserChatRooms 處理獲取使用者所有聊天室的請求
func GetUserChatRooms(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized: User ID not found in context", http.StatusUnauthorized)
		return
	}

	chatRooms, err := database.GetUserChatRooms(userID)
	if err != nil {
		log.Printf("Error getting chatrooms for user %s: %v", userID.Hex(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(chatRooms)
}

// UpdateChatRoom 處理更新聊天室的請求
func UpdateChatRoom(w http.ResponseWriter, r *http.Request) {
    var req UpdateChatRoomRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // 從URL獲取聊天室ID
    roomIDStr := r.URL.Query().Get("id")
    if roomIDStr == "" {
        http.Error(w, "Room ID is required", http.StatusBadRequest)
        return
    }

    roomID, err := primitive.ObjectIDFromHex(roomIDStr)
    if err != nil {
        http.Error(w, "Invalid room ID format", http.StatusBadRequest)
        return
    }

    // 將參與者ID字串轉換為ObjectID
    var participantObjectIDs []primitive.ObjectID
    for _, idStr := range req.ParticipantIDs {
        objID, err := primitive.ObjectIDFromHex(idStr)
        if err != nil {
            http.Error(w, "Invalid participant ID format", http.StatusBadRequest)
            return
        }
        participantObjectIDs = append(participantObjectIDs, objID)
    }

    // 獲取所有參與者的用戶名
    usernames, err := getUsernames(participantObjectIDs)
    if err != nil {
        log.Printf("Error getting usernames: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // 更新聊天室
    updatedRoom, err := database.UpdateChatRoom(roomID, participantObjectIDs, generateRoomName(usernames))
    if err != nil {
        log.Printf("Error updating chatroom: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(updatedRoom)
}
