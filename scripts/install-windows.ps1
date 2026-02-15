param(
    [string]$BinDir = "$HOME\.local\bin"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")

Write-Host "Installing git-cc-ai Go binary..."
Push-Location $repoRoot
try {
    go install ./cmd/git-cc-ai
} finally {
    Pop-Location
}

Write-Host "Installing launchers into $BinDir ..."
New-Item -ItemType Directory -Path $BinDir -Force | Out-Null

Copy-Item (Join-Path $PSScriptRoot "git-ai.cmd") (Join-Path $BinDir "git-ai.cmd") -Force
Copy-Item (Join-Path $PSScriptRoot "git-ai.ps1") (Join-Path $BinDir "git-ai.ps1") -Force

Write-Host ""
Write-Host "Done."
Write-Host "Ensure '$BinDir' is on your PATH, then run: git-ai"
