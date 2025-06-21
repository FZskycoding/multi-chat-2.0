package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go-chat/backend/database" // 引入 database 套件
	"go-chat/backend/models"   // 引入 models 套件

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatHistoryResponse 代表聊天記錄的回應結構
type ChatHistoryResponse struct {
	Messages []models.Message `json:"messages"`
	Error    string           `json:"error,omitempty"`
}

// HandleChatHistory 處理獲取聊天記錄的請求
func HandleChatHistory(w http.ResponseWriter, r *http.Request) {
	// 從 URL 查詢參數獲取用戶 ID
	user1IDStr := r.URL.Query().Get("user1Id")
	user2IDStr := r.URL.Query().Get("user2Id")

	if user1IDStr == "" || user2IDStr == "" {
		http.Error(w, "Both user IDs are required", http.StatusBadRequest)
		return
	}

	// 轉換字符串ID為ObjectID
	user1ID, err := primitive.ObjectIDFromHex(user1IDStr)
	if err != nil {
		http.Error(w, "Invalid user1Id format", http.StatusBadRequest)
		return
	}

	user2ID, err := primitive.ObjectIDFromHex(user2IDStr)
	if err != nil {
		http.Error(w, "Invalid user2Id format", http.StatusBadRequest)
		return
	}

	// 獲取聊天記錄
	messages, err := database.GetChatHistory(user1ID, user2ID)
	if err != nil {
		log.Printf("Error getting chat history: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 設置響應頭
	w.Header().Set("Content-Type", "application/json")

	// 返回聊天記錄
	response := ChatHistoryResponse{
		Messages: messages,
	}
	json.NewEncoder(w).Encode(response)
}

const (
	// 將訊息寫入到遠端對等點的最長時間
	writeWait = 10 * time.Second

	// 允許從遠端對等點讀取下一個 pong 訊息的最長時間。
	pongWait = 60 * time.Second

	// 發送 ping 訊息給遠端對等點的週期。
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// upgrader 用於將 HTTP 連線升級為 WebSocket 連線
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 設定true:允許所有來源的連線
		return true
	},
}

// Client 代表一個 WebSocket 客戶端
type Client struct {
	hub      *Hub                // 負責管理所有客戶端和訊息流
	conn     *websocket.Conn     //WebSocket 連線物件，透過它來讀寫訊息
	send     chan models.Message // 用於發送訊息的緩衝通道，類型改為 models.Message
	UserID   primitive.ObjectID
	Username string
}

// 讀取用戶傳來的訊息，並丟給 Hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Client disconnected gracefully.")
			} else {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		// 解析收到的訊息為 models.Message
		var msg models.Message
		if err := json.Unmarshal(p, &msg); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		// 填充發送者資訊和時間戳
		msg.SenderID = c.UserID
		msg.SenderUsername = c.Username
		msg.Timestamp = time.Now()

		// 如果類型未指定，則預設為聊天訊息
		if msg.Type == "" {
			msg.Type = models.MessageTypeChat
		}

		// 檢查是否為聊天室訊息
		if msg.RoomID != primitive.NilObjectID {
			// 檢查用戶是否為聊天室成員
			if clients, ok := c.hub.roomClients[msg.RoomID]; !ok || !clients[c] {
				log.Printf("User %s attempted to send message to room %s without being a member",
					c.UserID.Hex(), msg.RoomID.Hex())
				continue
			}
		}

		// 將訊息儲存到資料庫並獲得結果
		result, err := database.InsertMessage(msg)
		if err != nil {
			log.Printf("Error saving message to database: %v", err)
			return
		}

		// 設置訊息的 ID 為資料庫生成的唯一 ID
		msg.ID = result.InsertedID.(primitive.ObjectID)

		// 將包含 ID 的訊息廣播給所有客戶端
		c.hub.broadcast <- msg
	}
}

// 接收 Hub 廣播來的訊息，丟給前端
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		//負責處理從 Hub 或其他地方發送到 client.send 通道的實際聊天訊息，並將其傳送給客戶端瀏覽器。
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 如果這個 channel 被關閉了（ok == false），就送出 CloseMessage
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 將 models.Message(struct結構) 轉換為 JSON 格式發送
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshalling message: %v", err)
				return
			}

			//c.conn.WriteMessage(發送文字訊息, 發送內容)
			if err := c.conn.WriteMessage(websocket.TextMessage, jsonMessage); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

		// 接收定時器以保持連線活躍並檢測客戶端是否仍在線。
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Hub 維護所有活躍的 WebSocket 客戶端，並處理訊息的廣播
type Hub struct {
	clients         map[*Client]bool
	clientsByUserID map[primitive.ObjectID]*Client
	roomClients     map[primitive.ObjectID]map[*Client]bool // 每個聊天室的客戶端列表
	broadcast       chan models.Message
	register        chan *Client
	unregister      chan *Client
	invitation      chan models.Invitation // 處理邀請的通道
}

