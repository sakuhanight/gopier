# 共通ユーティリティ機能モジュール
# プロジェクト全体で使用される共通機能を提供

# 管理者権限チェック
function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Goコマンドの確認
function Test-GoCommand {
    try {
        $env:PATH += ";C:\Program Files\Go\bin"
        go version | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

# プロジェクトルートディレクトリを取得
function Get-ProjectRoot {
    $scriptPath = $MyInvocation.MyCommand.Path
    if ($scriptPath) {
        # スクリプトからプロジェクトルートを特定
        $currentDir = Split-Path -Parent $scriptPath
        while ($currentDir -and -not (Test-Path (Join-Path $currentDir "go.mod"))) {
            $currentDir = Split-Path -Parent $currentDir
        }
        if ($currentDir) {
            return $currentDir
        }
    }
    
    # 現在のディレクトリからgo.modを探す
    $currentDir = Get-Location
    while ($currentDir -and -not (Test-Path (Join-Path $currentDir "go.mod"))) {
        $currentDir = Split-Path -Parent $currentDir
    }
    return $currentDir
}

# プロジェクトルートに移動
function Set-ProjectRoot {
    $projectRoot = Get-ProjectRoot
    if ($projectRoot -and (Test-Path $projectRoot)) {
        Set-Location $projectRoot
        return $true
    }
    return $false
}

# ファイルサイズを人間が読みやすい形式で表示
function Format-FileSize {
    param(
        [long]$SizeInBytes
    )
    
    $units = @("B", "KB", "MB", "GB", "TB")
    $size = $SizeInBytes
    $unitIndex = 0
    
    while ($size -ge 1024 -and $unitIndex -lt ($units.Length - 1)) {
        $size = $size / 1024
        $unitIndex++
    }
    
    return "$([math]::Round($size, 2)) $($units[$unitIndex])"
}

# 実行時間を計測
function Measure-ExecutionTime {
    param(
        [scriptblock]$ScriptBlock,
        [string]$Description = "処理"
    )
    
    $startTime = Get-Date
    $result = & $ScriptBlock
    $endTime = Get-Date
    $duration = $endTime - $startTime
    
    return @{
        Result = $result
        Duration = $duration
        StartTime = $startTime
        EndTime = $endTime
    }
}

# 環境変数を安全に設定
function Set-EnvironmentVariable {
    param(
        [string]$Name,
        [string]$Value,
        [bool]$Temporary = $true
    )
    
    if ($Temporary) {
        Set-Item -Path "env:$Name" -Value $Value
    } else {
        [Environment]::SetEnvironmentVariable($Name, $Value, [EnvironmentVariableTarget]::User)
    }
}

# 環境変数を安全に取得
function Get-EnvironmentVariable {
    param(
        [string]$Name,
        [string]$DefaultValue = ""
    )
    
    $value = [Environment]::GetEnvironmentVariable($Name)
    if (-not $value) {
        return $DefaultValue
    }
    return $value
}

# ファイルの存在確認と詳細情報取得
function Test-FileWithInfo {
    param(
        [string]$Path,
        [switch]$GetInfo
    )
    
    if (-not (Test-Path $Path)) {
        return $false
    }
    
    if ($GetInfo) {
        $item = Get-Item $Path
        return @{
            Exists = $true
            Name = $item.Name
            FullName = $item.FullName
            Length = $item.Length
            LastWriteTime = $item.LastWriteTime
            IsReadOnly = $item.IsReadOnly
        }
    }
    
    return $true
}

# ディレクトリの存在確認と作成
function Test-DirectoryWithCreate {
    param(
        [string]$Path,
        [switch]$CreateIfNotExists
    )
    
    if (Test-Path $Path) {
        return $true
    }
    
    if ($CreateIfNotExists) {
        try {
            New-Item -ItemType Directory -Path $Path -Force | Out-Null
            return $true
        } catch {
            return $false
        }
    }
    
    return $false
}

# プロセスの実行結果を取得
function Invoke-ProcessWithResult {
    param(
        [string]$FilePath,
        [string[]]$ArgumentList = @(),
        [string]$WorkingDirectory = $null,
        [int]$TimeoutSeconds = 300
    )
    
    $processInfo = New-Object System.Diagnostics.ProcessStartInfo
    $processInfo.FileName = $FilePath
    $processInfo.Arguments = $ArgumentList -join " "
    $processInfo.UseShellExecute = $false
    $processInfo.RedirectStandardOutput = $true
    $processInfo.RedirectStandardError = $true
    $processInfo.CreateNoWindow = $true
    
    if ($WorkingDirectory) {
        $processInfo.WorkingDirectory = $WorkingDirectory
    }
    
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $processInfo
    
    try {
        $process.Start() | Out-Null
        $process.WaitForExit($TimeoutSeconds * 1000)
        
        $output = $process.StandardOutput.ReadToEnd()
        $errorOutput = $process.StandardError.ReadToEnd()
        
        return @{
            ExitCode = $process.ExitCode
            Output = $output
            Error = $errorOutput
            TimedOut = $process.HasExited -eq $false
        }
    } finally {
        if (-not $process.HasExited) {
            $process.Kill()
        }
        $process.Dispose()
    }
}

# 設定ファイルを読み込み
function Read-ConfigFile {
    param(
        [string]$Path,
        [hashtable]$DefaultConfig = @{}
    )
    
    if (-not (Test-Path $Path)) {
        return $DefaultConfig
    }
    
    try {
        $content = Get-Content $Path -Raw -ErrorAction Stop
        $config = $content | ConvertFrom-Json -AsHashtable -ErrorAction Stop
        return $config
    } catch {
        Write-WarningLog "設定ファイルの読み込みに失敗: $($_.Exception.Message)"
        return $DefaultConfig
    }
}

# 設定ファイルを保存
function Save-ConfigFile {
    param(
        [string]$Path,
        [hashtable]$Config
    )
    
    try {
        $configJson = $Config | ConvertTo-Json -Depth 10 -Compress:$false
        $configJson | Out-File -FilePath $Path -Encoding UTF8 -ErrorAction Stop
        return $true
    } catch {
        Write-ErrorLog "設定ファイルの保存に失敗: $($_.Exception.Message)"
        return $false
    }
}

# バージョン情報を取得
function Get-VersionInfo {
    $version = if (git describe --tags --always --dirty 2>$null) { 
        git describe --tags --always --dirty 
    } else { 
        "dev" 
    }
    
    $buildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $goVersion = if (Test-GoCommand) { 
        (go version).Split(" ")[2] 
    } else { 
        "unknown" 
    }
    
    return @{
        Version = $version
        BuildTime = $buildTime
        GoVersion = $goVersion
        Platform = "$env:OS $env:PROCESSOR_ARCHITECTURE"
    }
}

# 初期化
$script:ProjectRoot = Get-ProjectRoot 