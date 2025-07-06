# 共通ログ機能モジュール
# 統一されたログ出力とエラーハンドリングを提供

# ログレベル
enum LogLevel {
    Debug = 0
    Info = 1
    Warning = 2
    Error = 3
    Fatal = 4
}

# ログ設定
$script:LogConfig = @{
    Level = [LogLevel]::Info
    EnableFileLog = $true
    LogDirectory = "logs"
    EnableColor = $true
    TimestampFormat = "yyyy-MM-dd HH:mm:ss"
}

# ログファイル名を生成
function Get-LogFileName {
    param(
        [string]$Prefix = "gopier"
    )
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    return "$Prefix`_$timestamp.log"
}

# ログディレクトリを確保
function Initialize-LogDirectory {
    if ($LogConfig.EnableFileLog -and -not (Test-Path $LogConfig.LogDirectory)) {
        New-Item -ItemType Directory -Path $LogConfig.LogDirectory -Force | Out-Null
    }
}

# 色付き出力関数
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White",
        [LogLevel]$Level = [LogLevel]::Info
    )
    
    if ($Level -lt $LogConfig.Level) {
        return
    }
    
    $timestamp = Get-Date -Format $LogConfig.TimestampFormat
    $levelText = $Level.ToString().ToUpper()
    $formattedMessage = "[$timestamp] [$levelText] $Message"
    
    if ($LogConfig.EnableColor) {
        Write-Host $formattedMessage -ForegroundColor $Color
    } else {
        Write-Host $formattedMessage
    }
}

# ログ出力関数
function Write-Log {
    param(
        [string]$Message,
        [LogLevel]$Level = [LogLevel]::Info,
        [string]$LogFile = $null
    )
    
    if ($Level -lt $LogConfig.Level) {
        return
    }
    
    $timestamp = Get-Date -Format $LogConfig.TimestampFormat
    $levelText = $Level.ToString().ToUpper()
    $logEntry = "[$timestamp] [$levelText] $Message"
    
    # コンソール出力
    $color = switch ($Level) {
        ([LogLevel]::Debug) { "Gray" }
        ([LogLevel]::Info) { "White" }
        ([LogLevel]::Warning) { "Yellow" }
        ([LogLevel]::Error) { "Red" }
        ([LogLevel]::Fatal) { "Magenta" }
        default { "White" }
    }
    
    Write-ColorOutput $Message $color $Level
    
    # ファイル出力
    if ($LogConfig.EnableFileLog) {
        Initialize-LogDirectory
        
        $logFilePath = if ($LogFile) {
            Join-Path $LogConfig.LogDirectory $LogFile
        } else {
            Join-Path $LogConfig.LogDirectory (Get-LogFileName)
        }
        
        try {
            $logEntry | Out-File -FilePath $logFilePath -Append -Encoding UTF8
        } catch {
            Write-ColorOutput "ログファイルの書き込みに失敗: $($_.Exception.Message)" "Red"
        }
    }
}

# デバッグログ
function Write-DebugLog {
    param([string]$Message)
    Write-Log $Message ([LogLevel]::Debug)
}

# 情報ログ
function Write-InfoLog {
    param([string]$Message)
    Write-Log $Message ([LogLevel]::Info)
}

# 警告ログ
function Write-WarningLog {
    param([string]$Message)
    Write-Log $Message ([LogLevel]::Warning)
}

# エラーログ
function Write-ErrorLog {
    param([string]$Message)
    Write-Log $Message ([LogLevel]::Error)
}

# 致命的エラーログ
function Write-FatalLog {
    param([string]$Message)
    Write-Log $Message ([LogLevel]::Fatal)
}

# ログ設定を更新
function Set-LogConfig {
    param(
        [LogLevel]$Level = $null,
        [bool]$EnableFileLog = $null,
        [string]$LogDirectory = $null,
        [bool]$EnableColor = $null,
        [string]$TimestampFormat = $null
    )
    
    if ($Level -ne $null) { $script:LogConfig.Level = $Level }
    if ($EnableFileLog -ne $null) { $script:LogConfig.EnableFileLog = $EnableFileLog }
    if ($LogDirectory -ne $null) { $script:LogConfig.LogDirectory = $LogDirectory }
    if ($EnableColor -ne $null) { $script:LogConfig.EnableColor = $EnableColor }
    if ($TimestampFormat -ne $null) { $script:LogConfig.TimestampFormat = $TimestampFormat }
}

# ログ設定を取得
function Get-LogConfig {
    return $script:LogConfig.Clone()
}

# エラーハンドリング関数
function Invoke-WithErrorHandling {
    param(
        [scriptblock]$ScriptBlock,
        [string]$ErrorMessage = "エラーが発生しました",
        [bool]$ContinueOnError = $false
    )
    
    try {
        & $ScriptBlock
        return $true
    } catch {
        Write-ErrorLog "$ErrorMessage`: $($_.Exception.Message)"
        if (-not $ContinueOnError) {
            throw
        }
        return $false
    }
}

# 初期化
Initialize-LogDirectory 