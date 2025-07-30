// backend/utils/utils_test.go
package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert" // 引入 testify/assert
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGenerateJWT(t *testing.T) {
	// 準備測試資料
	userID := primitive.NewObjectID()
	username := "testuser"
	secret := "test-secret"

	// 執行要測試的函式
	tokenString, err := GenerateJWT(userID, username, secret)

	// --- 使用 testify/assert 進行斷言 ---

	// 1. 斷言錯誤為 nil
	assert.NoError(t, err, "生成 JWT 不應該返回錯誤")

	// 2. 斷言 token 字串不為空
	assert.NotEmpty(t, tokenString, "生成的 JWT token 不應該是空的")

	// 3. 解析並驗證 token 內容
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 驗證簽名演算法是否正確
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		assert.True(t, ok, "非預期的簽名演算法")
		return []byte(secret), nil
	})

	// 斷言 token 解析成功且有效
	assert.NoError(t, err, "解析 JWT token 不應該返回錯誤")
	assert.True(t, token.Valid, "JWT token 應該是有效的")

	// 4. 驗證 token 的聲明 (Claims)
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok, "無法讀取 JWT claims")

	// 驗證 claims 內容是否符合預期
	assert.Equal(t, userID.Hex(), claims["userId"], "userId claim 應該與原始 userID 相同")
	assert.Equal(t, username, claims["username"], "username claim 應該與原始 username 相同")

	// 驗證過期時間 (exp) 是否在未來
	exp, ok := claims["exp"].(float64)
	assert.True(t, ok, "exp claim 格式錯誤")
	assert.Greater(t, int64(exp), time.Now().Unix(), "過期時間應該在未來")
}
