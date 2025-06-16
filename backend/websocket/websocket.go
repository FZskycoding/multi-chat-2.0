package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go-chat/backend/database" // 引入 database 套件
	"go-chat/backend/models"   // 引入 models 套件

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
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

		// 設置訊息的 ID 為資料庫生成的 ID
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
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 如果這個 channel 被關閉了（ok == false），就送出 CloseMessage
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 將 models.Message 轉換為 JSON 格式發送
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshalling message: %v", err)
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(jsonMessage)

			// 檢查 channel 裡還有幾個訊息
			// 把後續的訊息一個一個繼續拿出來包進同一個 WebSocket frame 裡，
			// 每一個訊息之間，就會插入 newline，這樣訊息之間才不會黏在一起。
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				queuedMsg, err := json.Marshal(<-c.send)
				if err != nil {
					log.Printf("Error marshalling queued message: %v", err)
					continue
				}
				w.Write(queuedMsg)
			}

			if err := w.Close(); err != nil {
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

// Hub 維護所有活躍的 WebSocket 客戶端，並處理訊息的廣播
type Hub struct {
	clients         map[*Client]bool
	clientsByUserID map[primitive.ObjectID]*Client // 新增：按使用者ID索引的客戶端
	broadcast       chan models.Message            // 類型改為 models.Message
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
		case client := <-h.register:
			h.clients[client] = true
			h.clientsByUserID[client.UserID] = client // 將客戶端加入按使用者ID索引的地圖
			log.Printf("Client registered. Total clients: %d, By UserID: %d", len(h.clients), len(h.clientsByUserID))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.clientsByUserID, client.UserID) // 從按使用者ID索引的地圖中移除
				close(client.send)
				log.Printf("Client unregistered. Total clients: %d, By UserID: %d", len(h.clients), len(h.clientsByUserID))
			}
		case message := <-h.broadcast:
			if message.RecipientID != primitive.NilObjectID {
				// 一對一訊息：發送給特定接收者
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
					log.Printf("Recipient %s not found for private message.", message.RecipientID.Hex())
				}
				// ====== 新增：將訊息發送回給發送者 (即自己) ======
				if senderClient, ok := h.clientsByUserID[message.SenderID]; ok { //
					// 確保發送者不是接收者本人 (避免重複發送給同一個人，雖然不影響功能)
					if senderClient.UserID != message.RecipientID { //
						select { //
						case senderClient.send <- message: //
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
		historicalMessages, err := database.GetMessages(50)
		if err != nil {
			log.Printf("Error getting historical messages: %v", err)
			return
		}

		// 將歷史訊息發送給新連接的客戶端
		for i := len(historicalMessages) - 1; i >= 0; i-- { // 反向發送以確保順序
			// 檢查是否符合models.Message的定義
			if msg, ok := historicalMessages[i].(models.Message); ok {
				select {
				case client.send <- msg:
				case <-time.After(time.Second): // 防止阻塞(如果訊息放入時等待超過1秒鐘就return)
					log.Printf("Timeout sending historical message to client %s", client.UserID.Hex())
					return
				}
			} else {
				log.Printf("Failed to cast historical message to models.Message: %+v", historicalMessages[i])
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
			case client.send <- msg:
				unreadMessageIDs = append(unreadMessageIDs, msg.ID)
			case <-time.After(time.Second): // 防止阻塞
				log.Printf("Timeout sending unread message to client %s", client.UserID.Hex())
				break // 跳出內層 select，繼續處理下一個未讀訊息
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
