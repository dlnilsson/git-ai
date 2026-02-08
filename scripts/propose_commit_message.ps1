param(
    [string]$Type = "",
    [string]$Scope = "",
    [string]$Summary = "",
    [string]$Body = "",
    [string]$Footer = "",
    [switch]$Breaking,
    [string]$BreakingDesc = "",
    [int]$MaxSummary = 72
)

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Error "git not found in PATH"
    exit 1
}

$gitRoot = git rev-parse --show-toplevel 2>$null
if (-not $gitRoot) {
    Write-Error "Not inside a git repository"
    exit 1
}

$gopath = (go env GOPATH 2>$null).Trim()
$bin = if ($gopath) { Join-Path $gopath "bin\git-ai.exe" } else { "" }
$localGo = Join-Path $PSScriptRoot "git_ai_commit.go"

Push-Location $gitRoot | Out-Null
try {
    $argsList = @()
    if ($Type) { $argsList += @("-type", $Type) }
    if ($Scope) { $argsList += @("-scope", $Scope) }
    if ($Summary) { $argsList += @("-summary", $Summary) }
    if ($Body) { $argsList += @("-body", $Body) }
    if ($Footer) { $argsList += @("-footer", $Footer) }
    if ($Breaking) { $argsList += "-breaking" }
    if ($BreakingDesc) { $argsList += @("-breaking-desc", $BreakingDesc) }
    if ($MaxSummary) { $argsList += @("-max-summary", $MaxSummary) }

    if ($bin -and (Test-Path $bin)) {
        $message = & $bin @argsList
    } else {
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
            Write-Error "go not found in PATH and git-ai binary missing"
            exit 1
        }
        $message = & go run $localGo @argsList
    }

    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    "Suggested commit message:" | Write-Host
    $message | Write-Host
} finally {
    Pop-Location | Out-Null
}
