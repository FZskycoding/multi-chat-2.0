// backend/handlers/auth_handler_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-chat/backend/config"
	"go-chat/backend/models"
	"go-chat/backend/store/mocks" 
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/mock/gomock"
)

// TestLoginUser 包含了對 LoginUser handler 的所有測試案例
func TestLoginUser(t *testing.T) {
	// --- 測試準備 ---
	// 建立一個 gomock 控制器，它管理著所有 mock 物件的生命週期和期望
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // 在測試結束時，檢查所有期望是否都已滿足

	// 建立一個 mock UserStore
	mockUserStore := mocks.NewMockUserStorer(ctrl)

	// 建立一個假的 config
	cfg := &config.Config{
		JWTSecret: "test-secret-for-login",
	}

	// 建立我們要測試的 AuthHandler，並注入 mock store 和假 config
	authHandler := NewAuthHandler(mockUserStore, cfg)

	// --- 測試案例 1: 登入失敗 - 使用者不存在 ---
	t.Run("登入失敗 - 使用者不存在", func(t *testing.T) {
		// 準備請求的 body
		loginCredentials := map[string]string{
			"email":    "notfound@example.com",
			"password": "somepassword",
		}
		body, _ := json.Marshal(loginCredentials)

		// 建立一個假的 HTTP 請求
		req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
		// 建立一個 response recorder 來捕捉 handler 的回應
		rr := httptest.NewRecorder()

		// --- 設定 Mock 的期望行為 ---
		// 我們期望 FindUserByEmail 方法會被呼叫一次
		// 參數是任何 context 和我們指定的 email
		// 當它被呼叫時，它應該回傳 mongo.ErrNoDocuments (代表查無此人)
		mockUserStore.EXPECT().
			FindUserByEmail(gomock.Any(), "notfound@example.com").
			Return(nil, mongo.ErrNoDocuments).
			Times(1)

		// --- 執行被測試的 handler ---
		authHandler.LoginUser(rr, req)

		// --- 斷言結果 ---
		// 斷言 HTTP 狀態碼是否為 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, rr.Code, "狀態碼應該是 401")

		// 斷言回應的 JSON body 是否包含我們預期的錯誤訊息
		var errorResponse models.ErrorResponse
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid credentials", errorResponse.Message, "回應的錯誤訊息不符預期")
	})

	// --- 測試案例 2: 登入失敗 - 密碼錯誤 ---
	t.Run("登入失敗 - 密碼錯誤", func(t *testing.T) {
		// 準備請求的 body
		loginCredentials := map[string]string{
			"email":    "user@example.com",
			"password": "wrong-password",
		}
		body, _ := json.Marshal(loginCredentials)

		req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// 準備一個假的使用者資料，和一個預先算好的密碼 hash
		// 這裡我們用 bcrypt 將 "correct-password" 進行 hash
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
		mockUser := &models.User{
			ID:       primitive.NewObjectID(),
			Email:    "user@example.com",
			Username: "testuser",
			Password: string(hashedPassword),
		}

		// --- 設定 Mock 的期望行為 ---
		// 期望 FindUserByEmail 被呼叫，並且這次要回傳我們準備好的假使用者
		mockUserStore.EXPECT().
			FindUserByEmail(gomock.Any(), "user@example.com").
			Return(mockUser, nil). // 回傳 mockUser 和 nil 錯誤
			Times(1)

		// --- 執行 handler ---
		authHandler.LoginUser(rr, req)

		// --- 斷言結果 ---
		assert.Equal(t, http.StatusUnauthorized, rr.Code, "狀態碼應該是 401")
		var errorResponse models.ErrorResponse
		json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		assert.Equal(t, "Invalid credentials", errorResponse.Message, "密碼錯誤時，回應的錯誤訊息不符預期")
	})

	// --- 測試案例 3: 登入成功 ---
	t.Run("登入成功", func(t *testing.T) {
		// 準備請求的 body，這次用正確的密碼
		loginCredentials := map[string]string{
			"email":    "user@example.com",
			"password": "correct-password",
		}
		body, _ := json.Marshal(loginCredentials)

		req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// 準備假的使用者資料
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
		mockUser := &models.User{
			ID:       primitive.NewObjectID(),
			Email:    "user@example.com",
			Username: "testuser",
			Password: string(hashedPassword),
		}

		// --- 設定 Mock 的期望行為 ---
		mockUserStore.EXPECT().
			FindUserByEmail(gomock.Any(), "user@example.com").
			Return(mockUser, nil).
			Times(1)

		// --- 執行 handler ---
		authHandler.LoginUser(rr, req)

		// --- 斷言結果 ---
		assert.Equal(t, http.StatusOK, rr.Code, "狀態碼應該是 200 OK")

		// 斷言回應的 body 中有 token
		var loginResponse models.LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &loginResponse)
		assert.NoError(t, err, "解析成功的回應 body 不應出錯")
		assert.Equal(t, "Login successful", loginResponse.Message)
		assert.Equal(t, mockUser.Username, loginResponse.Username)
	})
}
