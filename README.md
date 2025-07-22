# GoChat - 即時多人聊天系統

GoChat 是一個以 Go 後端 + React 前端 所打造的 即時多人聊天系統，支援使用者註冊、登入、一對一及群組聊天室功能，並透過 WebSocket 提供穩定高效的即時訊息體驗。

## 功能特色 (Features)
👤使用者認證:
1.使用者註冊與登入
2.基於 JWT 的安全驗證機制

🧑‍🤝‍🧑 使用者與聊天室管理:
1.查看所有已註冊使用者
2.建立聊天室（一對一或群組）
3.顯示使用者已加入的聊天室列表
4.聊天室成員邀請、退出、動態命名更新

💬 即時訊息系統:
1.WebSocket 即時通訊
2.多人聊天室訊息廣播
3.系統通知（使用者加入/離開）

🕓 訊息歷史與 TTL 管理:
1.讀取聊天室歷史訊息
2.訊息 TTL（自動清除過期訊息）

## 技術架構 (Tech Stack)
🔙 Backend (Go):
1.語言：Go
2.框架：Gorilla Mux
3.WebSocket：Gorilla WebSocket
4.資料庫：MongoDB（使用 go.mongodb.org/mongo-driver）
5.JWT 認證：github.com/golang-jwt/jwt/v5
6.密碼加密：golang.org/x/crypto/bcrypt
7.CORS：github.com/rs/cors
8.環境管理：joho/godotenv

🔜 Frontend (React):
1.工具鏈：Vite
2.語言：TypeScript
3.UI 框架：Mantine
4.圖示庫：Tabler Icons
5.路由管理：React Router DOM
6.狀態管理：React Hooks（useState, useEffect, useRef, useCallback）
7.通知功能：Mantine Notifications

## 架構設計 (Architecture)
採用 前後端分離架構：
1.RESTful API 處理註冊、登入、聊天室等資源管理
2.WebSocket 處理即時訊息傳輸與聊天室廣播
3.前端為 SPA 單頁應用，負責 UI 顯示與互動

# 安裝與執行 (Installation & Running)

## 📌 先決條件
1.Go 1.24+
2.Node.js（建議使用 LTS）
3.MongoDB（本地或雲端）

## 🔙後端(Backend)
git clone https://github.com/FZskycoding/multi-chat-2.0.git
cd go-chat/backend
建立 .env 檔案

安裝依賴與啟動：
go mod tidy
go run main.go

## 🔜前端 (Frontend)
cd ../frontend
npm install      # 或 yarn install
npm run dev      # 或 yarn dev

# API 端點 (API Endpoints

## 🔐 認證
1.POST /register：使用者註冊
2.POST /login：使用者登入
3.GET /all-users：獲取所有使用者（需 JWT）

## 💬 聊天室管理
1.POST /creat-chatrooms：建立聊天室
2.GET /user-chatrooms：查詢使用者聊天室
3.PUT /chatrooms/{id}/update：更新聊天室成員/名稱
4.POST /chatrooms/{id}/leave：退出聊天室
5.PUT /chatrooms/{id}/participants：邀請新成員
6.GET /chat-history?roomId={roomId}：查詢歷史訊息

## 🌐 WebSocket
GET /ws?userId={userId}&username={username}：建立 WebSocket 連線

# 🔭 未來功能規劃 (Planned Enhancements)
✅ 已讀 / 未讀訊息狀態
✅ 在線使用者列表
✅ 使用者頭像與暱稱設定
✅ 圖片 / 檔案傳輸支援
✅ 聊天室未讀數量顯示
✅ 桌面通知提醒
✅ 聊天室搜尋
✅ 更細緻的權限控管
