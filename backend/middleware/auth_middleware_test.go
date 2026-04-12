package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-chat/backend/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestJWTMiddleware(t *testing.T) {
	jwtSecret := "test-secret-for-middleware"

	t.Run("成功情境 - 有效的 Token", func(t *testing.T) {
		var handlerCalled bool

		expectedUserID := primitive.NewObjectID()
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			userID, err := utils.GetUserIDFromContext(r.Context())
			assert.NoError(t, err, "Context 中應該要能找到 userID")
			assert.Equal(t, expectedUserID, userID, "Context 中的 userID 應該與 token 內的一致")
			w.WriteHeader(http.StatusOK) // 表示成功處理
		})

		middleware := JWTMiddleware(nextHandler, jwtSecret)

		
		token, err := utils.GenerateJWT(expectedUserID, "testuser", jwtSecret)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected-route", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})

		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "狀態碼應該是 200 OK，因為 next handler 被成功呼叫")
		assert.True(t, handlerCalled, "Next handler 應該要被呼叫")

	})

	t.Run("失敗情境 - 沒有 Token Cookie", func(t *testing.T) {
		var handlerCalled bool

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := JWTMiddleware(nextHandler, jwtSecret)

		req := httptest.NewRequest("GET", "/protected-route", nil)
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code, "沒有 token cookie 時應該回傳 401")
		assert.False(t, handlerCalled, "沒有 token cookie 時不應該呼叫 next handler")
	})

	t.Run("失敗情境 - 無效的 Token", func(t *testing.T) {
		var handlerCalled bool

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := JWTMiddleware(nextHandler, jwtSecret)

		req := httptest.NewRequest("GET", "/protected-route", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: "invalid-token",
		})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code, "無效 token 應該回傳 401")
		assert.False(t, handlerCalled, "無效 token 時不應該呼叫 next handler")

		setCookieHeader := rr.Result().Header.Get("Set-Cookie")
		assert.NotEmpty(t, setCookieHeader, "無效 token 時應該回傳 Set-Cookie")
		assert.Contains(t, setCookieHeader, "token=", "應該清除 token cookie")
		assert.Contains(t, setCookieHeader, "Max-Age=0", "應該讓 token cookie 立即失效")
	})

	// token 是 JWT，但不是合法簽發的
	t.Run("失敗情境 - Token 簽名錯誤", func(t *testing.T) {
		var handlerCalled bool

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := JWTMiddleware(nextHandler, jwtSecret)

		userID := primitive.NewObjectID()
		wrongSecret := "wrong-secret"
		token, err := utils.GenerateJWT(userID, "testuser", wrongSecret)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected-route", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: token,
		})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code, "簽名錯誤的 token 應該回傳 401")
		assert.False(t, handlerCalled, "簽名錯誤的 token 不應該呼叫 next handler")

		setCookieHeader := rr.Result().Header.Get("Set-Cookie")
		assert.NotEmpty(t, setCookieHeader, "簽名錯誤時應該回傳 Set-Cookie")
		assert.Contains(t, setCookieHeader, "token=", "應該清除 token cookie")
		assert.Contains(t, setCookieHeader, "Max-Age=0", "應該讓 token cookie 立即失效")
	})

	t.Run("失敗情境 - 過期的 Token", func(t *testing.T) {
		var handlerCalled bool

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := JWTMiddleware(nextHandler, jwtSecret)

		userID := primitive.NewObjectID()

		claims := jwt.MapClaims{
			"userId":   userID.Hex(),
			"username": "testuser",
			"exp":      time.Now().Add(-1 * time.Hour).Unix(), // 已過期
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expiredToken, err := token.SignedString([]byte(jwtSecret))
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected-route", nil)
		req.AddCookie(&http.Cookie{
			Name:  "token",
			Value: expiredToken,
		})
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code, "過期 token 應該回傳 401")
		assert.False(t, handlerCalled, "過期 token 不應該呼叫 next handler")

		setCookieHeader := rr.Result().Header.Get("Set-Cookie")
		assert.NotEmpty(t, setCookieHeader, "過期 token 時應該回傳 Set-Cookie")
		assert.Contains(t, setCookieHeader, "token=", "應該清除 token cookie")
		assert.Contains(t, setCookieHeader, "Max-Age=0", "應該讓 token cookie 立即失效")
	})

}
