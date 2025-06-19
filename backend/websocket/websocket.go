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
	// 從 URL 查詢參數提取用戶 ID
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
	clientsByUserID map[primitive.ObjectID]*Client // 按使用者ID索引的客戶端
	broadcast       chan models.Message            
	register        chan *Client
	unregister      chan *Client
}

// NewHub 創建並返回一個新的 Hub 實例
func NewHub() *Hub {
	return &Hub{
		broadcast:       make(chan models.Message), // 類型改為 models.Message
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		clientsByUserID: make(map[primitive.ObjectID]*Client), // 初始化
	}
}

// Run 啟動 Hub 的運行迴圈
func (h *Hub) Run() {
	for {
		select {
		// 一個新的 WebSocket 連線建立後，檢測到 h.register channel 有資料傳入
		// 將這個新的 client 物件加入到 h.clients 這個 map 中
		// 表示這個客戶端目前是活躍的
		case client := <-h.register:
			h.clients[client] = true
			h.clientsByUserID[client.UserID] = client 
			log.Printf("Client registered. Total clients: %d, By UserID: %d", len(h.clients), len(h.clientsByUserID))
			fmt.Println(client)
		// 當一個客戶端斷開 WebSocket 連線時
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok { //如果這個client真的存在，就從按使用者ID索引的地圖中移除
				delete(h.clients, client)
				delete(h.clientsByUserID, client.UserID) 
				close(client.send)
				log.Printf("Client unregistered. Total clients: %d, By UserID: %d", len(h.clients), len(h.clientsByUserID))
			}
		case message := <-h.broadcast:
			// 檢查 RecipientID 不為空
			if message.RecipientID != primitive.NilObjectID {
				// 檢查接收者是否登入
				if recipientClient, ok := h.clientsByUserID[message.RecipientID]; ok {

					select {
					case recipientClient.send <- message:
					default:
						close(recipientClient.send)
						delete(h.clients, recipientClient)
						delete(h.clientsByUserID, recipientClient.UserID)
						log.Printf("Recipient client channel is full, unregistered client %s", recipientClient.UserID.Hex())
					}
				} else {
					// 顯示接收者未登入
					log.Printf("Recipient %s is offline", message.RecipientID.Hex())
				}
				// 將訊息發送回給發送者自己
				// 檢查發送者是否登入
				if senderClient, ok := h.clientsByUserID[message.SenderID]; ok { //
					// 確保發送者不是接收者本人 (避免重複發送給同一個人，雖然不影響功能)
					if senderClient.UserID != message.RecipientID { //
						select { //
						case senderClient.send <- message:
						default: //
							close(senderClient.send)                                                                                      //
							delete(h.clients, senderClient)                                                                               //
							delete(h.clientsByUserID, senderClient.UserID)                                                                //
							log.Printf("Sender client channel is full, unregistered client %s (self-receipt)", senderClient.UserID.Hex()) //
						}
					}
				} else { //
					log.Printf("Sender %s not found for self-receipt (可能是剛離線).", message.SenderID.Hex()) //
				}
			} else {
				// 廣播訊息：發送給所有客戶端
				for client := range h.clients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
						delete(h.clientsByUserID, client.UserID)
						log.Printf("Client channel is full, unregistered client %s", client.UserID.Hex())
					}
				}
			}
		}
	}
}

// 全局 Hub 實例
var GlobalHub = NewHub()

// HandleConnections 處理 WebSocket 連線請求
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
		send:     make(chan models.Message, 256), // 類型改為 models.Message
		UserID:   userID,
		Username: username,
	}
	client.hub.register <- client

	// 在單獨的 goroutine 中發送歷史訊息
	go func() {
		// 獲取最近的 50 條歷史訊息
		historicalMessages, err := database.GetMessages(10)
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
