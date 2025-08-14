// backend/handlers/auth_integration_test.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-chat/backend/config"
	"go-chat/backend/database"
	"go-chat/backend/handlers"
	"go-chat/backend/models"
	"go-chat/backend/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestRegisterUser_Integration 是一個整合測試，測試使用者註冊的完整流程
func TestRegisterUser_Integration(t *testing.T) {
	// --- 測試準備 ---
	// 1. 建立一個真實的 MongoUserStore，它會連線到我們在 TestMain 中啟動的測試資料庫
	userStore := store.NewMongoUserStore()

	// 2. 建立一個假的 config
	cfg := &config.Config{} // 在這個測試中，RegisterUser 不會用到 cfg，所以可以給空值

	// 3. 建立我們要測試的 AuthHandler，並注入「真實」的 store
	authHandler := handlers.NewAuthHandler(userStore, cfg)

	// --- 測試案例: 成功註冊一個新使用者 ---
	t.Run("成功註冊", func(t *testing.T) {
		// 準備請求 body
		registerCredentials := map[string]string{
			"email":    "integration-test@example.com",
			"username": "integration-user",
			"password": "password123",
		}
		body, _ := json.Marshal(registerCredentials)

		// 建立 HTTP 請求與回應記錄器
		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// --- 執行 handler ---
		authHandler.RegisterUser(rr, req)

		// --- 斷言 HTTP 回應 ---
		// 斷言 HTTP 狀態碼是否為 201 Created
		assert.Equal(t, http.StatusCreated, rr.Code, "狀態碼應該是 201")

		// --- 直接驗證資料庫 ---
		// 這是整合測試的關鍵：我們直接檢查資料庫的狀態，確認資料是否真的被寫入了。
		usersCollection := database.GetCollection("users")
		var createdUser models.User
		// 使用 require，如果這一步出錯，測試將直接停止，因為後續的斷言都沒有意義了
		err := usersCollection.FindOne(context.Background(), bson.M{"email": "integration-test@example.com"}).Decode(&createdUser)
		require.NoError(t, err, "在資料庫中應該要能找到剛剛建立的使用者")

		// 斷言資料庫中的資料是否正確
		assert.Equal(t, "integration-user", createdUser.Username, "資料庫中的 username 不符預期")
		// 密碼應該是被 hash 過的，所以它不應該等於原始密碼
		assert.NotEmpty(t, createdUser.Password, "密碼欄位不應為空")
		assert.NotEqual(t, "password123", createdUser.Password, "儲存的密碼應該是 hash 過的")
	})

	// --- 測試案例: 註冊一個已存在的 email ---

	t.Run("註冊失敗 - Email已存在", func(t *testing.T) {
		// --- 第一步：先成功註冊一個使用者，為測試製造「已存在的 Email」---
		// 1. 準備請求的 Body
		registerCredentials := map[string]string{
			"email":    "existMail@example.com",
			"username": "exist-user",
			"password": "password123",
		}
		body, _ := json.Marshal(registerCredentials)

		// 2. 建立假的 HTTP 請求和回應記錄器
		req1 := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr1 := httptest.NewRecorder()

		// 在這裡，我們需要準備一個註冊請求，並呼叫一次 authHandler.RegisterUser
		authHandler.RegisterUser(rr1, req1)

		// 4. 斷言第一次請求是成功的
		//    這能確保我們的測試設定是正確的
		assert.Equal(t, http.StatusCreated, rr1.Code, "第一次註冊應該要成功")

		// --- 第二步：嘗試使用「完全相同」的 Email 再次註冊 ---
		req2 := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr2 := httptest.NewRecorder()
		authHandler.RegisterUser(rr2, req2)

		assert.Equal(t, http.StatusConflict, rr2.Code, "用已存在的 Email 註冊，狀態碼應該是 409 Conflict")

		var errorResponse models.ErrorResponse
		err := json.Unmarshal(rr2.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Equal(t, "Email already registered", errorResponse.Message)

	})

}
