param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ArgsList
)

$ErrorActionPreference = "Stop"

$tmp = [System.IO.Path]::GetTempFileName()
try {
    $msg = & git-cc-ai @ArgsList 2> $tmp
    $exitCode = $LASTEXITCODE

    $stderr = Get-Content -Path $tmp -Raw
    if ($stderr) {
        [Console]::Error.Write($stderr)
    }

    if ($exitCode -ne 0) {
        exit $exitCode
    }

    $msg | git commit -F - --edit
} catch {
    $stderr = Get-Content -Path $tmp -Raw -ErrorAction SilentlyContinue
    if ($stderr) {
        [Console]::Error.Write($stderr)
    }
    throw
} finally {
    Remove-Item -Path $tmp -ErrorAction SilentlyContinue
}