// NewHub 創建並返回一個新的 Hub 實例
func NewHub() *Hub {
	return &Hub{
		broadcast:       make(chan models.Message),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		clientsByUserID: make(map[primitive.ObjectID]*Client),
		roomClients:     make(map[primitive.ObjectID]map[*Client]bool),
		invitation:      make(chan models.Invitation),
	}
}

// JoinRoom 將客戶端加入聊天室
func (h *Hub) JoinRoom(roomID primitive.ObjectID, client *Client) {
	if _, ok := h.roomClients[roomID]; !ok {
		h.roomClients[roomID] = make(map[*Client]bool)
	}
	h.roomClients[roomID][client] = true

	// 發送系統消息通知其他成員
	joinMessage := models.Message{
		Type:           models.MessageTypeSystem,
		RoomID:         roomID,
		Content:        fmt.Sprintf("%s 已加入聊天室", client.Username),
		Timestamp:      time.Now(),
		SenderID:       client.UserID,
		SenderUsername: "系統",
	}

	// 向聊天室內的所有成員廣播系統消息
	for c := range h.roomClients[roomID] {
		select {
		case c.send <- joinMessage:
		default:
			close(c.send)
			delete(h.roomClients[roomID], c)
			delete(h.clients, c)
			delete(h.clientsByUserID, c.UserID)
		}
	}
}

// LeaveRoom 將客戶端從聊天室移除
func (h *Hub) LeaveRoom(roomID primitive.ObjectID, client *Client) {
	if clients, ok := h.roomClients[roomID]; ok {
		delete(clients, client)
		// 如果聊天室沒有其他成員了，刪除這個聊天室的映射
		if len(clients) == 0 {
			delete(h.roomClients, roomID)
		}
	}
}

// BroadcastToRoom 向特定聊天室的所有成員廣播消息
func (h *Hub) BroadcastToRoom(roomID primitive.ObjectID, message models.Message) {
	if clients, ok := h.roomClients[roomID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
				delete(h.clients, client)
				delete(h.clientsByUserID, client.UserID)
			}
		}
	}
}

// Run 啟動 Hub 的運行迴圈
func (h *Hub) Run() {
	for {
		select {
		// 處理新客戶端註冊
		case client := <-h.register:
			h.clients[client] = true
			h.clientsByUserID[client.UserID] = client
			log.Printf("Client registered. Total clients: %d", len(h.clients))

		// 處理用戶離線
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				// 從所有聊天室中移除該客戶端
				for roomID := range h.roomClients {
					h.LeaveRoom(roomID, client)
				}
				delete(h.clients, client)
				delete(h.clientsByUserID, client.UserID)
				close(client.send)
				log.Printf("Client unregistered. Total clients: %d", len(h.clients))
			}

		// 處理訊息廣播
		case message := <-h.broadcast:
			if message.RecipientID != primitive.NilObjectID {
				// 私人訊息處理
				if recipientClient, ok := h.clientsByUserID[message.RecipientID]; ok {
					select {
					case recipientClient.send <- message:
					default:
						close(recipientClient.send)
						delete(h.clients, recipientClient)
						delete(h.clientsByUserID, recipientClient.UserID)
					}
				}

				// 同時發送給發送者
				if senderClient, ok := h.clientsByUserID[message.SenderID]; ok {
					if senderClient.UserID != message.RecipientID {
						select {
						case senderClient.send <- message:
						default:
							close(senderClient.send)
							delete(h.clients, senderClient)
							delete(h.clientsByUserID, senderClient.UserID)
						}
					}
				}
			} else if message.RoomID != primitive.NilObjectID {
				// 聊天室訊息處理
				// 所有聊天室消息直接廣播（包括系統消息）
				h.BroadcastToRoom(message.RoomID, message)
			} else {
				log.Printf("Warning: Message without proper routing information received: %v", message)
			}

		// 處理邀請
		case invitation := <-h.invitation:
			// 創建邀請記錄
			result, err := database.CreateInvitation(invitation)
			if err != nil {
				log.Printf("Error creating invitation: %v", err)
				continue
			}

			// 儲存邀請 ID 供後續使用
			invitationID := result.InsertedID.(primitive.ObjectID)
			_ = invitationID // 暫時不使用，但保留以供將來擴展

			// 向被邀請者發送邀請通知
			if inviteeClient, ok := h.clientsByUserID[invitation.InviteeID]; ok {
				inviteMessage := models.Message{
					Type:           models.MessageTypeSystem,
					Content:        fmt.Sprintf("您被邀請加入聊天室，請接受或拒絕"),
					SenderID:       invitation.InviterID,
					RecipientID:    invitation.InviteeID,
					RoomID:         invitation.RoomID,
					Timestamp:      time.Now(),
					SenderUsername: "系統",
				}

				select {
				case inviteeClient.send <- inviteMessage:
				default:
					close(inviteeClient.send)
					delete(h.clients, inviteeClient)
					delete(h.clientsByUserID, inviteeClient.UserID)
				}
			} else {
				log.Printf("Invitee %s is offline", invitation.InviteeID.Hex())
			}
		}
	}
}

