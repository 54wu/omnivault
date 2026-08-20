#requires -Version 5.1
<#
.SYNOPSIS
    OmniVault / 万象档案袋 - 唯一入口（首次使用 & 启动）
.DESCRIPTION
    这是本程序的唯一入口（首次使用 & 启动）。
    它会自动判断并完成：
      1. 目录里已有 OmniVault.exe  → 无需 Go，直接进入“启动”流程
      2. 目录里没有 exe             → 自动从源码编译（需已安装 Go）
      3. 读取/校验 secret.key 路径（已记住的用 DPAPI，否则交互输入）
      4. 记住该路径并创建桌面快捷方式
      5. 启动 OmniVault.exe（WebView2 界面）

    日常使用：直接双击本文件即可。若被系统执行策略拦截，
    请右键本文件 → “使用 PowerShell 运行”。
.USAGE
    首次使用.ps1                    # 启动（无 exe 时自动构建）
    首次使用.ps1 -NoStart           # 只构建+配置，不启动
    首次使用.ps1 -SkipIcon          # 构建时跳过图标资源，仅生成裸 exe
    首次使用.ps1 -VaultDir <路径>    # 设置 vault.db 的读取地址（目录），会记住供重复使用
    首次使用.ps1 -KeyPath <路径>     # 设置 secret.key 的读取地址（文件），会记住供重复使用
.NOTES
    本脚本不把 key 或路径写入 vault 文件夹；生成的 exe 与本脚本同目录。
    本脚本会自动感知全新环境：若 ~/.omnivault（或 VAULT_DIR）还没有 vault.db，
    则视为全新，直接启动 exe 进入首次创建；否则按“已有环境”记住 key 路径后启动。
    若你想全新创建但机器上已有旧库，请先清空 ~\.omnivault，或用 VAULT_DIR 隔离到空目录。
#>

param(
    [switch]$SkipIcon,   # 跳过图标资源嵌入（仅生成裸 exe）
    [switch]$NoStart,    # 只构建+配置，不自动启动
    [string]$VaultDir,   # 设置 vault.db 的读取地址（目录）；会被记住，供重复使用
    [string]$KeyPath     # 设置 secret.key 的读取地址（文件）；会被记住，供重复使用
)

# 让双击/运行遇错时停留在窗口，而不是一闪而过。
# 主流程包在 try/catch 里展示错误；顶部设置 BOM 之外的中文输出用 UTF-8。
$ErrorActionPreference = 'Stop'

