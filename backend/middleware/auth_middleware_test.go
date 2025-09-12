package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-chat/backend/utils"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestJWTMiddleware(t *testing.T) {
	jwtSecret := "test-secret-for-middleware"

	t.Run("成功情境 - 有效的 Token", func(t *testing.T) {
		var handlerCalled bool
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			userID, err := utils.GetUserIDFromContext(r.Context())
			assert.NoError(t, err, "Context 中應該要能找到 userID")
			assert.NotEmpty(t, userID, "Context 中的 userID 不應該是空的")
			w.WriteHeader(http.StatusOK) // 表示成功處理
		})

		middleware := JWTMiddleware(nextHandler, jwtSecret)

		userID := primitive.NewObjectID()
		token, _ := utils.GenerateJWT(userID, "testuser", jwtSecret)
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

}
