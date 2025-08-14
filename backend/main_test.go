// backend/main_test.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"go-chat/backend/database"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// TestMain 是整個測試套件的進入點
func TestMain(m *testing.M) {
	// --- 設定測試環境 ---
	ctx := context.Background()

	// 1. 建立並啟動一個 MongoDB 容器
	// 我們指定使用 mongo:6 這個 image
	mongodbContainer, err := mongodb.Run(ctx, "mongo:6")
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	// 2. 獲取容器動態分配的連線 URI
	// Testcontainers 會自動找到容器的 IP 和 Port
	uri, err := mongodbContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get connection string: %s", err)
	}

	// 3. 使用獲取到的 URI 連接到測試資料庫
	// 注意：這裡我們直接操作 database 套件的公開變數，這是為了測試
	database.ConnectMongoDB(uri, "test-db")
	fmt.Println("Successfully connected to test MongoDB container!")

	// --- 執行所有測試 ---
	// m.Run() 會執行這個套件中所有其他的 Test... 函式
	exitCode := m.Run()

	// --- 清理測試環境 ---
	// 5. 在所有測試執行完畢後，終止並移除容器
	if err := mongodbContainer.Terminate(ctx); err != nil {
		log.Fatalf("failed to terminate container: %s", err)
	}

	// 6. 退出測試
	os.Exit(exitCode)
}
