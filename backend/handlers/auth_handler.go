package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"go-chat/backend/config"
	"go-chat/backend/database"
	"go-chat/backend/models"
	"go-chat/backend/store"
	"go-chat/backend/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler 包含處理認證請求的所有依賴
type AuthHandler struct {
	UserStore store.UserStorer // <<<--- 依賴於介面，而不是具體實作
	Cfg       *config.Config
}

// NewAuthHandler 是一個工廠函式，用於建立新的 AuthHandler
func NewAuthHandler(userStore store.UserStorer, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		UserStore: userStore,
		Cfg:       cfg,
	}
}

// sendJSONError 統一發送 JSON 格式錯誤響應
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	var errorResponse models.ErrorResponse
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse.Message = message
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

// RegisterUser 處理使用者註冊請求
func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var registerReq models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		sendJSONError(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if registerReq.Email == "" || registerReq.Username == "" || registerReq.Password == "" {
		sendJSONError(w, "Email, username, and password are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// --- 使用 UserStore 介面進行操作 ---
	emailExists, usernameExists, err := h.UserStore.CheckUserExists(ctx, registerReq.Email, registerReq.Username)
	if err != nil {
		log.Printf("Error checking existing user: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if emailExists {
		sendJSONError(w, "Email already registered", http.StatusConflict)
		return
	}
	if usernameExists {
		sendJSONError(w, "Username already taken", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerReq.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user := models.User{
		Email:    registerReq.Email,
		Username: registerReq.Username,
		Password: string(hashedPassword),
	}

	// --- 使用 UserStore 介面進行操作 ---
	insertedID, err := h.UserStore.CreateUser(ctx, user)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		sendJSONError(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	log.Printf("User registered successfully: %v", insertedID.Hex())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.RegisterResponse{
		Message: "User registered successfully",
		ID:      insertedID.Hex(),
	})
}

// LoginUser 處理使用者登入請求
func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		sendJSONError(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if credentials.Email == "" || credentials.Password == "" {
		sendJSONError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// --- 使用 UserStore 介面進行操作 ---
	user, err := h.UserStore.FindUserByEmail(ctx, credentials.Email)
	if err != nil {
		// 根據錯誤類型回傳不同訊息
		if err == mongo.ErrNoDocuments {
			sendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			log.Printf("Error finding user by email: %v", err)
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		sendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := utils.GenerateJWT(user.ID, user.Username, h.Cfg.JWTSecret)
	if err != nil {
		log.Printf("Error generating JWT token for user %s: %v", user.ID.Hex(), err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	log.Printf("User logged in successfully: %s", user.Email)

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true, // true表示JS無法讀取
		Secure:   true, //只在 HTTPS 連線下傳送
		SameSite: http.SameSiteStrictMode, // 當請求完全來自自己的網站時，瀏覽器才會帶上這個cookie
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.LoginResponse{
		Message:  "Login successful",
		ID:       user.ID.Hex(),
		Username: user.Username,
		// Token:    token,
	})
}

// GetAllUsers 處理獲取所有使用者列表的請求
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	usersCollection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 查找所有使用者
	cursor, err := usersCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Error finding all users: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		log.Printf("Error decoding users: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 創建一個只包含公開資訊的 PublicUser 列表
	var publicUsers []models.PublicUser
	for _, user := range users {
		publicUsers = append(publicUsers, models.PublicUser{
			ID:       user.ID.Hex(),
			Username: user.Username,
			Email:    user.Email, // 考慮是否要暴露 Email，如果不需要可以移除
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(publicUsers); err != nil {
		log.Printf("Error encoding public users: %v", err)
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
	}
}

var googleOAuthConfig *oauth2.Config
var oauthStateString string

// 初始化 Google OAuth 設定
func InitializeOAuthGoogle() {
	cfg := config.LoadConfig()
	googleOAuthConfig = &oauth2.Config{
		RedirectURL:  cfg.GoogleRedirectURL,
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

// HandleGoogleLogin 處理導向 Google 登入頁面的請求
func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// 產生一個隨機 state 字串，用於 CSRF 保護
	b := make([]byte, 16)
	rand.Read(b)
	oauthStateString = base64.URLEncoding.EncodeToString(b)

	// 將 state 存入 cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "oauthstate",
		Value:   oauthStateString,
		Expires: time.Now().Add(10 * time.Minute),
	})

	url := googleOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback 處理 Google 重新導向回來的請求
func HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// 檢查 state 是否相符
	oauthState, _ := r.Cookie("oauthstate")
	if r.FormValue("state") != oauthState.Value {
		log.Println("invalid oauth google state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// 用授權碼交換 token
	token, err := googleOAuthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		log.Printf("code exchange wrong: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// 用 token 獲取使用者資訊
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		log.Printf("failed getting user info: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("failed read user info: %s\n", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	json.Unmarshal(contents, &googleUser)

	usersCollection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 檢查使用者是否存在
	var user models.User
	err = usersCollection.FindOne(ctx, bson.M{"googleId": googleUser.ID}).Decode(&user)

	// 如果使用者不存在，就建立新使用者
	if err == mongo.ErrNoDocuments {
		newUser := models.User{
			GoogleID: googleUser.ID,
			Email:    googleUser.Email,
			Username: googleUser.Name, // 直接使用 Google 名稱作為使用者名稱
		}
		res, err := usersCollection.InsertOne(ctx, newUser)
		if err != nil {
			log.Printf("Error inserting new Google user: %v", err)
			http.Redirect(w, r, "/auth?error=registration_failed", http.StatusTemporaryRedirect)
			return
		}
		user.ID = res.InsertedID.(primitive.ObjectID)
		user.Username = newUser.Username
		user.Email = newUser.Email
	} else if err != nil {
		log.Printf("Error finding user by Google ID: %v", err)
		http.Redirect(w, r, "/auth?error=db_error", http.StatusTemporaryRedirect)
		return
	}

	// 產生 JWT
	cfg := config.LoadConfig()
	jwtToken, err := utils.GenerateJWT(user.ID, user.Username, cfg.JWTSecret)
	if err != nil {
		log.Printf("Error generating JWT for Google user: %v", err)
		http.Redirect(w, r, "/auth?error=token_generation_failed", http.StatusTemporaryRedirect)
		return
	}

	// 將 JWT 存入 cookie 並導回前端首頁
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwtToken,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true, // 建議設為 true 增加安全性
		Secure: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name: "user_info",
		Value: fmt.Sprintf(`{"id":"%s","username":"%s"}`, user.ID.Hex(), url.QueryEscape(user.Username)),
		Path: "/",
		Expires: time.Now().Add(time.Hour * 24),
	})

	http.Redirect(w, r, "http://localhost:5173/home", http.StatusTemporaryRedirect)
}

// LogoutUser 處理使用者登出請求
func (h *AuthHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	// 1. 設定一個 http.Cookie 物件，其名稱與登入時設定的相同 ("token")
	// 2. 將其 MaxAge 設為 -1，這是一個標準指令，告訴瀏覽器立即刪除這個 cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		Value:  "", // 值可以為空
		Path:   "/",
		MaxAge: -1, // 【核心】立即過期
	})

	// 同時也刪除 user_info cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "user_info",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}
