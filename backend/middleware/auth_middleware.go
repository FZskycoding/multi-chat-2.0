// backend/middleware/auth_middleware.go
package middleware

import (
	"context"
	"log"
	"net/http"

	"go-chat/backend/config" // 確保 config 包能獲取 JWT Secret
	"go-chat/backend/utils"
)

// JWTMiddleware 驗證 JWT Token 並將使用者 ID 放入 context
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := config.LoadConfig()
		jwtSecret := cfg.JWTSecret

		// --- 核心修改：從 Cookie 讀取 Token ---
		cookie, err := r.Cookie("token")
		if err != nil {
			// 如果 Cookie 不存在 (http.ErrNoCookie)，或其他錯誤，都視為未授權
			if err == http.ErrNoCookie {
				http.Error(w, "Authorization cookie required", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// 從 Cookie 中獲取 token 字串
		tokenString := cookie.Value
		// --- 修改結束 ---

		// 驗證 token 的邏輯保持不變
		userID, err := utils.GetUserIDFromToken(tokenString, jwtSecret)
		if err != nil {
			log.Printf("Invalid JWT token from cookie: %v", err)
			// 當 token 無效或過期時，可以順便命令瀏覽器刪除這個無用的 cookie
			http.SetCookie(w, &http.Cookie{
				Name:   "token",
				Value:  "",
				Path:   "/",
				MaxAge: -1, // MaxAge < 0 會立即刪除 cookie
			})
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// 將使用者 ID 存儲到請求的 context 中
		ctx := context.WithValue(r.Context(), utils.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
