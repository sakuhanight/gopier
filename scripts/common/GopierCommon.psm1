# Gopier 共通モジュール
# プロジェクト全体で使用される共通機能を提供

# モジュール情報
$PSScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path

# 共通モジュールを読み込み
. (Join-Path $PSScriptRoot "Logger.ps1")
. (Join-Path $PSScriptRoot "Utils.ps1")
. (Join-Path $PSScriptRoot "Config.ps1")

# エクスポートする関数
Export-ModuleMember -Function @(
    # Logger functions
    "Write-Log",
    "Write-DebugLog",
    "Write-InfoLog",
    "Write-WarningLog",
    "Write-ErrorLog",
    "Write-FatalLog",
    "Write-ColorOutput",
    "Set-LogConfig",
    "Get-LogConfig",
    "Invoke-WithErrorHandling",
    
    # Utils functions
    "Test-AdminPrivileges",
    "Test-GoCommand",
    "Get-ProjectRoot",
    "Set-ProjectRoot",
    "Format-FileSize",
    "Measure-ExecutionTime",
    "Set-EnvironmentVariable",
    "Get-EnvironmentVariable",
    "Test-FileWithInfo",
    "Test-DirectoryWithCreate",
    "Invoke-ProcessWithResult",
    "Read-ConfigFile",
    "Save-ConfigFile",
    "Get-VersionInfo",
    
    # Config functions
    "Get-ProjectConfig",
    "Set-ProjectConfig",
    "Get-BuildConfig",
    "Get-TestConfig",
    "Get-LogConfig",
    "Get-PlatformConfig",
    "Get-AdminConfig",
    "Test-ProjectConfig",
    "Initialize-ProjectConfig",
    "Reset-ProjectConfig",
    "Show-ProjectConfig"
) 