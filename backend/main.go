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

	"github.com/gorilla/mux"
	"github.com/rs/cors" // 引入 CORS 庫
)

func main() {
	cfg := config.LoadConfig()

	database.ConnectMongoDB(cfg.MongoDBURI, cfg.DBName)
	defer database.DisconnectMongoDB()

	router := mux.NewRouter()

	// 健康檢查路由
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Backend is running!")
	}).Methods("GET")

	// 註冊 API 路由
	router.HandleFunc("/register", handlers.RegisterUser).Methods("POST")
	// 登入 API 路由
	router.HandleFunc("/login", handlers.LoginUser).Methods("POST")
	// 新增：獲取所有使用者 API 路由
	router.HandleFunc("/users", handlers.GetAllUsers).Methods("GET") 

	// 設置 CORS 中介軟體
	// 允許來自任何來源的請求，並允許 POST, GET, OPTIONS 方法
	// 實際生產環境中，你應該將 AllowedOrigins 限制為你的前端網域
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"}, 
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
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
