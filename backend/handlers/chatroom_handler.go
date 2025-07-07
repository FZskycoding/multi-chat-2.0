package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go-chat/backend/database"
	"go-chat/backend/models"
	"go-chat/backend/utils" // 引入 utils 套件

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateChatRoomRequest 定義創建聊天室的請求體
type CreateChatRoomRequest struct {
	Name          string   `json:"name"`
	ParticipantIDs []string `json:"participantIds"` // 參與者的使用者 ID 字串列表
}

// CreateChatRoom 處理創建聊天室的請求
func CreateChatRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateChatRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 驗證請求參數
	if req.Name == "" || len(req.ParticipantIDs) < 2 {
		http.Error(w, "Room name and at least two participants are required", http.StatusBadRequest)
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

	// 創建新的聊天室物件
	newChatRoom := models.ChatRoom{
		Name:        req.Name,
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
