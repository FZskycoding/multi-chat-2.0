param(
    [string]$BaseUrl = "http://localhost:18080",
    [int]$UserCount = 100,
    [string]$Password = "LoadTest!123",
    [string]$Prefix = "loadtest",
    [string]$OutputPath = ".\loadtest\login-targets.jsonl"
)

$ErrorActionPreference = "Stop"

$outputFullPath = Join-Path (Get-Location) $OutputPath
$outputDirectory = Split-Path -Parent $outputFullPath
if (-not (Test-Path $outputDirectory)) {
    New-Item -ItemType Directory -Path $outputDirectory | Out-Null
}

$utf8NoBom = [System.Text.UTF8Encoding]::new($false)
$writer = New-Object System.IO.StreamWriter($outputFullPath, $false, $utf8NoBom)

try {
    for ($i = 1; $i -le $UserCount; $i++) {
        $email = "{0}+{1:00}@example.com" -f $Prefix, $i
        $bodyJson = @{
            email    = $email
            password = $Password
        } | ConvertTo-Json -Compress
        $bodyBase64 = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($bodyJson))

        $target = [ordered]@{
            method = "POST"
            url    = "$BaseUrl/login"
            header = @{
                "Content-Type" = @("application/json")
            }
            body   = $bodyBase64
        } | ConvertTo-Json -Compress -Depth 5

        $writer.WriteLine($target)
    }
}
finally {
    $writer.Dispose()
}

Write-Host "Generated $UserCount login targets."
Write-Host "Saved to: $outputFullPath"
