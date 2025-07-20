package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go-chat/backend/database"
	"go-chat/backend/models"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ... (ChatHistoryResponse 和 HandleChatHistory 保持不變) ...
// ChatHistoryResponse 代表聊天記錄的回應結構
type ChatHistoryResponse struct {
	Messages []models.Message `json:"messages"`
	Error    string           `json:"error,omitempty"`
}

// HandleChatHistory 處理獲取聊天記錄的請求
func HandleChatHistory(w http.ResponseWriter, r *http.Request) {
	// 從 URL 查詢參數提取聊天室 ID
	roomID := r.URL.Query().Get("roomId")

	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	// 獲取聊天記錄
	messages, err := database.GetChatHistory(roomID)
	if err != nil {
		log.Printf("Error getting chat history for room %s: %v", roomID, err)
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
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 代表一個 WebSocket 客戶端
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan models.Message
	UserID   primitive.ObjectID
	Username string
	// 【修改】RoomID 和 RoomName 不再是必要項，僅表示用戶當前活躍的房間
	ActiveRoomID   string
	ActiveRoomName string
}

// readPump 讀取用戶傳來的訊息，並丟給 Hub
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
				log.Printf("Client %s disconnected gracefully.", c.UserID.Hex())
			} else {
				log.Printf("Error reading message for client %s: %v", c.UserID.Hex(), err)
			}
			break
		}

		var msg models.Message
		if err := json.Unmarshal(p, &msg); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		// 填充發送者資訊和時間戳
		msg.SenderID = c.UserID
		msg.SenderUsername = c.Username
		msg.Timestamp = time.Now()
		// RoomID 和 RoomName 應該由客戶端發送訊息時提供，因為現在一個連線可能對應多個房間
		// msg.Type 預設為 normal

		// 檢查 RoomID 是否有效
		if msg.RoomID == "" {
			log.Printf("Message from user %s missing RoomID", c.UserID.Hex())
			continue
		}

		// 為了確保 RoomName 是最新的，我們從資料庫獲取
		roomID, err := primitive.ObjectIDFromHex(msg.RoomID)
		if err != nil {
			log.Printf("Invalid RoomID format: %s", msg.RoomID)
			continue
		}

		room, err := database.FindChatRoomByID(roomID)
		if err != nil || room == nil {
			log.Printf("Room not found or database error for RoomID: %s", msg.RoomID)
			continue
		}
		msg.RoomName = room.Name

		result, err := database.InsertMessage(msg)
		if err != nil {
			log.Printf("Error saving message to database: %v", err)
			continue
		}

		msg.ID = result.InsertedID.(primitive.ObjectID)

		// 將帶有 ID 的完整訊息廣播出去
		c.hub.Broadcast <- msg
	}
}

// writePump 保持不變
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshalling message: %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, jsonMessage); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Hub 維護所有活躍的 WebSocket 客戶端
type Hub struct {
	clients map[*Client]bool
	// 【核心修改】不再按房間分組，而是按使用者 ID 索引，確保每個使用者只有一個連線
	clientsByUserID map[primitive.ObjectID]*Client
	Broadcast       chan models.Message
	register        chan *Client
	unregister      chan *Client
}

// NewHub 創建並返回一個新的 Hub 實例
func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan models.Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		// 【核心修改】初始化新的 map
		clientsByUserID: make(map[primitive.ObjectID]*Client),
	}
}

// Run 啟動 Hub 的運行迴圈
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// 【核心修改】註冊邏輯
			// 如果此 user ID 已有連線，先斷開舊的
			if oldClient, ok := h.clientsByUserID[client.UserID]; ok {
				log.Printf("User %s already connected, closing old connection.", client.UserID.Hex())
				close(oldClient.send)
				delete(h.clients, oldClient)
			}
			h.clients[client] = true
			h.clientsByUserID[client.UserID] = client
			log.Printf("Client %s (%s) registered. Total clients: %d", client.UserID.Hex(), client.Username, len(h.clients))

		case client := <-h.unregister:
			// 【核心修改】取消註冊邏輯
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// 確保我們只刪除與當前 client 物件完全相同的連線
				if h.clientsByUserID[client.UserID] == client {
					delete(h.clientsByUserID, client.UserID)
				}
				close(client.send)
				log.Printf("Client %s (%s) unregistered. Total clients: %d", client.UserID.Hex(), client.Username, len(h.clients))
			}

		case message := <-h.Broadcast:
			// 【核心修改】廣播邏輯
			roomID, err := primitive.ObjectIDFromHex(message.RoomID)
			if err != nil {
				log.Printf("Invalid RoomID in broadcast message: %s", message.RoomID)
				continue
			}

			// 根據 RoomID 從資料庫查找聊天室，以獲取完整的參與者列表
			room, err := database.FindChatRoomByID(roomID) //
			if err != nil {
				log.Printf("Error finding room for broadcast: %v", err)
				continue
			}
			if room == nil {
				log.Printf("Room %s not found for broadcasting", roomID.Hex())
				continue
			}

			// 遍歷聊天室的所有參與者
			for _, participantID := range room.Participants {
				// 檢查參與者是否在線
				if client, ok := h.clientsByUserID[participantID]; ok {
					select {
					case client.send <- message:
					default:
						// 如果發送失敗（通道已滿或關閉），則認為客戶端已離線
						log.Printf("Client %s channel full or closed, unregistering.", client.UserID.Hex())
						close(client.send)
						delete(h.clients, client)
						delete(h.clientsByUserID, client.UserID)
					}
				}
			}
		}
	}
}

// 全局 Hub 實例
var GlobalHub = NewHub()

// BroadcastMessage 保持不變，它只是將訊息放入廣播通道
func BroadcastMessage(message models.Message) {
	GlobalHub.Broadcast <- message
}

// HandleConnections 處理 WebSocket 連線請求
func HandleConnections(w http.ResponseWriter, r *http.Request) {
	// 步驟 1: 記錄收到請求
	log.Printf("DEBUG: New request to HandleConnections. Raw query: %s", r.URL.RawQuery)

	// 步驟 2: 解析參數
	userIDStr := r.URL.Query().Get("userId")
	username := r.URL.Query().Get("username")
	log.Printf("DEBUG: Parsed userId: '%s', username: '%s'", userIDStr, username)

	// 步驟 3: 驗證參數是否為空
	if userIDStr == "" || username == "" {
		log.Println("ERROR: Connection rejected. Reason: User ID or username is empty.")
		http.Error(w, "User ID and username are required for WebSocket connection", http.StatusBadRequest)
		return
	}

	// 步驟 4: 驗證 User ID 格式
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		log.Printf("ERROR: Connection rejected. Reason: Invalid User ID format. Error: %v", err)
		http.Error(w, "Invalid User ID format", http.StatusBadRequest)
		return
	}

	// 步驟 5: 升級連線為 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// upgrader.Upgrade 會自動寫入 HTTP 錯誤回應，所以這裡只需要記錄日誌
		log.Printf("ERROR: Connection rejected. Reason: Failed to upgrade to WebSocket. Error: %v", err)
		return
	}

	// 如果能執行到這裡，表示連線成功
	log.Println("DEBUG: WebSocket upgrade successful. Proceeding to register client.")

	client := &Client{
		hub:      GlobalHub,
		conn:     conn,
		send:     make(chan models.Message, 256),
		UserID:   userID,
		Username: username,
	}
	client.hub.register <- client

	go client.writePump()
	client.readPump()
}
