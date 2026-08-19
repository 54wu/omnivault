#requires -Version 5.1
<#
.SYNOPSIS
    OmniVault - Windows one-click build
.DESCRIPTION
    Generates the Windows icon/version resources (via go-winres, unless -SkipIcon),
    then compiles the exe to OmniVault.exe at the repo root.
.USAGE
    .\build.ps1                # icon resources + build -> .\OmniVault.exe
    .\build.ps1 -Tests         # build + run all tests with race detector
    .\build.ps1 -SkipIcon      # skip icon resource, build a bare exe
.NOTES
    Requires Go 1.26+. If go-winres is missing it is installed automatically.
    The built exe is written to OmniVault.exe at the repo root (git-ignored).
    For the first-run launcher that also sets up the vault and desktop shortcut,
    see 首次使用.ps1 (Chinese) / FirstRun.ps1 (English).
#>
param(
    [switch]$SkipIcon,   # Skip icon resource embedding (bare exe, no version info)
    [switch]$Tests       # Also run `go test -race ./...` after building
)

$ErrorActionPreference = 'Stop'

$root     = $PSScriptRoot
$exe      = Join-Path $root 'OmniVault.exe'
$goWinres = Join-Path $env:USERPROFILE 'go\bin\go-winres.exe'

function Write-Step([string]$msg) { Write-Host "==> $msg" -ForegroundColor Cyan }

# ---------- 1. Windows resources (icon / version / manifest) ----------
if (-not $SkipIcon) {
    if (-not (Test-Path $goWinres)) {
        Write-Step 'go-winres not found, installing...'
        go install github.com/tc-hib/go-winres@latest
        if ($LASTEXITCODE -ne 0) { throw 'go-winres install failed (check your network and retry)' }
    }
    Write-Step 'Generating Windows resources (icon/version/manifest)...'
    Push-Location $root
    try {
        & $goWinres make --in build/winres.json --arch amd64 --out cmd/omnivault/rsrc
        if ($LASTEXITCODE -ne 0) { throw 'go-winres make failed' }
    } finally { Pop-Location }
} else {
    Write-Step 'Skipping icon resource embedding (-SkipIcon)'
}

# ---------- 2. Build ----------
Write-Step 'Building omnivault.exe ...'
Push-Location $root
try {
    go build -trimpath -ldflags "-s -w" -o $exe ./cmd/omnivault
    if ($LASTEXITCODE -ne 0) { throw 'go build failed; see output above (common: dependency download needs network)' }
} finally { Pop-Location }
Write-Step "Built: $exe"

# ---------- 3. Tests (optional) ----------
if ($Tests) {
    Write-Step 'Running tests (go test -race ./...) ...'
    go test -race ./...
    if ($LASTEXITCODE -ne 0) { throw 'tests failed' }
}

Write-Host ''
Write-Host 'Done.' -ForegroundColor Green
