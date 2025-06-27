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
	hub        *Hub                // 負責管理所有客戶端和訊息流
	conn       *websocket.Conn     // WebSocket 連線物件，透過它來讀寫訊息
	send       chan models.Message // 用於發送訊息的緩衝通道，類型改為 models.Message
	UserID     primitive.ObjectID
	Username   string
	RoomID     string // 客戶端所在的聊天室ID
	RoomName   string // 客戶端所在的聊天室名稱
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

		// 填充發送者資訊、聊天室資訊和時間戳
		msg.SenderID = c.UserID
		msg.SenderUsername = c.Username
		msg.RoomID = c.RoomID     // 從客戶端連線資訊獲取 RoomID
		msg.RoomName = c.RoomName // 從客戶端連線資訊獲取 RoomName
		msg.Timestamp = time.Now()

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
	clientsByRoom   map[string]map[*Client]bool // 按聊天室ID索引的客戶端
	broadcast       chan models.Message
	register        chan *Client
	unregister      chan *Client
}

// NewHub 創建並返回一個新的 Hub 實例
func NewHub() *Hub {
	return &Hub{
		broadcast:       make(chan models.Message),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		clientsByRoom:   make(map[string]map[*Client]bool), // 初始化
	}
}

// Run 啟動 Hub 的運行迴圈
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			if _, ok := h.clientsByRoom[client.RoomID]; !ok {
				h.clientsByRoom[client.RoomID] = make(map[*Client]bool)
			}
			h.clientsByRoom[client.RoomID][client] = true
			log.Printf("Client %s registered to room %s. Total clients in room: %d", client.UserID.Hex(), client.RoomID, len(h.clientsByRoom[client.RoomID]))
			fmt.Println(client)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if _, ok := h.clientsByRoom[client.RoomID]; ok {
					delete(h.clientsByRoom[client.RoomID], client)
					if len(h.clientsByRoom[client.RoomID]) == 0 {
						delete(h.clientsByRoom, client.RoomID) // 如果房間沒有客戶端了，就刪除房間
					}
				}
				close(client.send)
				log.Printf("Client %s unregistered from room %s. Total clients in room: %d", client.UserID.Hex(), client.RoomID, len(h.clientsByRoom[client.RoomID]))
			}
		case message := <-h.broadcast:
			// 廣播訊息到特定聊天室
			if clientsInRoom, ok := h.clientsByRoom[message.RoomID]; ok {
				for client := range clientsInRoom {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(clientsInRoom, client)
						if len(clientsInRoom) == 0 {
							delete(h.clientsByRoom, message.RoomID)
						}
						delete(h.clients, client) // 從總客戶端列表中移除
						log.Printf("Client channel is full, unregistered client %s from room %s", client.UserID.Hex(), client.RoomID)
					}
				}
			} else {
				log.Printf("Room %s not found for broadcasting message.", message.RoomID)
			}
		}
	}
}

// 全局 Hub 實例
var GlobalHub = NewHub()

// HandleConnections 處理 WebSocket 連線請求
func HandleConnections(w http.ResponseWriter, r *http.Request) {
	// 從 URL 查詢參數中獲取 UserID, Username, RoomID 和 RoomName
	userIDStr := r.URL.Query().Get("userId")
	username := r.URL.Query().Get("username")
	roomID := r.URL.Query().Get("roomId")
	roomName := r.URL.Query().Get("roomName")

	if userIDStr == "" || username == "" || roomID == "" || roomName == "" {
		http.Error(w, "User ID, username, room ID, and room name are required for WebSocket connection", http.StatusBadRequest)
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
		RoomID:   roomID,
		RoomName: roomName,
	}
	client.hub.register <- client

	// 在單獨的 goroutine 中發送歷史訊息
	go func() {
		// 獲取最近的 50 條歷史訊息 (針對特定聊天室)
		historicalMessages, err := database.GetChatHistory(client.RoomID)
		if err != nil {
			log.Printf("Error getting historical messages for room %s: %v", client.RoomID, err)
			return
		}

		// 將歷史訊息發送給新連接的客戶端
		for i := len(historicalMessages) - 1; i >= 0; i-- { // 反向發送以確保順序
			select {
			case client.send <- historicalMessages[i]:
			case <-time.After(time.Second): // 防止阻塞(如果訊息放入時等待超過1秒鐘就return)
				log.Printf("Timeout sending historical message to client %s in room %s", client.UserID.Hex(), client.RoomID)
				return
			}
		}
	}()

	go client.writePump()
	client.readPump() // readPump 會在連線關閉時自動取消註冊
}
