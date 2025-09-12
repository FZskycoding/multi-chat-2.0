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
			Times(1)               // 只會發生1次，如果不是1次則測試失敗

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

		// 1. 斷言 JSON body 的內容是正確的
		var loginResponse models.LoginResponse
		err := json.Unmarshal(rr.Body.Bytes(), &loginResponse)
		assert.NoError(t, err, "解析成功的回應 body 不應出錯")
		assert.Equal(t, "Login successful", loginResponse.Message)
		assert.Equal(t, mockUser.Username, loginResponse.Username) //Username必須等於我們一開始準備的mockUser名字

		// 2. 斷言 Set-Cookie 標頭存在且包含我們需要的內容
		setCookieHeader := rr.Result().Header.Get("Set-Cookie") //從回應的 Header 中，把 Set-Cookie 這一項的完整內容拿出來。
		assert.NotEmpty(t, setCookieHeader, "回應標頭中應該包含 Set-Cookie")
		assert.Contains(t, setCookieHeader, "token=", "Set-Cookie 標頭應該包含 token")
		assert.Contains(t, setCookieHeader, "HttpOnly", "Cookie 應該被設定為 HttpOnly")
		assert.Contains(t, setCookieHeader, "SameSite=Strict", "Cookie 應該被設定為 SameSite=Strict")
	})
}

func TestRegisterUser(t *testing.T) {
	// 建立一個 gomock 控制器，它管理著所有 mock 物件的生命週期和期望
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // 在測試結束時，檢查所有期望是否都已滿足

	// 建立一個 mock UserStore
	mockUserStore := mocks.NewMockUserStorer(ctrl)

	cfg := &config.Config{}
	authHandler := NewAuthHandler(mockUserStore, cfg)

	t.Run("成功註冊", func(t *testing.T) {
		registerCredentials := models.RegisterRequest{
			Email:    "newUser@gmail.com",
			Username: "newUser",
			Password: "123456",
		}
		body, _ := json.Marshal(registerCredentials)

		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		mockUserStore.EXPECT().
			CheckUserExists(gomock.Any(), registerCredentials.Email, registerCredentials.Username).
			Return(false, false, nil). // 回報「Email不存在」和「Username不存在」和資料庫沒有發生錯誤，表示可正常註冊
			Times(1)

		mockUserStore.EXPECT().
			CreateUser(gomock.Any(), gomock.Any()). // 這裡先用 gomock.Any() 簡化
			Return(primitive.NewObjectID(), nil).   // 回傳一個假的、新產生的使用者 ID
			Times(1)

		authHandler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code, "狀態碼應該是 201 Created")

		var registerResponse models.RegisterResponse
		err := json.Unmarshal(rr.Body.Bytes(), &registerResponse)
		assert.NoError(t, err)
		assert.Equal(t, "User registered successfully", registerResponse.Message)
		assert.NotEmpty(t, registerResponse.ID, "回應中應該包含一個使用者ID") // 如果registerResponse.ID是空的，就顯示錯誤訊息

	})

	t.Run("註冊失敗 - Email 已存在", func(t *testing.T) {
		registerCredentials := models.RegisterRequest{
			Email:    "existing@gmail.com",
			Username: "newUser",
			Password: "123456",
		}
		body, _ := json.Marshal(registerCredentials)

		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		mockUserStore.EXPECT().
			CheckUserExists(gomock.Any(), registerCredentials.Email, registerCredentials.Username).
			Return(true, false, nil). // 回報「Email已存在」和「Username不存在」和資料庫沒有發生錯誤
			Times(1)

		authHandler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code, "狀態碼應該是 409 Conflict")

		var errorResponse models.RegisterResponse
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Equal(t, "Email already registered", errorResponse.Message, "錯誤訊息不符預期") //如果errorResponse.Message不等於"Email already registered"的話就出現錯誤訊息

	})

	t.Run("註冊失敗 - 使用者名稱已存在", func(t *testing.T) {
		registerCredentials := models.RegisterRequest{
			Email:    "existing@gmail.com",
			Username: "newUser",
			Password: "123456",
		}
		body, _ := json.Marshal(registerCredentials)

		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		mockUserStore.EXPECT().
			CheckUserExists(gomock.Any(), registerCredentials.Email, registerCredentials.Username).
			Return(false, true, nil). // 回報「Email不存在」和「Username已存在」和資料庫沒有發生錯誤
			Times(1)

		authHandler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code, "狀態碼應該是 409 Conflict")

		var errorResponse models.RegisterResponse
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Equal(t, "Username already taken", errorResponse.Message, "錯誤訊息不符預期") //如果errorResponse.Message不等於"Email already registered"的話就出現錯誤訊息

	})

	t.Run("註冊失敗 - 缺少密碼", func(t *testing.T) {
		registerCredentials := models.RegisterRequest{
			Email:    "existing@gmail.com",
			Username: "newUser",
			Password: "",
		}
		body, _ := json.Marshal(registerCredentials)

		req := httptest.NewRequest("POST", "/register", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		authHandler.RegisterUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code, "狀態碼應該是 400 Bad Request")

		var errorResponse models.RegisterResponse
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Equal(t, "Email, username, and password are required", errorResponse.Message, "錯誤訊息不符預期") //如果errorResponse.Message不等於"Email already registered"的話就出現錯誤訊息

	})

}
