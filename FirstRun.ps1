#requires -Version 5.1
<#
.SYNOPSIS
    OmniVault - The single entry point (first-run setup & launcher)
.DESCRIPTION
    This is the program's single entry point (first-run setup & launcher).
    It automatically:
      1. If OmniVault.exe already sits next to this script → no Go needed,
         go straight to the launch flow
      2. If the exe is missing                               → build from source
         automatically (requires Go installed)
      3. Resolve / validate the secret.key path (DPAPI-remembered or prompted)
      4. Remember the path and create a desktop shortcut
      5. Launch OmniVault.exe (WebView2 UI)

    For daily use: just double-click this file. If the OS blocks the script,
    right-click this file and choose "Run with PowerShell".
.USAGE
    FirstRun.ps1               # launch (auto-build when the exe is missing)
    FirstRun.ps1 -NoStart      # build/configure only, do not launch
    FirstRun.ps1 -SkipIcon     # skip icon resource, build a bare exe
    FirstRun.ps1 -VaultDir <path>  # set the vault.db read address (dir), remembered for reuse
    FirstRun.ps1 -KeyPath <path>   # set the secret.key read address (file), remembered for reuse
.NOTES
    This script never writes the key or its path into the vault folder. The
    built exe is placed next to this script.
    It auto-detects a fresh environment: if ~/.omnivault (or VAULT_DIR) has no
    vault.db, it is treated as fresh and the exe is launched into first-run
    setup; otherwise the key path is remembered as an existing environment.
    For a truly fresh setup when a vault already exists, clear ~\.omnivault or
    point VAULT_DIR at an empty directory.
#>

param(
    [switch]$SkipIcon,   # Skip icon resource embedding (build a bare exe)
    [switch]$NoStart,    # Build/configure only, do not auto-launch
    [string]$VaultDir,   # Set the vault.db read address (dir); remembered for reuse
    [string]$KeyPath     # Set the secret.key read address (file); remembered for reuse
)

$ErrorActionPreference = 'Stop'

