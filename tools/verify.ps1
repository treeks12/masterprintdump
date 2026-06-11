param(
    [switch]$RequireCorpus
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$corpusDir = "C:\Program Files (x86)\paulimaq\ARQUIVOS"

if ($RequireCorpus -or $env:MASTERPRINT_REQUIRE_CORPUS -eq "1") {
    if (-not (Test-Path -LiteralPath $corpusDir)) {
        throw "Required Paulimaq ETQ corpus not found: $corpusDir"
    }
    $env:MASTERPRINT_REQUIRE_CORPUS = "1"
}

Push-Location -LiteralPath $repoRoot
try {
    go test ./... -count=1
    $env:GOARCH = "amd64"
    go build -o "build\masterprint-native.exe" .
} finally {
    Pop-Location
}