// 全局 Hub 實例
var GlobalHub = NewHub()

// HandleInvitation 處理聊天室邀請
func HandleInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var invitation models.Invitation
	if err := json.NewDecoder(r.Body).Decode(&invitation); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	GlobalHub.invitation <- invitation
	w.WriteHeader(http.StatusOK)
}

// HandleJoinRoom 處理加入聊天室請求
func HandleJoinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("userId")
	roomID := r.URL.Query().Get("roomId")
	invitationID := r.URL.Query().Get("invitationId")

	if userID == "" || roomID == "" || invitationID == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// 轉換 ID 為 ObjectID
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	roomObjID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	invitationObjID, err := primitive.ObjectIDFromHex(invitationID)
	if err != nil {
		http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
		return
	}

	// 更新邀請狀態
	if err := database.UpdateInvitationStatus(invitationObjID, "accepted"); err != nil {
		http.Error(w, "Failed to update invitation status", http.StatusInternalServerError)
		return
	}

	// 將用戶加入聊天室
	if err := database.AddMemberToChatRoom(roomObjID, userObjID); err != nil {
		http.Error(w, "Failed to add member to chat room", http.StatusInternalServerError)
		return
	}

	// 如果用戶已連線，將其加入聊天室的 WebSocket 連線
	if client, ok := GlobalHub.clientsByUserID[userObjID]; ok {
		GlobalHub.JoinRoom(roomObjID, client)
	}

	w.WriteHeader(http.StatusOK)
}

// 處理 WebSocket 連線請求
func HandleConnections(w http.ResponseWriter, r *http.Request) {
	// 從 URL 查詢參數中獲取 UserID 和 Username
	userIDStr := r.URL.Query().Get("userId")
	username := r.URL.Query().Get("username")

	if userIDStr == "" || username == "" {
		http.Error(w, "User ID and username are required for WebSocket connection", http.StatusBadRequest)
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "Invalid User ID format", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	client := &Client{
		hub:      GlobalHub,
		conn:     conn,
		send:     make(chan models.Message, 256),
		UserID:   userID,
		Username: username,
	}
	client.hub.register <- client

	// 在單獨的 goroutine 中發送歷史訊息
	go func() {
		// 獲取最近的 50 條歷史訊息
		historicalMessages, err := database.GetMessages(50)
		if err != nil {
			log.Printf("Error getting historical messages: %v", err)
			return
		}

		// 將歷史訊息發送給新連接的客戶端
		for i := len(historicalMessages) - 1; i >= 0; i-- { // 反向發送以確保順序
			select {
			case client.send <- historicalMessages[i]:
			case <-time.After(time.Second): // 防止阻塞(如果訊息放入時等待超過1秒鐘就return)
				log.Printf("Timeout sending historical message to client %s", client.UserID.Hex())
				return
			}
		}

		// 獲取並發送未讀訊息
		unreadMessages, err := database.GetUnreadMessages(client.UserID)
		if err != nil {
			log.Printf("Error getting unread messages for client %s: %v", client.UserID.Hex(), err)
			return
		}

		var unreadMessageIDs []primitive.ObjectID
		for _, msg := range unreadMessages {
			select {
			case client.send <- msg: //嘗試將訊息推送進 client.send channel
				unreadMessageIDs = append(unreadMessageIDs, msg.ID)
				// fmt.Println(unreadMessageIDs)
			case <-time.After(time.Second): // 如果在 1 秒內沒辦法把訊息送出去的狀況
				log.Printf("Timeout sending unread message to client %s", client.UserID.Hex())
			}
		}

		// 將已發送的未讀訊息標記為已讀
		if len(unreadMessageIDs) > 0 {
			if _, err := database.MarkMessagesAsRead(unreadMessageIDs); err != nil {
				log.Printf("Error marking unread messages as read for client %s: %v", client.UserID.Hex(), err)
			}
		}
	}()

	go client.writePump()
	client.readPump() // readPump 會在連線關閉時自動取消註冊
}
