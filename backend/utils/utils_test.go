package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
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

func TestGetUserIDFromToken(t *testing.T) {
	// 準備共用的測試資料
	userID := primitive.NewObjectID()
	username := "testuser"
	secret := "a-very-secret-key"

	// --- 測試情境 1: 成功的案例 ---
	t.Run("成功案例 - 有效的 Token", func(t *testing.T) {
		// 產生一個有效的 token
		validToken, err := GenerateJWT(userID, username, secret)
		assert.NoError(t, err)

		// 執行要測試的函式
		parsedUserID, err := GetUserIDFromToken(validToken, secret)

		// 斷言結果
		assert.NoError(t, err, "解析有效的 token 不應該返回錯誤")
		assert.Equal(t, userID, parsedUserID, "解析出的 userID 應該與原始的 userID 相同")
	})

	// --- 測試情境 2: 失敗的案例 (無效簽名) ---
	t.Run("失敗案例 - 無效的簽名", func(t *testing.T) {
		// 產生一個有效的 token
		validToken, err := GenerateJWT(userID, username, secret)
		assert.NoError(t, err)

		// 嘗試用錯誤的 secret 去解析
		_, err = GetUserIDFromToken(validToken, "wrong-secret")

		// 斷言結果
		assert.Error(t, err, "使用錯誤的 secret 解析應該要返回錯誤")
		// 我們可以更精確地檢查錯誤類型或訊息
		assert.Contains(t, err.Error(), "signature is invalid", "錯誤訊息應該包含 'signature is invalid'")
	})

	// --- 測試情境 3: 失敗的案例 (格式錯誤) ---
	t.Run("失敗案例 - 格式錯誤的 Token", func(t *testing.T) {
		// 傳入一個亂寫的字串
		_, err := GetUserIDFromToken("this-is-not-a-jwt-token", secret)

		// 斷言結果
		assert.Error(t, err, "解析格式錯誤的 token 應該要返回錯誤")
	})
}
