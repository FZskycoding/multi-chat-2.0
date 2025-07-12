package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"go-chat/backend/database"
	"go-chat/backend/models"
	"go-chat/backend/utils" // 引入 utils 套件
	"go-chat/backend/websocket"

	"github.com/gorilla/mux"
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
		Name:         generateRoomName(usernames),
		CreatorID:    creatorID,
		Participants: participantObjectIDs,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
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

// LeaveChatRoom 處理使用者退出聊天室的請求
func LeaveChatRoom(w http.ResponseWriter, r *http.Request) {
	// 設置響應類型
	w.Header().Set("Content-Type", "application/json")

	// 從URL路徑參數獲取聊天室ID
	vars := mux.Vars(r)
	roomIDStr := vars["id"]
	if roomIDStr == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid room ID format", http.StatusBadRequest)
		return
	}

	// 從JWT token中獲取使用者ID
	userID, err := utils.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized: User ID not found in context", http.StatusUnauthorized)
		return
	}

	// 獲取當前聊天室
	room, err := database.FindChatRoomByID(roomID)
	if err != nil {
		log.Printf("Error getting chatroom: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if room == nil {
		http.Error(w, "Chat room not found", http.StatusNotFound)
		return
	}

	// 從參與者列表中移除使用者
	newParticipants := make([]primitive.ObjectID, 0)
	for _, participantID := range room.Participants {
		if participantID != userID {
			newParticipants = append(newParticipants, participantID)
		}
	}

	// 更新聊天室的參與者列表
	if len(newParticipants) > 0 {
		// 獲取剩餘參與者的用戶名
		usernames, err := getUsernames(newParticipants)
		if err != nil {
			log.Printf("Error getting usernames: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// 更新聊天室
		_, err = database.UpdateChatRoom(roomID, newParticipants, generateRoomName(usernames))
		if err != nil {
			log.Printf("Error updating chatroom: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// 如果沒有參與者了，刪除聊天室
		err = database.DeleteChatRoom(roomID)
		if err != nil {
			log.Printf("Error deleting empty chatroom: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// 獲取退出使用者的用戶名和創建系統消息
	exitingUser, err := database.GetUserByID(userID)
	if err != nil {
		log.Printf("Error getting exiting user: %v", err)
		// 如果無法獲取用戶名，使用備用訊息
		exitingUser = &models.User{Username: "某人"}
	}

	// 創建並發送系統消息
	systemMessage := models.Message{
		Type:           models.MessageTypeSystem,
		SenderID:       userID,
		SenderUsername: "系統",
		RoomID:         roomID.Hex(),
		RoomName:       room.Name,
		Content:        exitingUser.Username + " 已離開聊天室",
		Timestamp:      time.Now(),
		IsRead:         true,
	}

	// 通過 WebSocket 廣播系統消息
	websocket.BroadcastMessage(systemMessage)

	// 存儲系統消息
	_, err = database.InsertMessage(systemMessage)
	if err != nil {
		log.Printf("Error inserting system message: %v", err)
		// 繼續執行，不中斷流程
	}

	// 返回成功狀態
	response := map[string]bool{"success": true}
	json.NewEncoder(w).Encode(response)
}

// UpdateChatRoom 處理更新聊天室的請求
func UpdateChatRoom(w http.ResponseWriter, r *http.Request) {
	// 設置響應類型
	w.Header().Set("Content-Type", "application/json")

	var req UpdateChatRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 從URL路徑參數獲取聊天室ID
	vars := mux.Vars(r)
	roomIDStr := vars["id"]
	if roomIDStr == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid room ID format", http.StatusBadRequest)
		return
	}

	// 檢查聊天室是否存在
	existingRoom, err := database.FindChatRoomByID(roomID)
	if err != nil {
		log.Printf("Error getting chatroom: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingRoom == nil {
		http.Error(w, "Chat room not found", http.StatusNotFound)
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

	json.NewEncoder(w).Encode(updatedRoom)
}
