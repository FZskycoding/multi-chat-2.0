package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort" // 引入 sort 套件用於排序使用者名稱
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
// 此結構體在當前命名邏輯下可能不會直接用於更新名稱，但保留以防其他用途
type UpdateChatRoomRequest struct {
	ParticipantIDs []string `json:"participantIds"` // 新的參與者列表
}

// AddParticipantsRequest 定義邀請參與者的請求體
type AddParticipantsRequest struct {
	NewParticipantIDs []string `json:"newParticipantIds"` // 新增參與者的使用者 ID 字串列表
}

// getUsernames 根據用戶ID獲取用戶名稱列表
func getUsernames(userIDs []primitive.ObjectID) ([]string, error) {
	if len(userIDs) == 0 {
		return []string{}, nil
	}

	users, err := database.GetUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	usernames := make([]string, 0, len(users))
	for _, user := range users {
		usernames = append(usernames, user.Username)
	}

	// 選擇性：對用戶名進行排序，確保聊天室名稱的順序一致性
	sort.Strings(usernames)
	return usernames, nil
}

// generateRoomName 根據參與者生成聊天室名稱
func generateRoomName(participantUsernames []string) string {
	if len(participantUsernames) == 0 {
		return "空聊天室" // 或者其他預設名稱
	}
	return strings.Join(participantUsernames, "、") + " 的聊天室"
}

