# Load Test Setup

這個資料夾先提供「最小可行場景」初始化工具，目標是準備：

- 20 個可登入的測試使用者
- 1 個所有人都已加入的聊天室
- 一份包含 `roomId` 與所有使用者 `token` 的輸出檔

## 為什麼先做這一步

你的專案真正的送訊息路徑是 WebSocket，不是 HTTP `POST /messages`。

所以壓測需要拆成兩段：

1. 用 HTTP API 準備測試資料
2. 用 WebSocket 工具模擬大量已登入使用者送訊息

`vegeta` 很適合第 1 段，例如登入、建房、查詢聊天室這些 HTTP 行為。
真正的高頻送訊息則比較適合用現有的 [backend/ws-load.js](/E:/coding/GO/Multi-person%20chat%202.0/backend/ws-load.js) 或其他 WebSocket 壓測工具。

## 快速開始

先確定後端已啟動於 `http://localhost:8080`，然後在專案根目錄執行：

```powershell
.\loadtest\setup-minimal-scenario.ps1
```

如果你想自訂人數或 API 位址：

```powershell
.\loadtest\setup-minimal-scenario.ps1 -BaseUrl "http://localhost:8080" -UserCount 20 -Password "LoadTest!123"
```

完成後會產生：

`loadtest/minimal-scenario.json`

檔案內容包含：

- `room.id`
- `room.name`
- `users[].id`
- `users[].email`
- `users[].username`
- `users[].token`

## 這個場景模擬的是什麼

這是在模擬：

- 20 個已登入使用者
- 全部都在同一個熱門聊天室
- 之後可以同時連進 `/ws`
- 並對同一個 `roomId` 發送訊息

這是找「單一熱門聊天室」瓶頸的很好起點。

## 後續怎麼接壓測

下一步通常會做兩種壓測：

1. HTTP 壓測
用 `vegeta` 測登入、查詢聊天室、查歷史訊息等 REST API。

2. WebSocket 壓測
用 `k6` 或其他 WebSocket 工具測真正的送訊息高併發。

## 目前要注意的限制

- 後端登入會把 JWT 放在 `token` cookie，而不是放在 JSON response body。
- 這個腳本會從 `Set-Cookie` 直接抽出 token，後續可手動帶入 `Cookie: token=...`。
- 後端目前有「同一使用者只允許一條 WebSocket 連線」的邏輯，所以做併發時必須使用多個不同 token。

## Login API 壓測

如果你要測 `POST /login`，可以先產生 `vegeta` 用的 targets：

```powershell
powershell -ExecutionPolicy Bypass -File ".\loadtest\generate-login-targets.ps1" -BaseUrl "http://localhost:18080" -UserCount 100
```

這會產生：

`loadtest/login-targets.jsonl`

接著用 `vegeta` 在 5 秒內打 100 個不同使用者登入：

```powershell
vegeta attack -format=json -rate=20/s -duration=5s -targets=".\loadtest\login-targets.jsonl" | vegeta report
```

如果你想保留二進位結果方便之後再分析：

```powershell
vegeta attack -format=json -rate=20/s -duration=5s -targets=".\loadtest\login-targets.jsonl" > .\loadtest\login-results.bin
vegeta report .\loadtest\login-results.bin
```
