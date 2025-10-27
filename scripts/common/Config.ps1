# 設定管理モジュール
# プロジェクト全体の設定を一元管理

# デフォルト設定
$script:DefaultConfig = @{
    Build = @{
        BinaryName = "gopier.exe"
        BuildDir = "build"
        EnableOptimization = $true
        EnableDebug = $false
        MemoryLimit = "512MiB"
        GarbageCollection = "50"
    }
    Test = @{
        EnableCoverage = $true
        CoverageOutput = "coverage.out"
        CoverageHTML = "coverage.html"
        TimeoutSeconds = 300
        ParallelTests = $true
    }
    Log = @{
        Level = "Info"
        EnableFileLog = $true
        LogDirectory = "logs"
        EnableColor = $true
        MaxLogFiles = 10
    }
    Platform = @{
        DefaultOS = "windows"
        DefaultArch = "amd64"
        SupportedOS = @("windows", "linux", "darwin")
        SupportedArch = @("amd64", "arm64")
    }
    Admin = @{
        RequireElevation = $true
        AutoConfirm = $false
        LogAdminActions = $true
    }
}

# 設定ファイルパス
$script:ConfigFilePath = "scripts/config.json"

# 設定を読み込み
function Get-ProjectConfig {
    param(
        [string]$Section = $null
    )
    
    # 設定ファイルを読み込み
    $fileConfig = Read-ConfigFile -Path $ConfigFilePath -DefaultConfig @{}
    
    # デフォルト設定とマージ
    $mergedConfig = Merge-Hashtables -Source $DefaultConfig -Target $fileConfig
    
    if ($Section) {
        if ($mergedConfig.ContainsKey($Section)) {
            return $mergedConfig[$Section]
        } else {
            Write-WarningLog "設定セクション '$Section' が見つかりません"
            return @{}
        }
    }
    
    return $mergedConfig
}

# 設定を保存
function Set-ProjectConfig {
    param(
        [hashtable]$Config,
        [string]$Section = $null
    )
    
    $currentConfig = Get-ProjectConfig
    
    if ($Section) {
        if (-not $currentConfig.ContainsKey($Section)) {
            $currentConfig[$Section] = @{}
        }
        $currentConfig[$Section] = Merge-Hashtables -Source $currentConfig[$Section] -Target $Config
    } else {
        $currentConfig = Merge-Hashtables -Source $currentConfig -Target $Config
    }
    
    return Save-ConfigFile -Path $ConfigFilePath -Config $currentConfig
}

# ハッシュテーブルをマージ
function Merge-Hashtables {
    param(
        [hashtable]$Source,
        [hashtable]$Target
    )
    
    $result = $Source.Clone()
    
    foreach ($key in $Target.Keys) {
        if ($result.ContainsKey($key) -and $result[$key] -is [hashtable] -and $Target[$key] -is [hashtable]) {
            $result[$key] = Merge-Hashtables -Source $result[$key] -Target $Target[$key]
        } else {
            $result[$key] = $Target[$key]
        }
    }
    
    return $result
}

# ビルド設定を取得
function Get-BuildConfig {
    return Get-ProjectConfig -Section "Build"
}

# テスト設定を取得
function Get-TestConfig {
    return Get-ProjectConfig -Section "Test"
}

# ログ設定を取得
function Get-LogConfig {
    return Get-ProjectConfig -Section "Log"
}

# プラットフォーム設定を取得
function Get-PlatformConfig {
    return Get-ProjectConfig -Section "Platform"
}

# 管理者権限設定を取得
function Get-AdminConfig {
    return Get-ProjectConfig -Section "Admin"
}

# 設定を検証
function Test-ProjectConfig {
    param(
        [hashtable]$Config = $null
    )
    
    if (-not $Config) {
        $Config = Get-ProjectConfig
    }
    
    $configErrors = @()
    
    # 必須設定の確認
    $requiredSections = @("Build", "Test", "Log", "Platform")
    foreach ($section in $requiredSections) {
        if (-not $Config.ContainsKey($section)) {
            $configErrors += "必須セクション '$section' が不足しています"
        }
    }
    
    # ビルド設定の検証
    if ($Config.Build) {
        if (-not $Config.Build.BinaryName) {
            $configErrors += "Build.BinaryName が設定されていません"
        }
        if (-not $Config.Build.BuildDir) {
            $configErrors += "Build.BuildDir が設定されていません"
        }
    }
    
    # プラットフォーム設定の検証
    if ($Config.Platform) {
        if (-not $Config.Platform.SupportedOS -or $Config.Platform.SupportedOS.Count -eq 0) {
            $configErrors += "Platform.SupportedOS が設定されていません"
        }
        if (-not $Config.Platform.SupportedArch -or $Config.Platform.SupportedArch.Count -eq 0) {
            $configErrors += "Platform.SupportedArch が設定されていません"
        }
    }
    
    return @{
        IsValid = ($configErrors.Count -eq 0)
        Errors = $configErrors
    }
}

# 設定を初期化
function Initialize-ProjectConfig {
    $configValidation = Test-ProjectConfig
    if (-not $configValidation.IsValid) {
        Write-WarningLog "設定に問題があります:"
        foreach ($configError in $configValidation.Errors) {
            Write-WarningLog "  - $configError"
        }
        
        # デフォルト設定で初期化
        Write-InfoLog "デフォルト設定で初期化します"
        Set-ProjectConfig -Config $DefaultConfig
        return $false
    }
    
    return $true
}

# 設定をリセット
function Reset-ProjectConfig {
    if (Test-Path $ConfigFilePath) {
        Remove-Item $ConfigFilePath -Force
        Write-InfoLog "設定ファイルを削除しました"
    }
    
    # デフォルト設定で初期化
    Set-ProjectConfig -Config $DefaultConfig
    Write-InfoLog "デフォルト設定で初期化しました"
}

# 設定を表示
function Show-ProjectConfig {
    param(
        [string]$Section = $null,
        [switch]$Verbose
    )
    
    $config = Get-ProjectConfig -Section $Section
    
    if ($Verbose) {
        $config | ConvertTo-Json -Depth 10 -Compress:$false
    } else {
        $config | Format-List
    }
}

# 初期化
Initialize-ProjectConfig 