// CreateChatRoom 處理創建聊天室的請求
func CreateChatRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateChatRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.ParticipantIDs) < 1 {
		http.Error(w, "At least one participant is required", http.StatusBadRequest)
		return
	}

	creatorID, err := utils.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized: Creator ID not found in context", http.StatusUnauthorized)
		return
	}

	var participantObjectIDs []primitive.ObjectID
	for _, idStr := range req.ParticipantIDs {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid participant ID format", http.StatusBadRequest)
			return
		}
		participantObjectIDs = append(participantObjectIDs, objID)
	}

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

	utils.SortObjectIDs(participantObjectIDs)

	existingRoom, err := database.FindChatRoomByParticipants(participantObjectIDs)
	if err != nil {
		log.Printf("Error checking for existing chatroom: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingRoom != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existingRoom)
		return
	}

	usernames, err := getUsernames(participantObjectIDs)
	if err != nil {
		log.Printf("Error getting usernames: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newChatRoom := models.ChatRoom{
		Name:         generateRoomName(usernames),
		CreatorID:    creatorID,
		Participants: participantObjectIDs,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	result, err := database.InsertChatRoom(newChatRoom)
	if err != nil {
		log.Printf("Error inserting new chatroom: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newChatRoom.ID = result.InsertedID.(primitive.ObjectID)

	// 【關鍵修正】在這裡補上 WebSocket 廣播
	// 建立一個 room_state_update 類型的訊息
	roomUpdateMessage := models.Message{
		Type:           models.MessageTypeUpdate,
		RoomID:         newChatRoom.ID.Hex(),
		RoomName:       newChatRoom.Name,
		SenderID:       primitive.NilObjectID, // 系統訊息 SenderID 為空
		SenderUsername: "系統更新",
		Content:        "聊天室已建立或更新。", // 內容不重要，前端主要依賴類型判斷
		Timestamp:      time.Now(),
		IsRead:         true,
	}

	// 為了持久化，可以選擇性地將這條更新訊息存入資料庫
	// (如果不需要儲存，可以省略這段)
	updateMsgResult, err := database.InsertMessage(roomUpdateMessage)
	if err != nil {
		log.Printf("Error inserting system update message on create: %v", err)
	} else {
		roomUpdateMessage.ID = updateMsgResult.InsertedID.(primitive.ObjectID)
	}

	// 透過 WebSocket 廣播給所有相關人員
	websocket.GlobalHub.Broadcast <- roomUpdateMessage
	log.Printf("Broadcasted room_state_update for new room %s", newChatRoom.ID.Hex())
	// 修正結束

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
	w.Header().Set("Content-Type", "application/json")

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

	userID, err := utils.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized: User ID not found in context", http.StatusUnauthorized)
		return
	}

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

	// 創建用於系統訊息的聊天室名稱變數
	var finalRoomNameForMessage string = room.Name // 預設為舊名稱

	// 更新聊天室的參與者列表
	if len(newParticipants) > 0 {
		// 獲取剩餘參與者的用戶名
		usernames, err := getUsernames(newParticipants)
		if err != nil {
			log.Printf("Error getting usernames for updated room name: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		newRoomName := generateRoomName(usernames)
		finalRoomNameForMessage = newRoomName // 如果更新，使用新名稱

		// 更新聊天室
		_, err = database.UpdateChatRoom(roomID, newParticipants, newRoomName)
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
		// 如果聊天室被刪除，finalRoomNameForMessage 仍使用舊名稱表示離開了哪個房間
	}

	// 獲取退出使用者的用戶名和創建系統消息
	exitingUser, err := database.GetUserByID(userID)
	if err != nil {
		log.Printf("Error getting exiting user: %v", err)
		exitingUser = &models.User{Username: "某人"} // 如果無法獲取用戶名，使用備用訊息
	}

	// 創建並發送系統消息
	systemMessage := models.Message{
		Type:           models.MessageTypeSystem,
		SenderID:       userID,
		SenderUsername: "系統訊息",
		RoomID:         roomID.Hex(),
		RoomName:       finalRoomNameForMessage, // 使用更新或舊的聊天室名稱
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

	roomUpdateMessage := models.Message{
		Type:           models.MessageTypeUpdate, // 使用更新消息類型
		RoomID:         roomID.Hex(),
		RoomName:       finalRoomNameForMessage, // 使用更新或舊的聊天室名稱
		SenderID:       primitive.NilObjectID,   // 系統訊息的 SenderID 設置為空
		SenderUsername: "系統更新",                 // 區分於邀請通知
		Content:        "聊天室成員或名稱已更新。",         // 可以是任意通知內容，客戶端主要依賴類型判斷
		Timestamp:      time.Now(),
		IsRead:         true, // 通常不需要已讀狀態
	}
	// 存儲系統消息
	roomUpdateMessageResult, err := database.InsertMessage(roomUpdateMessage)
	if err != nil {
		log.Printf("Error inserting system update message on leave: %v", err)
	} else {
		// 如果插入成功，更新 roomUpdateMessage 的 ID
		roomUpdateMessage.ID = roomUpdateMessageResult.InsertedID.(primitive.ObjectID)
	}
	websocket.GlobalHub.Broadcast <- roomUpdateMessage // 廣播隱藏的更新消息

	// 返回成功狀態
	response := map[string]bool{"success": true}
	json.NewEncoder(w).Encode(response)
}

// AddParticipants 處理將新使用者加入聊天室的請求
// 這個 API 端點會是 PUT /chatrooms/{id}/participants
func AddParticipants(w http.ResponseWriter, r *http.Request) {
	// 從 context 中獲取邀請者的 userID
	userID, err := utils.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	roomIDStr := vars["id"] // 從 URL 路徑中獲取 room ID
	if roomIDStr == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	roomID, err := primitive.ObjectIDFromHex(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid room ID format", http.StatusBadRequest)
		return
	}

	// 解析請求體
	var req AddParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 查找現有聊天室
	existingRoom, err := database.FindChatRoomByID(roomID)
	if err != nil {
		log.Printf("Error finding chatroom %s: %v", roomID.Hex(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existingRoom == nil {
		http.Error(w, "Chat room not found", http.StatusNotFound)
		return
	}

	// 將新的參與者ID字串轉換為ObjectID
	var newParticipantObjectIDs []primitive.ObjectID
	for _, idStr := range req.NewParticipantIDs {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid new participant ID format", http.StatusBadRequest)
			return
		}
		newParticipantObjectIDs = append(newParticipantObjectIDs, objID)
	}

	// 過濾掉已經在聊天室中的參與者
	var actualNewParticipants []primitive.ObjectID
	existingParticipantMap := make(map[primitive.ObjectID]bool)
	for _, p := range existingRoom.Participants {
		existingParticipantMap[p] = true
	}

	for _, newID := range newParticipantObjectIDs {
		if !existingParticipantMap[newID] {
			actualNewParticipants = append(actualNewParticipants, newID)
		}
	}

	if len(actualNewParticipants) == 0 {
		// 如果沒有新的參與者需要添加，則直接返回成功
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "No new participants to add"})
		return
	}

	// 合併現有參與者和實際新加入的參與者
	updatedParticipants := append(existingRoom.Participants, actualNewParticipants...)
	utils.SortObjectIDs(updatedParticipants) // 保持參與者列表有序

	// 生成新的聊天室名稱
	allParticipantsUsernames, err := getUsernames(updatedParticipants)
	if err != nil {
		log.Printf("Error getting usernames for new room name: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	newRoomName := generateRoomName(allParticipantsUsernames)

	// 更新聊天室到資料庫（更新參與者列表、名稱和 updatedAt）
	updatedRoom, err := database.UpdateChatRoom(roomID, updatedParticipants, newRoomName) // 傳遞新的名稱
	if err != nil {
		log.Printf("Error updating chatroom with new participants: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// --- 處理系統訊息 ---
	// 獲取邀請者和新參與者的用戶名
	var allUserIDsForMessage []primitive.ObjectID
	allUserIDsForMessage = append(allUserIDsForMessage, userID)                   // 邀請者 ID
	allUserIDsForMessage = append(allUserIDsForMessage, actualNewParticipants...) // 被邀請者 ID

	users, err := database.GetUsersByIDs(allUserIDsForMessage)
	if err != nil {
		log.Printf("Error getting users for system message: %v", err)
		// 即使獲取用戶名失敗，也應該返回成功，只是系統訊息可能顯示為「未知使用者」
	}

	inviterUsername := "未知使用者"
	newParticipantUsernames := []string{}
	for _, user := range users {
		if user.ID == userID {
			inviterUsername = user.Username
		} else {
			// 確保只包含實際被添加的用戶名
			for _, newP := range actualNewParticipants {
				if user.ID == newP {
					newParticipantUsernames = append(newParticipantUsernames, user.Username)
					break
				}
			}
		}
	}

	// 根據被邀請人數決定訊息內容
	systemMessageContent := ""
	if len(newParticipantUsernames) == 1 {
		systemMessageContent = inviterUsername + " 已邀請 " + newParticipantUsernames[0] + " 加入群組。"
	} else if len(newParticipantUsernames) > 1 {
		systemMessageContent = inviterUsername + " 已邀請 " + strings.Join(newParticipantUsernames, "、") + " 加入群組。"
	}

	if systemMessageContent != "" {
		systemMessage := models.Message{
			Type:           models.MessageTypeSystem,
			RoomID:         roomID.Hex(),
			RoomName:       updatedRoom.Name,      // 使用更新後的聊天室名稱
			SenderID:       primitive.NilObjectID, // 系統訊息的 SenderID 設置為空
			SenderUsername: "系統訊息",                // 系統訊息的發送者名稱
			Content:        systemMessageContent,
			Timestamp:      time.Now(),
			IsRead:         false, // 系統訊息通常不需要已讀狀態
		}
		// 存儲系統消息
		systemMessageResult, err := database.InsertMessage(systemMessage) // 獲取 InsertResult
		if err != nil {
			log.Printf("Error inserting system invite message: %v", err)
		} else {
			// 如果插入成功，更新 systemMessage 的 ID
			systemMessage.ID = systemMessageResult.InsertedID.(primitive.ObjectID)
		}

		log.Printf("System Message ID before broadcast: %s", systemMessage.ID.Hex())

		roomUpdateMessage := models.Message{
			Type:           models.MessageTypeUpdate, // 使用新的更新消息類型
			RoomID:         roomID.Hex(),
			RoomName:       updatedRoom.Name,
			SenderID:       primitive.NilObjectID, // 系統訊息的 SenderID 設置為空
			SenderUsername: "系統更新",                // 區分於邀請通知
			Content:        "聊天室成員或名稱已更新。",        // 可以是任意通知內容，客戶端主要依賴類型判斷
			Timestamp:      time.Now(),
			IsRead:         true, // 通常不需要已讀狀態
		}
		// 存儲系統消息
		roomUpdateMessageResult, err := database.InsertMessage(roomUpdateMessage) // 獲取 InsertResult
		if err != nil {
			log.Printf("Error inserting system update message: %v", err)
		} else {
			// 如果插入成功，更新 systemMessage 的 ID
			roomUpdateMessage.ID = roomUpdateMessageResult.InsertedID.(primitive.ObjectID)
		}
		websocket.GlobalHub.Broadcast <- roomUpdateMessage // 廣播隱藏的更新消息

		// 廣播系統訊息給所有連接到該聊天室的客戶端
		websocket.GlobalHub.Broadcast <- systemMessage
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedRoom)
}

// UpdateChatRoom 處理更新聊天室的請求
// 在此新命名邏輯下，此函數可能不再用於直接更新名稱，但可用於更新其他屬性
func UpdateChatRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req UpdateChatRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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

	var participantObjectIDs []primitive.ObjectID
	for _, idStr := range req.ParticipantIDs {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid participant ID format", http.StatusBadRequest)
			return
		}
		participantObjectIDs = append(participantObjectIDs, objID)
	}

	// 根據新的參與者列表生成名稱
	usernames, err := getUsernames(participantObjectIDs)
	if err != nil {
		log.Printf("Error getting usernames: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	newRoomName := generateRoomName(usernames)

	// 更新聊天室
	updatedRoom, err := database.UpdateChatRoom(roomID, participantObjectIDs, newRoomName)
	if err != nil {
		log.Printf("Error updating chatroom: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedRoom)
}