function Write-Step([string]$msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Pause-Exit([int]$code = 1) {
    Write-Host ''
    # No message and no keypress: window stays open for copying; user closes it manually.
    while ($true) { Start-Sleep -Milliseconds 500 }
}

# Persisted vault.db read address (next to this script; stores a directory path only).
$cfgFile = Join-Path $PSScriptRoot 'omnivault.config'

# Return the previously remembered vault.db read directory, or $null.
function Read-SavedVaultDir {
    if (Test-Path -LiteralPath $cfgFile) {
        try { return ([System.IO.File]::ReadAllText($cfgFile)).Trim() } catch {}
    }
    return $null
}

# Remember the vault.db read directory (written without BOM to avoid invisible leading bytes).
function Save-VaultDir([string]$dir) {
    if (-not $dir) { return }
    try {
        [System.IO.File]::WriteAllText($cfgFile, $dir.Trim(), (New-Object System.Text.UTF8Encoding($false)))
    } catch {
        Write-Host "Warning: could not save vault.db path config: $_" -ForegroundColor Yellow
    }
}

# Convert an absolute path to a same-level relative address (.\personal data) when it lives
# under the script directory; otherwise return it unchanged. Used for display and persistence.
function To-RelativeVaultDir([string]$abs) {
    if (-not $abs) { return $abs }
    $abs = $abs.Trim()
    $rootTrim = $root.TrimEnd('\')
    if ($abs.TrimEnd('\') -ieq $rootTrim) { return '.\' }
    $prefix = $rootTrim + '\'
    if ($abs.ToLowerInvariant().StartsWith($prefix.ToLowerInvariant())) {
        return '.\' + $abs.Substring($prefix.Length)
    }
    return $abs
}

# Resolve a vault.db read address (config may hold a same-level relative address like
# .\personal data) into an absolute path; return $null if it cannot be resolved.
function Expand-VaultDir([string]$saved) {
    if (-not $saved) { return $null }
    $saved = $saved.Trim().Trim('"', "'").Trim()
    if ($saved -eq '.' -or $saved -eq '.\') { return $root }
    if ($saved.StartsWith('.\')) {
        return [System.IO.Path]::GetFullPath((Join-Path $root ($saved.Substring(2))))
    }
    if ([System.IO.Path]::IsPathRooted($saved)) {
        return [System.IO.Path]::GetFullPath($saved)
    }
    return $null
}

# When no exe is present, first try to download the windows/amd64 prebuilt from
# GitHub Releases. Returns $true on success, $false on failure (caller then
# falls back to building from source).
function Get-ReleaseExe {
    $exePath = Join-Path $PSScriptRoot 'OmniVault.exe'
    try {
        Write-Step 'No OmniVault.exe found - trying to download the prebuilt binary from GitHub ...'
        $headers = @{ 'User-Agent' = 'OmniVault-launcher' }
        $rel = Invoke-RestMethod -Uri 'https://api.github.com/repos/54wu/omnivault/releases/latest' -Headers $headers -TimeoutSec 20
        $asset = @($rel.assets) | Where-Object { $_.name -match 'windows.*amd64.*\.zip$' } | Select-Object -First 1
        if (-not $asset) { $asset = @($rel.assets) | Where-Object { $_.name -match 'windows.*\.zip$' } | Select-Object -First 1 }
        if (-not $asset) { throw 'No Windows package in this release' }

        $zip = Join-Path $PSScriptRoot 'omnivault-latest.tmp.zip'
        $tmp = Join-Path $PSScriptRoot '.omnivault-dl-tmp'
        Write-Step ("Downloading " + $asset.name + " ...")
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip -UseBasicParsing -TimeoutSec 120
        if (Test-Path $tmp) { Remove-Item -LiteralPath $tmp -Recurse -Force }
        Expand-Archive -LiteralPath $zip -DestinationPath $tmp -Force
        $exeIn = Get-ChildItem $tmp -Recurse -Filter *.exe | Where-Object { $_.Name -match 'omnivault' } | Select-Object -First 1
        if (-not $exeIn) { $exeIn = Get-ChildItem $tmp -Recurse -Filter *.exe | Select-Object -First 1 }
        if (-not $exeIn) { throw 'No executable inside the archive' }
        Copy-Item -LiteralPath $exeIn.FullName -Destination $exePath -Force
        Write-Step "Downloaded prebuilt binary: $exePath"
        return $true
    } catch {
        Write-Host "[GitHub download failed] $($_.Exception.Message)" -ForegroundColor Yellow
        return $false
    } finally {
        Remove-Item (Join-Path $PSScriptRoot 'omnivault-latest.tmp.zip') -Force -ErrorAction SilentlyContinue
        Remove-Item (Join-Path $PSScriptRoot '.omnivault-dl-tmp') -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# In an already-initialized environment, if the key sits in the same directory as
# the vault, suggest (and guide) moving it somewhere safe. Calls "key relocate"
# when the user agrees; returns the final key path to use.
function Offer-KeyRelocate([string]$Key, [string]$VaultDir, [string]$Exe) {
    $keyDir = Split-Path -Parent $Key
    $sameDir = [string]::Equals($keyDir.TrimEnd('\'), $VaultDir.TrimEnd('\'), [System.StringComparison]::OrdinalIgnoreCase)
    if (-not $sameDir) { return $Key }   # already external, nothing to move

    Write-Host ''
    Write-Host 'Tip: the key is in the same folder as the vault data. It is safer to move it to an external location (USB/encrypted drive).' -ForegroundColor Yellow
    $ans = Read-Host 'Move the key now? [y] Yes / [Enter] No'
    if ($ans -notmatch '^[yY]') {
        Write-Host '(Keeping the current location. Re-running this script later lets you move it.)' -ForegroundColor Yellow
        return $Key
    }

    $target = Read-Host 'Destination absolute path (dir or full secret.key path, e.g. E:\keys\secret.key)'
    $target = $target.Trim().Trim('"', "'").Trim()
    if (-not $target) { Write-Host 'No path given, skipping the move.' -ForegroundColor Yellow; return $Key }
    # If the input looks like a directory (already exists, or has no file extension),
    # append the default file name secret.key.
    if ([System.IO.Directory]::Exists($target) -or -not [System.IO.Path]::GetExtension($target)) {
        $target = Join-Path $target 'secret.key'
        Write-Host "  Target looks like a directory; completed as: $target" -ForegroundColor Cyan
    }
    if ([string]::Equals((Split-Path -Parent $target).TrimEnd('\'), $VaultDir.TrimEnd('\'), [System.StringComparison]::OrdinalIgnoreCase)) {
        Write-Host 'Destination must not be inside the data directory, skipping.' -ForegroundColor Yellow
        return $Key
    }
    Write-Step 'Moving the key ...'
    $moveOut = & $Exe 'key' 'relocate' --to $target 2>&1
    $moveOut | ForEach-Object { Write-Host "  $_" }
    $rc = $LASTEXITCODE
    if ($rc -eq 0 -and (Test-Path -LiteralPath $target)) {
        Write-Host "Key moved to: $target" -ForegroundColor Green
        return $target
    }
    Write-Host 'Move failed, keeping the original key path.' -ForegroundColor Yellow
    return $Key
}

# Create the OmniVault desktop shortcut (called for both fresh and existing environments)
function Create-DesktopShortcut([string]$Exe, [string]$Root) {
    $desktop = [Environment]::GetFolderPath('Desktop')
    if (-not $desktop -or -not (Test-Path $desktop)) {
        Write-Warning 'Desktop folder not found, skipping shortcut'
        return
    }
    try {
        $ws = New-Object -ComObject WScript.Shell
        $lnk = $ws.CreateShortcut((Join-Path $desktop 'OmniVault.lnk'))
        $lnk.TargetPath = $Exe
        $lnk.WorkingDirectory = $Root
        $lnk.IconLocation = "$Exe,0"
        $lnk.Description = 'OmniVault'
        $lnk.Save()
        Write-Step 'Desktop shortcut created'
    } catch { Write-Warning "Failed to create desktop shortcut: $_" }
}

try {
    # Let the exe's UTF-8 output render correctly on the console (avoid mojibake)
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    # Locate this script's directory (source/deploy root)
    $root = $PSScriptRoot
    $exe = Join-Path $root 'OmniVault.exe'
    $goWinres = Join-Path $env:USERPROFILE 'go\bin\go-winres.exe'
    $cmdMain = Join-Path $root 'cmd\omnivault'

    Write-Step 'OmniVault - single entry point'
    Write-Host '  Usage: double-click this file to launch; if no exe it downloads the prebuilt (else builds from source).'
    Write-Host '  (If blocked by the OS, right-click this file > "Run with PowerShell")'
    Write-Host ''

    # Decide whether the exe exists: if it does, just launch. Otherwise try to
    # download the prebuilt from GitHub Releases first, then fall back to building
    # from source (requires Go).
    $needBuild = -not (Test-Path $exe)
    if ($needBuild) {
        if (Get-ReleaseExe) { Write-Step "Got prebuilt binary from GitHub: $exe" }
        # Re-check: only build if the exe still doesn't exist after the download.
        $needBuild = -not (Test-Path $exe)
    } else {
        Write-Step 'Found an existing OmniVault.exe, no compile needed - going straight to launch.'
    }

    # ---------- 1. Build prerequisite: Go is only required if we still must compile ----------
    if ($needBuild) {
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
            Write-Host '[ERROR] Go was not found, and the prebuilt binary could not be downloaded from GitHub.' -ForegroundColor Red
            Write-Host 'Building from source is required, which needs Go.'
            Write-Host 'Install Go at https://go.dev/dl/ then restart and rerun.'
            Write-Host ''
            Write-Host 'To just use a ready-made program:'
            Write-Host '  1) Put OmniVault.exe in this folder and rerun; or'
            Write-Host '  2) Install Go and let this script build it from source.'
            Pause-Exit 1
        }
        # Source-root check only matters when compiling
        if (-not (Test-Path $cmdMain)) {
            Write-Host "[ERROR] Source directory cmd\omnivault not found." -ForegroundColor Red
            Write-Host "Put this script in the source root (same level as go.mod). Current: $root"
            Pause-Exit 1
        }
    }

    # ---------- 2. Build only when the exe is missing ----------
    if ($needBuild) {
        Write-Step 'OmniVault.exe missing - building from source ...'
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

        Write-Step 'Building OmniVault.exe ...'
        Push-Location $root
        try {
            go build -o $exe ./cmd/omnivault
            if ($LASTEXITCODE -ne 0) { throw 'go build failed; see output above (common: dependency download needs network)' }
        } finally { Pop-Location }
        Write-Step "Built: $exe"
    }

    # ---------- 3.5 Auto-detect a fresh environment ----------
    # The data directory defaults to "personal data" under this program folder,
    # no longer ~/.omnivault. Override with VAULT_DIR to any empty directory
    # (e.g. .\omnitest). Set it as a process-level env var so the exe child
    # processes launched below inherit the same directory (key and vault stay
    # together).
    # vault.db read address is anchored to this script's own directory. The default is
    # "personal data" at the same level as the ps1 (first locate the ps1 folder, then
    # its sibling "personal data"), and it does NOT inherit an external VAULT_DIR.
    # Only an explicit -VaultDir, or a previously remembered custom (non-default) dir,
    # overrides it.
    $vaultDir = $null
    if ($VaultDir) {
        $vaultDir = Expand-VaultDir $VaultDir
        if (-not $vaultDir) { $vaultDir = [System.IO.Path]::GetFullPath($VaultDir.Trim().Trim('"', "'")) }
    }
    if (-not $vaultDir) {
        # Only a same-level relative custom dir (.\xxx) is reused; foreign absolute
        # paths are ignored, so we always anchor to the ps1-relative sibling default.
        $cached = Read-SavedVaultDir
        if ($cached -and $cached.StartsWith('.\')) { $vaultDir = Expand-VaultDir $cached }
    }
    if (-not $vaultDir) {
        $vaultDir = Join-Path $root 'personal data'
    }
    $vaultDir = [System.IO.Path]::GetFullPath($vaultDir)
    # Persist and display the same-level relative address (.\personal data); keep the
    # absolute path for the runtime env var so resolution never depends on the CWD.
    $vaultDirDisplay = To-RelativeVaultDir $vaultDir
    Save-VaultDir $vaultDirDisplay
    $env:VAULT_DIR = $vaultDir
    $freshEnv = -not (Test-Path (Join-Path $vaultDir 'vault.db'))

    # If you intend a FRESH setup but a vault already exists in that directory
    # (vault.db present), the script treats it as an EXISTING environment and
    # launches the old vault directly. For a truly fresh test, do one of:
    #   1) Wipe the data dir: Remove-Item '.\personal data' -Recurse -Force
    #   2) Isolate: set VAULT_DIR to an empty dir (e.g. .\omnitest) before running.
    if (-not $freshEnv) {
        Write-Host ''
        Write-Host 'Detected an existing vault (not a fresh environment).' -ForegroundColor Yellow
        Write-Host "Data directory (same-level relative $vaultDirDisplay; resolved to $vaultDir):"
        Write-Host "  vault.db read address : $(Join-Path $vaultDirDisplay 'vault.db')"
        Write-Host "  secret.key read address: $(Join-Path $vaultDirDisplay 'secret.key')"
        Write-Host 'For a fresh setup, clear the data directory above or set a dedicated VAULT_DIR (see note at top).' -ForegroundColor Yellow
        Write-Host ''
    }

    if ($freshEnv) {
        Write-Step 'Fresh environment detected (vault not initialized yet) - going straight to first-run setup.'
        Write-Host '  The app will ask you to set an initial password and generate a brand-new secret.key.'
        Write-Host '  Note this key on screen and back it up somewhere safe (see docs/usage.md).'
        Write-Host ''
        if (-not $NoStart) {
            # -WindowStyle Hidden: the exe is a console program, so a plain Start-Process would
            # pop up an extra console window. Hide it to avoid a confusing second window;
            # only the WebView2 main window remains.
            Start-Process -FilePath $exe -WorkingDirectory $root -WindowStyle Hidden | Out-Null
            Write-Step 'OmniVault launched (runs independently; the script does not wait for it to close).'
            Write-Host 'In the window that opens, create a new vault: set an initial password and write down the secret.key shown on screen for safekeeping.' -ForegroundColor Yellow
            Write-Host 'The vault is only actually created once you finish this step - only then is initialization complete.' -ForegroundColor Yellow
            Write-Host 'Done.' -ForegroundColor Green
        } else {
            Write-Host "(-NoStart specified: fresh environment needs no setup, just run $exe directly.)" -ForegroundColor Yellow
        }
        # Create the desktop shortcut for fresh environments too
        Write-Step 'Creating desktop shortcut'
        Create-DesktopShortcut -Exe $exe -Root $root
        Write-Host ''
        Write-Step 'Data & key locations'
        Write-Host "  Vault database : $(Join-Path $vaultDirDisplay 'vault.db')"
        Write-Host "  Secret key     : $(Join-Path $vaultDirDisplay 'secret.key')"
        Write-Host ''
        Write-Host 'Tip: to move the key to a safe external location (USB/encrypted drive),' -ForegroundColor Yellow
        Write-Host '      close the app, then run this script again and follow the prompts.' -ForegroundColor Yellow
        Write-Host ''
        Pause-Exit 0
    }

    # ---------- 4. Resolve secret.key read address ----------
    # Priority: -KeyPath > env var > DPAPI-remembered > default secret.key in data dir
    $key = $null
    if ($KeyPath) {
        $KeyPath = $KeyPath.Trim().Trim('"', "'")
        if (Test-Path -LiteralPath $KeyPath -PathType Leaf) {
            $key = $KeyPath
            Write-Step "Using secret.key read address from -KeyPath: $key"
        } else {
            Write-Host "Warning: -KeyPath file not found ($KeyPath); falling back to other lookup methods." -ForegroundColor Yellow
        }
    }
    if (-not $key -and $env:OVAULT_KEY_PATH) {
        $key = $env:OVAULT_KEY_PATH
    } elseif (-not $key) {
        try {
            $where = & $exe 'key' 'where' 2>$null
            if ($LASTEXITCODE -eq 0 -and $where -and $where -notmatch 'not found|find|找不到') {
                $key = $where
            }
        } catch { $key = $null }
    }

    # Validate the final key read address really exists. If a remembered path is
    # stale/deleted (e.g. old credential-manager record), clear it and fall back
    # below, so "key remember" does not crash the script.
    if ($key -and -not (Test-Path -LiteralPath $key -PathType Leaf)) {
        Write-Host "Warning: secret.key read address is invalid ($key); clearing the remembered entry." -ForegroundColor Yellow
        & $exe 'key' 'forget' 2>$null | Out-Null
        $key = $null
    }

    # If the data dir already holds the default secret.key (the app generates it there
    # on first run), use it directly without forcing the user to type a path.
    $legacyKey = Join-Path $vaultDir 'secret.key'
    if (-not $key -and (Test-Path -LiteralPath $legacyKey -PathType Leaf)) {
        $key = $legacyKey
        Write-Step "Data dir already has secret.key; using read address: $(Join-Path $vaultDirDisplay 'secret.key')"
        & $exe 'key' 'remember' $key 2>$null | Out-Null
    }

    if (-not $key) {
        # Remind, do not force. Entering a path sets and remembers it; pressing Enter
        # skips and lets you set it on a later run. Never blocks first/repeated launch.
        Write-Step 'No secret.key found yet.'
        Write-Host '  secret.key is your encryption key. Keep it somewhere safe (USB drive / private'
        Write-Host '  encrypted folder, separate from the app and data).'
        Write-Host '  Provide its path now, or press Enter to skip and set it on a later run.'
        Write-Host ''
        $input = (Read-Host 'Full path of secret.key (Enter to skip)').Trim()
        if ($input) {
            if (-not (Test-Path -LiteralPath $input -PathType Leaf)) {
                Write-Host "Skipped: path does not exist or is not accessible ($input); not launching. Rerun later to set it." -ForegroundColor Yellow
                Pause-Exit 0
            }
            $key = $input
        } else {
            Write-Host 'Skipped key setup. To launch, rerun this script and provide the secret.key path.' -ForegroundColor Yellow
            Pause-Exit 0
        }
    }

    # ---------- 5. Remember path + desktop shortcut ----------
    Write-Step 'Remembering key path (Windows Credential Manager)'
    & $exe 'key' 'remember' $key 2>$null | Out-Null

    Write-Step 'Creating desktop shortcut'
    Create-DesktopShortcut -Exe $exe -Root $root

    # Already-initialized: if the key shares the vault folder, ask whether to move it
    $key = Offer-KeyRelocate -Key $key -VaultDir $vaultDir -Exe $exe

    # ---------- 6. Launch ----------
    Write-Step "Using key: $key"
    if (-not $NoStart) {
        Write-Step 'Launching OmniVault ...'
        $env:OVAULT_KEY_PATH = $key
        & $exe
        if ($LASTEXITCODE -ne 0) {
            Write-Host 'OmniVault exited.' -ForegroundColor Yellow
        }
    }

    Write-Host ''
    Write-Step 'Data & key locations (this script can re-set them)'
    Write-Host "  vault.db read address : $(Join-Path $vaultDirDisplay 'vault.db')"
    Write-Host "  secret.key read address: $key"
    Write-Host ''
    Write-Host 'To change the read addresses, rerun with flags:' -ForegroundColor Yellow
    Write-Host '  FirstRun.ps1 -VaultDir <dir containing vault.db>  # change vault.db read address' -ForegroundColor Yellow
    Write-Host '  FirstRun.ps1 -KeyPath <full path to secret.key>  # change secret.key read address' -ForegroundColor Yellow
    Write-Host ''
    Write-Host 'Done.' -ForegroundColor Green
    Pause-Exit 0
}
catch {
    Write-Host ''
    Write-Host '[ERROR] Script failed:' -ForegroundColor Red
    Write-Host "  $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ''
    Write-Host 'If Go is not installed, install it from https://go.dev/dl/ and retry.'
    Pause-Exit 1
}