function Write-Step([string]$msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Pause-Exit([int]$code = 1) {
    Write-Host ''
    # 不再显示提示语，也不要求按键：窗口保持打开方便复制，由用户手动关闭。
    while ($true) { Start-Sleep -Milliseconds 500 }
}

# vault.db 读取地址持久化文件（本脚本同目录；只存目录路径，不含密钥）。
$cfgFile = Join-Path $PSScriptRoot 'omnivault.config'

# 读取上次记住的 vault.db 读取目录；没有则返回 $null。
function Read-SavedVaultDir {
    if (Test-Path -LiteralPath $cfgFile) {
        try { return ([System.IO.File]::ReadAllText($cfgFile)).Trim() } catch {}
    }
    return $null
}

# 记住 vault.db 读取目录（无 BOM 写入，避免路径前混入不可见字符）。
function Save-VaultDir([string]$dir) {
    if (-not $dir) { return }
    try {
        [System.IO.File]::WriteAllText($cfgFile, $dir.Trim(), (New-Object System.Text.UTF8Encoding($false)))
    } catch {
        Write-Host "警告：未能保存 vault.db 路径配置：$_" -ForegroundColor Yellow
    }
}

# 把绝对路径转成“同级相对地址”：位于脚本目录下时显示为 .\…（如 .\personal data），
# 其余情况原样返回。用于展示与写回配置文件的一致、可读的相对地址。
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

# 解析 vault.db 读取地址（配置文件里可能是 .\personal data 等相对同级地址），
# 统一转为绝对路径返回；无法解析返回 $null。
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

# 当本目录没有现成 exe 时，优先从 GitHub Releases 下载 windows/amd64 预编译版；
# 成功返回 $true，失败返回 $false（调用方再回退到源码编译）。
function Get-ReleaseExe {
    $exePath = Join-Path $PSScriptRoot 'OmniVault.exe'
    try {
        Write-Step '未找到现成 OmniVault.exe，尝试从 GitHub 下载预编译版 ...'
        $headers = @{ 'User-Agent' = 'OmniVault-launcher' }
        $rel = Invoke-RestMethod -Uri 'https://api.github.com/repos/54wu/omnivault/releases/latest' -Headers $headers -TimeoutSec 20
        $asset = @($rel.assets) | Where-Object { $_.name -match 'windows.*amd64.*\.zip$' } | Select-Object -First 1
        if (-not $asset) { $asset = @($rel.assets) | Where-Object { $_.name -match 'windows.*\.zip$' } | Select-Object -First 1 }
        if (-not $asset) { throw '该版本中没有 Windows 发布包' }

        $zip = Join-Path $PSScriptRoot 'omnivault-latest.tmp.zip'
        $tmp = Join-Path $PSScriptRoot '.omnivault-dl-tmp'
        Write-Step ("正在下载 " + $asset.name + " ...")
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip -UseBasicParsing -TimeoutSec 120
        if (Test-Path $tmp) { Remove-Item -LiteralPath $tmp -Recurse -Force }
        Expand-Archive -LiteralPath $zip -DestinationPath $tmp -Force
        $exeIn = Get-ChildItem $tmp -Recurse -Filter *.exe | Where-Object { $_.Name -match 'omnivault' } | Select-Object -First 1
        if (-not $exeIn) { $exeIn = Get-ChildItem $tmp -Recurse -Filter *.exe | Select-Object -First 1 }
        if (-not $exeIn) { throw '压缩包内未找到可执行文件' }
        Copy-Item -LiteralPath $exeIn.FullName -Destination $exePath -Force
        Write-Step "已下载预编译版：$exePath"
        return $true
    } catch {
        Write-Host "[GitHub 下载失败] $($_.Exception.Message)" -ForegroundColor Yellow
        return $false
    } finally {
        Remove-Item (Join-Path $PSScriptRoot 'omnivault-latest.tmp.zip') -Force -ErrorAction SilentlyContinue
        Remove-Item (Join-Path $PSScriptRoot '.omnivault-dl-tmp') -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# 已初始化环境下，若密钥与档案库同在一个目录，建议（并引导）把密钥移到安全的外部
# 位置。选择移动时调用 exe 的 key relocate；返回最终使用的密钥路径。
function Offer-KeyRelocate([string]$Key, [string]$VaultDir, [string]$Exe) {
    $keyDir = Split-Path -Parent $Key
    $sameDir = [string]::Equals($keyDir.TrimEnd('\'), $VaultDir.TrimEnd('\'), [System.StringComparison]::OrdinalIgnoreCase)
    if (-not $sameDir) { return $Key }   # 已在外部位置，无需移动

    Write-Host ''
    Write-Host '提示：密钥当前与档案库同在数据目录，建议移到安全的外部位置（U盘/加密盘）。' -ForegroundColor Yellow
    $ans = Read-Host '是否现在移动密钥位置？[y]是 / [回车]否'
    if ($ans -notmatch '^[yY]') {
        Write-Host '（保留当前位置。之后再次运行本脚本，可再次移动密钥。）' -ForegroundColor Yellow
        return $Key
    }

    $target = Read-Host '目标绝对路径（目录或 secret.key 完整路径，例如 E:\keys\secret.key）'
    $target = $target.Trim().Trim('"', "'").Trim()
    if (-not $target) { Write-Host '未输入路径，取消移动。' -ForegroundColor Yellow; return $Key }
    # 若输入的是目录（已存在）或无扩展名（如 E:\keys\老笔记本），自动补上默认文件名 secret.key。
    if ([System.IO.Directory]::Exists($target) -or -not [System.IO.Path]::GetExtension($target)) {
        $target = Join-Path $target 'secret.key'
        Write-Host "  目标为目录/未含文件名，已补全为：$target" -ForegroundColor Cyan
    }
    if ([string]::Equals((Split-Path -Parent $target).TrimEnd('\'), $VaultDir.TrimEnd('\'), [System.StringComparison]::OrdinalIgnoreCase)) {
        Write-Host '目标不能仍在数据目录内，取消移动。' -ForegroundColor Yellow
        return $Key
    }
    Write-Step '正在移动密钥 ...'
    $moveOut = & $Exe 'key' 'relocate' --to $target 2>&1
    $moveOut | ForEach-Object { Write-Host "  $_" }
    $rc = $LASTEXITCODE
    if ($rc -eq 0 -and (Test-Path -LiteralPath $target)) {
        Write-Host "密钥已移动到：$target" -ForegroundColor Green
        return $target
    }
    Write-Host '移动失败，继续使用原密钥路径。' -ForegroundColor Yellow
    return $Key
}

# 在桌面创建 OmniVault 快捷方式（全新环境与已有环境都会调用）
function Create-DesktopShortcut([string]$Exe, [string]$Root) {
    $desktop = [Environment]::GetFolderPath('Desktop')
    if (-not $desktop -or -not (Test-Path $desktop)) {
        Write-Warning '未找到桌面目录，跳过快捷方式'
        return
    }
    try {
        # 快捷方式指向“唯一入口脚本”而非裸 exe：如此数据目录(vault.db)与密钥
        # (secret.key) 都按脚本解析为同级 .\personal data。若直连 exe，快捷方式无法
        # 携带 VAULT_DIR，双击会落回 ~/.omnivault 而误判为“全新 → 新建密码”。
        $ps1 = if ($MyInvocation.MyCommand.Path) { $MyInvocation.MyCommand.Path } else { $PSCommandPath }
        $pwsh = Join-Path $env:WINDIR 'System32\WindowsPowerShell\v1.0\powershell.exe'
        $ws = New-Object -ComObject WScript.Shell
        $lnk = $ws.CreateShortcut((Join-Path $desktop 'OmniVault.lnk'))
        $lnk.TargetPath = $pwsh
        $lnk.Arguments = '-NoProfile -ExecutionPolicy Bypass -File "' + $ps1 + '"'
        $lnk.WorkingDirectory = $Root
        $lnk.IconLocation = "$Exe,0"
        $lnk.Description = 'OmniVault / 万象档案袋'
        $lnk.WindowStyle = 7
        $lnk.Save()
        Write-Step '已在桌面创建 OmniVault 快捷方式（指向唯一入口脚本，行为与直接运行脚本一致）'
    } catch { Write-Warning "创建桌面快捷方式失败：$_" }
}

try {
    # 让 exe 的 UTF-8 中文输出在控制台正确显示（避免乱码）
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    # 定位本脚本目录（源码根/部署根）
    $root = $PSScriptRoot
    $exe = Join-Path $root 'OmniVault.exe'
    $goWinres = Join-Path $env:USERPROFILE 'go\bin\go-winres.exe'
    $cmdMain = Join-Path $root 'cmd\omnivault'

    Write-Step 'OmniVault / 万象档案袋 唯一入口'
    Write-Host '  用法：直接双击本文件即启动程序；目录里没有 exe 时自动下载预编译版（失败则从源码编译）。'
    Write-Host '  （若弹出黄色脚本被阻止提示，请右键本文件→"使用 PowerShell 运行"）'
    Write-Host ''

    # 判断 exe 是否存在：有则直接用；没有则优先从 GitHub Releases 下载预编译版，
    # 下载失败则回退到源码编译（需 Go）。
    $needBuild = -not (Test-Path $exe)
    if ($needBuild) {
        if (Get-ReleaseExe) { Write-Step "已通过 GitHub 获取预编译版：$exe" }
        # 无论下载成功与否，都按"此刻 exe 是否真的存在"来决定是否还需编译
        $needBuild = -not (Test-Path $exe)
    } else {
        Write-Step "检测到现成的 OmniVault.exe，无需编译，直接进入启动流程。"
    }

    # ---------- 1. 编译前提：仅当最终仍需编译时才要求 Go ----------
    if ($needBuild) {
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
            Write-Host '[错误] 未找到 go 命令，且未能从 GitHub 下载预编译版。' -ForegroundColor Red
            Write-Host '需要从源码编译，因此必须安装 Go。'
            Write-Host '请安装 Go：https://go.dev/dl/（装完重启后再运行本脚本）。'
            Write-Host ''
            Write-Host '如果只是想直接用现成程序：'
            Write-Host '  ① 手动把 OmniVault.exe 放进本目录后重跑；或'
            Write-Host '  ② 安装 Go 后由本脚本自动从源码编译。'
            Pause-Exit 1
        }
        # 确认本目录是源码根（含 cmd\omnivault）—— 编译时需要
        if (-not (Test-Path $cmdMain)) {
            Write-Host "[错误] 未找到 cmd\omnivault 源码目录。" -ForegroundColor Red
            Write-Host "请把本脚本放到源码根目录（与 go.mod 同一层）。当前：$root"
            Pause-Exit 1
        }
    }

    # ---------- 2. 编译部分（仅有 exe 缺失时） ----------
    if ($needBuild) {
        Write-Step '现有 exe 缺失，将从源码构建 OmniVault.exe ...'
        # go-winres 图标资源
        if (-not $SkipIcon) {
            if (-not (Test-Path $goWinres)) {
                Write-Step '未找到 go-winres，正在安装...'
                go install github.com/tc-hib/go-winres@latest
                if ($LASTEXITCODE -ne 0) { throw 'go-winres 安装失败（请检查网络后重试）' }
            }
            Write-Step '生成 Windows 资源（图标/版本/清单）...'
            Push-Location $root
            try {
                & $goWinres make --in build/winres.json --arch amd64 --out cmd/omnivault/rsrc
                if ($LASTEXITCODE -ne 0) { throw 'go-winres make 失败' }
            } finally { Pop-Location }
        } else {
            Write-Step '跳过图标资源嵌入（-SkipIcon）'
        }

        # 编译
        Write-Step '编译 OmniVault.exe ...'
        Push-Location $root
        try {
            go build -o $exe ./cmd/omnivault
            if ($LASTEXITCODE -ne 0) { throw 'go build 失败，请阅读上方输出排查（常见：依赖下载失败需联网）' }
        } finally { Pop-Location }
        Write-Step "已生成：$exe"
    }

    # ---------- 3.5 自动感知全新环境 ----------
    # 数据目录默认放在本程序文件夹下的 "personal data"，不再使用 ~\.omnivault。
    # 也可用 -VaultDir 覆盖到任意目录（或 .\相对目录）。设为进程级环境变量给子进程。
    # vault.db 读取地址：以“本脚本自身目录”为基准。默认取同级的 personal data
    # （即先确定 ps1 自身地址，再找其同级的 personal data），不继承外部 VAULT_DIR。
    # 仅当显式传 -VaultDir，或上次记住过自定义目录（非默认）时才覆盖。
    $vaultDir = $null
    if ($VaultDir) {
        $vaultDir = Expand-VaultDir $VaultDir
        if (-not $vaultDir) { $vaultDir = [System.IO.Path]::GetFullPath($VaultDir.Trim().Trim('"', "'")) }
    }
    if (-not $vaultDir) {
        # 只有“相对同级”的自定义目录（.\xxx）才会被复用；外来绝对路径一律忽略，
        # 从而始终锚定到 ps1 自身目录的同级默认。
        $cached = Read-SavedVaultDir
        if ($cached -and $cached.StartsWith('.\')) { $vaultDir = Expand-VaultDir $cached }
    }
    if (-not $vaultDir) {
        $vaultDir = Join-Path $root 'personal data'
    }
    $vaultDir = [System.IO.Path]::GetFullPath($vaultDir)
    # 写回配置文件与展示时使用同级相对地址（如 .\personal data），运行时仍用绝对路径。
    $vaultDirDisplay = To-RelativeVaultDir $vaultDir
    Save-VaultDir $vaultDirDisplay
    $env:VAULT_DIR = $vaultDir
    $freshEnv = -not (Test-Path (Join-Path $vaultDir 'vault.db'))

    # 若你本意是“全新创建”，但该目录已存在档案库（vault.db），脚本会按“已有环境”
    # 处理并直接启动旧库。想真正全新测试，请二选一：
    #   ① 彻底清空数据目录：Remove-Item '.\personal data' -Recurse -Force
    #   ② 隔离测试：先设环境变量 VAULT_DIR 指向一个空目录（如 .\omnitest），再运行本脚本。
    if (-not $freshEnv) {
        Write-Host ''
        Write-Host '检测到已存在档案库（非全新环境）。' -ForegroundColor Yellow
        Write-Host "数据目录（同级相对地址 $vaultDirDisplay，解析后为 $vaultDir）：" 
        Write-Host "  档案库(vault.db)读取地址：$(Join-Path $vaultDirDisplay 'vault.db')"
        Write-Host "  密钥(secret.key)读取地址：$(Join-Path $vaultDirDisplay 'secret.key')"
        Write-Host '若你想全新创建，请清空上面的数据目录，或设置独立的 VAULT_DIR（详见脚本顶部说明）。' -ForegroundColor Yellow
        Write-Host ''
    }

    if ($freshEnv) {
        Write-Step '检测到全新环境（尚未初始化档案库），直接进入首次创建流程。'
        Write-Host '  程序会要求你设置初始密码，并自动生成一把全新的 secret.key。'
        Write-Host '  请在界面上记下这把密钥，并备份到安全位置（见 docs/usage.md）。'
        Write-Host ''
        if (-not $NoStart) {
            # -WindowStyle Hidden：exe 是控制台程序，直接 Start-Process 会多弹一个
            # 控制台窗口，这里隐藏，避免“英文版自动运行”的错觉，只保留 WebView2 主界面。
            Start-Process -FilePath $exe -WorkingDirectory $root -WindowStyle Hidden | Out-Null
            Write-Step 'OmniVault 已启动（独立运行，脚本无需等待其关闭）。'
            Write-Host '请在打开的窗口中新建一个档案：设置初始密码，并抄下界面展示的 secret.key 备份到安全位置。' -ForegroundColor Yellow
            Write-Host '必须完成这一步，档案库才正式生成，初始化才算完成。' -ForegroundColor Yellow
            Write-Host '完成。' -ForegroundColor Green
        } else {
            Write-Host "（-NoStart 已指定：全新环境无需任何配置，直接运行 $exe 即可。）" -ForegroundColor Yellow
        }
        # 全新环境同样创建桌面快捷方式
        Write-Step '创建桌面快捷方式'
        Create-DesktopShortcut -Exe $exe -Root $root
        Write-Host ''
        Write-Step '数据与密钥位置'
        Write-Host "  档案库(vault.db) : $(Join-Path $vaultDirDisplay 'vault.db')"
        Write-Host "  密钥(secret.key) : $(Join-Path $vaultDirDisplay 'secret.key')"
        Write-Host ''
        Write-Host '提示：如需把密钥移到安全的外部位置（U盘/加密盘），' -ForegroundColor Yellow
        Write-Host '      请关闭程序后，再次运行本脚本，并按提示移动密钥。' -ForegroundColor Yellow
        Write-Host ''
        Pause-Exit 0
    }

    # ---------- 4. 确定 secret.key 读取地址 ----------
    # 优先级：-KeyPath > 环境变量 > DPAPI 记住 > 数据目录内默认 secret.key
    $key = $null
    if ($KeyPath) {
        $KeyPath = $KeyPath.Trim().Trim('"', "'")
        if (Test-Path -LiteralPath $KeyPath -PathType Leaf) {
            $key = $KeyPath
            Write-Step "按 -KeyPath 使用 secret.key 读取地址：$key"
        } else {
            Write-Host "警告：-KeyPath 指向的文件不存在（$KeyPath），改用其他方式定位密钥。" -ForegroundColor Yellow
        }
    }
    if (-not $key -and $env:OVAULT_KEY_PATH) {
        $key = $env:OVAULT_KEY_PATH
    } elseif (-not $key) {
        try {
            $where = & $exe 'key' 'where' 2>$null
            if ($LASTEXITCODE -eq 0 -and $where -and $where -notmatch '找不到') {
                $key = $where
            }
        } catch { $key = $null }
    }

    # 校验最终密钥读取地址是否真实存在。若记住了但文件已被删除/移动（如陈旧的
    # 凭据管理器记录），先清除该记录，再走下方回收判断，避免 key remember 报错闪退。
    if ($key -and -not (Test-Path -LiteralPath $key -PathType Leaf)) {
        Write-Host "警告：密钥读取地址已失效（$key），正在清除该记住记录。" -ForegroundColor Yellow
        & $exe 'key' 'forget' 2>$null | Out-Null
        $key = $null
    }

    # 数据目录里若已有默认 secret.key（首跑时程序会在此生成），直接使用，不强制填写。
    $legacyKey = Join-Path $vaultDir 'secret.key'
    if (-not $key -and (Test-Path -LiteralPath $legacyKey -PathType Leaf)) {
        $key = $legacyKey
        Write-Step "数据目录已包含 secret.key，直接使用读取地址：$(Join-Path $vaultDirDisplay 'secret.key')"
        & $exe 'key' 'remember' $key 2>$null | Out-Null
    }

    if (-not $key) {
        # 尚未定位到密钥：提醒到位但不强制。输入路径即设置并记住；直接回车可跳过，
        # 留待之后再次运行本脚本设置，绝不阻塞首次/多次启动。
        Write-Step '尚未找到 secret.key。'
        Write-Host '  secret.key 是加密密钥，请保存在安全位置（U 盘/私人加密目录，与程序分开）。'
        Write-Host '  可现在就告知路径，也可直接回车跳过，之后再运行本脚本设置。'
        Write-Host ''
        $input = (Read-Host '输入 secret.key 完整路径（直接回车跳过）').Trim()
        if ($input) {
            if (-not (Test-Path -LiteralPath $input -PathType Leaf)) {
                Write-Host "跳过：路径不存在或无权限访问（$input），本次不启动。之后可再次运行本脚本设置。" -ForegroundColor Yellow
                Pause-Exit 0
            }
            $key = $input
        } else {
            Write-Host '已跳过密钥设置。需要启动时，请再次运行本脚本并按要求提供 secret.key 路径。' -ForegroundColor Yellow
            Pause-Exit 0
        }
    }

    # ---------- 5. 记住路径 + 桌面快捷方式 ----------
    Write-Step '记住密钥路径（Windows 凭据管理器）'
    & $exe 'key' 'remember' $key 2>$null | Out-Null

    Write-Step '创建桌面快捷方式'
    Create-DesktopShortcut -Exe $exe -Root $root

    # 已初始化环境：密钥若与库同目录，询问是否移到安全外部位置
    $key = Offer-KeyRelocate -Key $key -VaultDir $vaultDir -Exe $exe

    # ---------- 6. 启动 ----------
    Write-Step "使用密钥：$key"
    if (-not $NoStart) {
        Write-Step '启动 OmniVault ...'
        $env:OVAULT_KEY_PATH = $key
        & $exe
        if ($LASTEXITCODE -ne 0) {
            Write-Host 'OmniVault 已退出。' -ForegroundColor Yellow
        }
    }

    Write-Host ''
    Write-Step '数据与密钥位置（本脚本可重复设置）'
    Write-Host "  档案库(vault.db)读取地址 : $(Join-Path $vaultDirDisplay 'vault.db')"
    Write-Host "  密钥(secret.key)读取地址  : $key"
    Write-Host ''
    Write-Host '如需修改读取地址，再次运行本脚本并指定参数即可：' -ForegroundColor Yellow
    Write-Host '  首次使用.ps1 -VaultDir <vault.db所在目录>      # 修改 vault.db 读取地址' -ForegroundColor Yellow
    Write-Host '  首次使用.ps1 -KeyPath <secret.key完整路径>     # 修改 secret.key 读取地址' -ForegroundColor Yellow
    Write-Host ''
    Write-Host '完成。' -ForegroundColor Green
    Pause-Exit 0
}
catch {
    Write-Host ''
    Write-Host '[错误] 脚本执行失败：' -ForegroundColor Red
    Write-Host "  $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ''
    Write-Host '如果是因为没有安装 Go，请先到 https://go.dev/dl/ 安装，然后重试。'
    Write-Host '如果提示缺少源码目录，请把本脚本放在源码根目录重新运行。'
    Pause-Exit 1
}