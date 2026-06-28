# Dev mode for Windows (PowerShell): one port, hot-reload, no build step.
#   - Go server (air, -tags dev) on :1430, proxies SPA + HMR to Vite :5174.
#   - Open http://localhost:1430
$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

if (-not $env:ENOWX_PORT) { $env:ENOWX_PORT = "1430" }
if (-not $env:ENOWX_VITE_PORT) { $env:ENOWX_VITE_PORT = "5174" }
$env:ENOWX_VITE_URL = "http://127.0.0.1:$($env:ENOWX_VITE_PORT)"

if (-not (Get-Command air -ErrorAction SilentlyContinue)) {
  Write-Host "installing air..."
  go install github.com/air-verse/air@latest
}

if (-not (Test-Path web/node_modules)) {
  Write-Host "installing web deps..."
  Push-Location web; npm install; Pop-Location
}

Write-Host "enowx dev on http://localhost:$($env:ENOWX_PORT) (Go proxies -> Vite :$($env:ENOWX_VITE_PORT))"

$backend = Start-Process air -PassThru -NoNewWindow
$frontend = Start-Process npm -ArgumentList "run","dev" -WorkingDirectory web -PassThru -NoNewWindow
try {
  Wait-Process -Id $backend.Id, $frontend.Id
} finally {
  Stop-Process -Id $backend.Id, $frontend.Id -ErrorAction SilentlyContinue
}
