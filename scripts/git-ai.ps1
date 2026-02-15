param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ArgsList
)

$ErrorActionPreference = "Stop"

try {
    $msg = & git-cc-ai @ArgsList
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        exit $exitCode
    }

    $messageText = ($msg -join [Environment]::NewLine)
    $messageText | git commit -F - --edit
    exit $LASTEXITCODE
} catch {
    throw
}
