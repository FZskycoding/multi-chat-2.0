// backend/middleware/auth_middleware.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"go-chat/backend/config" // 確保 config 包能獲取 JWT Secret
	"go-chat/backend/utils"
)

// JWTMiddleware 驗證 JWT Token 並將使用者 ID 放入 context
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 從環境變數或配置中獲取 JWT Secret
		cfg := config.LoadConfig() // 假設 config.LoadConfig() 可以多次安全調用，或者您有其他方式傳遞配置
		jwtSecret := cfg.JWTSecret

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Authorization: Bearer <token>
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		userID, err := utils.GetUserIDFromToken(tokenString, jwtSecret)
		if err != nil {
			log.Printf("Invalid JWT token: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// 將使用者 ID 存儲到請求的 context 中
		ctx := context.WithValue(r.Context(), utils.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
