param(
    [string]$BaseUrl = "http://127.0.0.1:18080",
    [int]$ExpectedClients = 400,
    [bool]$AutoOpen = $true
)

$ErrorActionPreference = "Stop"

$body = @{
    expectedClients = $ExpectedClients
    autoOpen        = $AutoOpen
} | ConvertTo-Json

Invoke-WebRequest `
    -Method POST `
    -Uri "$BaseUrl/loadtest/barrier/reset" `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing
