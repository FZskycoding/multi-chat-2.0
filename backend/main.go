package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-chat/backend/config"
	"go-chat/backend/database"
	"go-chat/backend/handlers"
	"go-chat/backend/middleware"
	"go-chat/backend/websocket" // 引入 websocket 套件

	"github.com/gorilla/mux"
	"github.com/rs/cors" // 引入 CORS 庫
)

func main() {
	cfg := config.LoadConfig()

	database.ConnectMongoDB(cfg.MongoDBURI, cfg.DBName)
	defer database.DisconnectMongoDB()

	// 啟動 WebSocket Hub
	go websocket.GlobalHub.Run()

	router := mux.NewRouter()

	// 健康檢查路由 (通常不需要 JWT)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Backend is running!")
	}).Methods("GET")

	// 不需要 JWT 的路由
	router.HandleFunc("/register", handlers.RegisterUser).Methods("POST")
	router.HandleFunc("/login", handlers.LoginUser).Methods("POST")

	// --- 需要 JWT 驗證的路由 ---
	// 獲取所有使用者 API 路由 (需要登入才能看)
	router.Handle("/users", middleware.JWTMiddleware(http.HandlerFunc(handlers.GetAllUsers))).Methods("GET")

	// 聊天室相關路由 (需要登入才能操作)
router.Handle("/chatrooms", middleware.JWTMiddleware(http.HandlerFunc(handlers.CreateChatRoom))).Methods("POST")
router.Handle("/user-chatrooms", middleware.JWTMiddleware(http.HandlerFunc(handlers.GetUserChatRooms))).Methods("GET")
router.Handle("/chatrooms/{id}", middleware.JWTMiddleware(http.HandlerFunc(handlers.UpdateChatRoom))).Methods("PUT")
router.Handle("/chatrooms/{id}/leave", middleware.JWTMiddleware(http.HandlerFunc(handlers.LeaveChatRoom))).Methods("POST")

	// WebSocket 路由 (WebSocket 連線通常通過 URL 參數或 Cookies 進行認證，而不是 Authorization Header)
	// 如果你的 WebSocket 連接在 URL 中傳遞了 token，可能需要在 HandleConnections 內部進行驗證
	router.HandleFunc("/ws", websocket.HandleConnections)
	// 聊天記錄路由 (通常需要 JWT)
	// 這裡需要注意：websocket.HandleChatHistory 如果是 REST API 獲取歷史記錄，應該加上 JWT
	// 如果這個 /chat-history 是 WebSocket 協定的一部分，那麼應該在 HandleConnections 內部處理
	// 假設它是一個獨立的 REST API
	router.Handle("/chat-history", middleware.JWTMiddleware(http.HandlerFunc(websocket.HandleChatHistory))).Methods("GET")

	// 設置 CORS 中介軟體
	// 允許來自任何來源的請求，並允許 POST, GET, OPTIONS 方法
	// 實際生產環境中，你應該將 AllowedOrigins 限制為你的前端網域
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// 將 CORS 中介軟體應用到你的路由上
	handler := c.Handler(router)

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      handler, // 將處理器替換為帶有 CORS 的 handler
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 如果錯誤不是因為主動關閉伺服器，就記錄錯誤並結束程式
			log.Fatalf("Could not listen on %s: %v", serverAddr, err)
		}
	}()

	//當按下 Ctrl+C，程式會收到 SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Printf("Received signal %s, shutting down server...", sig)

	//最多等30秒關閉，避免資料損壞，請求中斷
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully.")
}
