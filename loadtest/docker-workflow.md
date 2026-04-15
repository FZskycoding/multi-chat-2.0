# Load Test Docker Workflow

這份流程是給「獨立、可刪除」的壓測環境用的。

## 環境內容

- MongoDB: `localhost:27018`
- Redis: `localhost:6380`
- Backend: `http://localhost:18080`

壓測結束後執行 `docker compose down -v`，整組資料會一起清掉。

## 1. 啟動壓測環境

在專案根目錄執行：

```powershell
docker compose -f .\docker-compose.loadtest.yml up -d --build
```

確認 backend 是否正常：

```powershell
Invoke-WebRequest http://localhost:18080/health -UseBasicParsing
```

## 2. 建立測試資料

這一步會建立測試使用者、登入取得 token、建立聊天室，並產生：

`loadtest/minimal-scenario.json`

執行：

```powershell
powershell -ExecutionPolicy Bypass -File ".\loadtest\setup-minimal-scenario.ps1" -BaseUrl "http://localhost:18080"
```

如果之後要更多帳號，例如 100 個：

```powershell
powershell -ExecutionPolicy Bypass -File ".\loadtest\setup-minimal-scenario.ps1" -BaseUrl "http://localhost:18080" -UserCount 100
```

## 3. 跑壓測

先從保守版本開始：

```powershell
docker run --rm -i -v "${PWD}:/work" -w /work grafana/k6 run --env VUS=20 --env MESSAGE_INTERVAL_MS=1000 --env BASE_URL=http://host.docker.internal:18080 /work/backend/ws-load.js
```

如果要提高壓力：

```powershell
docker run --rm -i -v "${PWD}:/work" -w /work grafana/k6 run --env VUS=20 --env MESSAGE_INTERVAL_MS=500 --env BASE_URL=http://host.docker.internal:18080 /work/backend/ws-load.js
```

## 4. 刪除環境

```powershell
docker compose -f .\docker-compose.loadtest.yml down -v
```

這會刪除：

- backend 容器
- mongo 容器
- redis 容器
- 對應 volume

## 補充

- `loadtest/minimal-scenario.json` 是本機檔案，不會隨容器刪除；不需要時可手動刪掉。
- 這份流程的前置建立時間不會算進 k6 結果，但建議建完資料後先確認 backend 穩定再開始壓測。
