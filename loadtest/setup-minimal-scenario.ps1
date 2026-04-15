param(
    [string]$BaseUrl = "http://localhost:8080",
    [int]$UserCount = 20,
    [string]$Password = "LoadTest!123",
    [string]$Prefix = "loadtest",
    [string]$OutputPath = ".\loadtest\minimal-scenario.json"
)

$ErrorActionPreference = "Stop"

function ConvertTo-JsonBody {
    param([hashtable]$Payload)
    return ($Payload | ConvertTo-Json -Depth 10)
}

function Invoke-JsonRequest {
    param(
        [string]$Method,
        [string]$Url,
        [hashtable]$Body,
        [hashtable]$Headers,
        [Microsoft.PowerShell.Commands.WebRequestSession]$WebSession
    )

    $invokeParams = @{
        Method      = $Method
        Uri         = $Url
        ContentType = "application/json"
        UseBasicParsing = $true
    }

    if ($Body) {
        $invokeParams.Body = ConvertTo-JsonBody -Payload $Body
    }

    if ($Headers) {
        $invokeParams.Headers = $Headers
    }

    if ($WebSession) {
        $invokeParams.WebSession = $WebSession
    }

    return Invoke-WebRequest @invokeParams
}

function New-TokenSession {
    param(
        [string]$BaseUrl,
        [string]$Token
    )

    $uri = [System.Uri]$BaseUrl
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $cookie = New-Object System.Net.Cookie
    $cookie.Name = "token"
    $cookie.Value = $Token
    $cookie.Path = "/"
    $cookie.Domain = $uri.Host
    $cookie.Secure = $false
    $session.Cookies.Add($uri, $cookie)

    return $session
}

function Get-TokenFromSetCookie {
    param($Response)

    $setCookieHeaders = @($Response.Headers["Set-Cookie"])
    foreach ($header in $setCookieHeaders) {
        if ($header -match "token=([^;]+)") {
            return $Matches[1]
        }
    }

    throw "Login response did not include a token cookie."
}

function Register-User {
    param(
        [string]$BaseUrl,
        [string]$Email,
        [string]$Username,
        [string]$Password
    )

    $body = @{
        email    = $Email
        username = $Username
        password = $Password
    }

    try {
        $response = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/register" -Body $body
        return ($response.Content | ConvertFrom-Json)
    }
    catch {
        $errorResponse = $_.Exception.Response
        if (-not $errorResponse) {
            throw
        }

        $reader = New-Object System.IO.StreamReader($errorResponse.GetResponseStream())
        $content = $reader.ReadToEnd()
        if ($errorResponse.StatusCode.value__ -eq 409) {
            return $null
        }

        throw "Register failed for $Email. Status=$($errorResponse.StatusCode.value__) Body=$content"
    }
}

function Login-User {
    param(
        [string]$BaseUrl,
        [string]$Email,
        [string]$Password
    )

    $body = @{
        email    = $Email
        password = $Password
    }

    $response = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/login" -Body $body
    $loginData = $response.Content | ConvertFrom-Json
    $token = Get-TokenFromSetCookie -Response $response

    return @{
        id       = $loginData.id
        username = $loginData.username
        token    = $token
    }
}

function New-Room {
    param(
        [string]$BaseUrl,
        [string]$Token,
        [string[]]$ParticipantIds
    )
    $session = New-TokenSession -BaseUrl $BaseUrl -Token $Token

    $body = @{
        participantIds = $ParticipantIds
    }

    $response = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/create-chatrooms" -Body $body -WebSession $session
    return ($response.Content | ConvertFrom-Json)
}

function Add-ParticipantsToRoom {
    param(
        [string]$BaseUrl,
        [string]$Token,
        [string]$RoomId,
        [string[]]$ParticipantIds
    )
    $session = New-TokenSession -BaseUrl $BaseUrl -Token $Token

    $body = @{
        newParticipantIds = $ParticipantIds
    }

    $response = Invoke-JsonRequest -Method "PUT" -Url "$BaseUrl/chatrooms/$RoomId/participants" -Body $body -WebSession $session
    return ($response.Content | ConvertFrom-Json)
}

$outputFullPath = Join-Path (Get-Location) $OutputPath
$outputDirectory = Split-Path -Parent $outputFullPath
if (-not (Test-Path $outputDirectory)) {
    New-Item -ItemType Directory -Path $outputDirectory | Out-Null
}

Write-Host "Creating or reusing $UserCount load test users from $BaseUrl ..."

$users = @()
for ($i = 1; $i -le $UserCount; $i++) {
    $email = "{0}+{1:00}@example.com" -f $Prefix, $i
    $username = "{0}_{1:00}" -f $Prefix, $i

    Register-User -BaseUrl $BaseUrl -Email $email -Username $username -Password $Password | Out-Null
    $loginResult = Login-User -BaseUrl $BaseUrl -Email $email -Password $Password

    $users += [pscustomobject]@{
        index    = $i
        id       = $loginResult.id
        email    = $email
        username = $loginResult.username
        token    = $loginResult.token
    }

    Write-Host ("[{0}/{1}] ready: {2}" -f $i, $UserCount, $email)
}

$owner = $users[0]
$room = New-Room -BaseUrl $BaseUrl -Token $owner.token -ParticipantIds @($owner.id)

if ($users.Count -gt 1) {
    $remainingIds = @($users | Select-Object -Skip 1 | ForEach-Object { $_.id })
    $room = Add-ParticipantsToRoom -BaseUrl $BaseUrl -Token $owner.token -RoomId $room.id -ParticipantIds $remainingIds
}

$result = [pscustomobject]@{
    createdAt = (Get-Date).ToString("o")
    baseUrl   = $BaseUrl
    room = [pscustomobject]@{
        id           = $room.id
        name         = $room.name
        participantCount = $room.participants.Count
    }
    users = $users
}

$result | ConvertTo-Json -Depth 10 | Set-Content -Encoding UTF8 $outputFullPath

Write-Host ""
Write-Host "Minimal scenario is ready."
Write-Host "Room ID: $($result.room.id)"
Write-Host "Saved to: $outputFullPath"
Write-Host "Example WebSocket cookie: token=$($users[0].token)"
