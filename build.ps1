#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Gopier ビルドスクリプト

.DESCRIPTION
    Gopierプロジェクトのビルド、テスト、リリース用のPowerShellスクリプト

.PARAMETER Action
    実行するアクション（build, test, release, clean, install, help）

.PARAMETER Platform
    ビルドするプラットフォーム（windows, linux, darwin, all）

.PARAMETER Architecture
    ビルドするアーキテクチャ（amd64, arm64）

.PARAMETER Output
    出力ファイル名

.EXAMPLE
    .\build.ps1 build
    .\build.ps1 test
    .\build.ps1 release
    .\build.ps1 cross-build -Platform all
#>

param(
    [Parameter(Position=0)]
    [ValidateSet("build", "test", "test-coverage", "release", "clean", "install", "cross-build", "help")]
    [string]$Action = "build",
    
    [Parameter()]
    [ValidateSet("windows", "linux", "darwin", "all")]
    [string]$Platform = "windows",
    
    [Parameter()]
    [ValidateSet("amd64", "arm64")]
    [string]$Architecture = "amd64",
    
    [Parameter()]
    [string]$Output = "gopier.exe"
)

# 共通モジュールを読み込み
$scriptPath = $MyInvocation.MyCommand.Path
$scriptDir = Split-Path -Parent $scriptPath
Import-Module (Join-Path $scriptDir "scripts\common\GopierCommon.psm1") -Force

# プロジェクトルートに移動
if (-not (Set-ProjectRoot)) {
    Write-ErrorLog "プロジェクトルートディレクトリが見つかりません"
    exit 1
}

# 設定を取得
$buildConfig = Get-BuildConfig
$logConfig = Get-LogConfig
$platformConfig = Get-PlatformConfig

# ログ設定を適用
Set-LogConfig -Level $logConfig.Level -EnableFileLog $logConfig.EnableFileLog -LogDirectory $logConfig.LogDirectory

# 設定
$ErrorActionPreference = "Stop"
$BinaryName = $Output
$BuildDir = $buildConfig.BuildDir
$Version = (Get-VersionInfo).Version
$BuildTime = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
$LDFlags = "-X github.com/sakuhanight/gopier/cmd.Version=$Version -X github.com/sakuhanight/gopier/cmd.BuildTime=$BuildTime"

# 色付き出力関数
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

# Goコマンドの確認
function Test-GoCommand {
    try {
        $env:PATH += ";C:\Program Files\Go\bin"
        go version | Out-Null
        return $true
    }
    catch {
        Write-ColorOutput "エラー: Goがインストールされていないか、PATHに設定されていません" "Red"
        Write-ColorOutput "Goをインストールしてください: https://golang.org/dl/" "Yellow"
        return $false
    }
}

# 通常ビルド
function Build-Project {
    Write-InfoLog "ビルド中..."
    
    if (-not (Test-GoCommand)) { 
        Write-ErrorLog "Goがインストールされていないか、PATHに設定されていません"
        Write-InfoLog "Goをインストールしてください: https://golang.org/dl/"
        exit 1
    }
    
    $executionResult = Measure-ExecutionTime -ScriptBlock {
        # メモリ使用量を最適化
        Set-EnvironmentVariable -Name "GOGC" -Value $buildConfig.GarbageCollection
        Set-EnvironmentVariable -Name "GOMEMLIMIT" -Value $buildConfig.MemoryLimit
        
        # ビルドキャッシュをクリア
        go clean -cache
        
        # 依存関係を事前にダウンロード
        Write-InfoLog "依存関係をダウンロード中..."
        go mod download
        
        # ビルド実行
        Write-InfoLog "Goビルド実行中..."
        go build -ldflags $LDFlags -o $BinaryName
    } -Description "ビルド処理"
    
    if ($LASTEXITCODE -eq 0) {
        $fileInfo = Test-FileWithInfo -Path $BinaryName -GetInfo
        if ($fileInfo) {
            $size = Format-FileSize -SizeInBytes $fileInfo.Length
            Write-InfoLog "ビルド完了: $BinaryName ($size) - 実行時間: $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
            exit 0
        } else {
            Write-ErrorLog "ビルドファイルが生成されませんでした"
            exit 1
        }
    } else {
        Write-ErrorLog "ビルドに失敗しました"
        exit 1
    }
}

# リリースビルド
function Build-Release {
    Write-InfoLog "リリースビルド中..."
    
    if (-not (Test-GoCommand)) { 
        Write-ErrorLog "Goがインストールされていないか、PATHに設定されていません"
        exit 1
    }
    
    $executionResult = Measure-ExecutionTime -ScriptBlock {
        go build -ldflags "$LDFlags -s -w" -o $BinaryName
    } -Description "リリースビルド処理"
    
    if ($LASTEXITCODE -eq 0) {
        $fileInfo = Test-FileWithInfo -Path $BinaryName -GetInfo
        if ($fileInfo) {
            $size = Format-FileSize -SizeInBytes $fileInfo.Length
            Write-InfoLog "リリースビルド完了: $BinaryName ($size) - 実行時間: $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
            exit 0
        } else {
            Write-ErrorLog "リリースビルドファイルが生成されませんでした"
            exit 1
        }
    } else {
        Write-ErrorLog "リリースビルドに失敗しました"
        exit 1
    }
}

# クロスプラットフォームビルド
function Build-CrossPlatform {
    Write-InfoLog "クロスプラットフォームビルド中..."
    
    if (-not (Test-GoCommand)) { 
        Write-ErrorLog "Goがインストールされていないか、PATHに設定されていません"
        exit 1
    }
    
    Test-DirectoryWithCreate -Path $BuildDir -CreateIfNotExists
    
    $platforms = @()
    if ($Platform -eq "all") {
        $platforms = @(
            @{OS="windows"; ARCH="amd64"; Ext=".exe"},
            @{OS="linux"; ARCH="amd64"; Ext=""},
            @{OS="darwin"; ARCH="amd64"; Ext=""},
            @{OS="darwin"; ARCH="arm64"; Ext=""}
        )
    } else {
        $ext = if ($Platform -eq "windows") { ".exe" } else { "" }
        $platforms = @(@{OS=$Platform; ARCH=$Architecture; Ext=$ext})
    }
    
    $buildSuccess = $true
    $totalStartTime = Get-Date
    
    foreach ($p in $platforms) {
        Write-InfoLog "$($p.OS) $($p.ARCH) ビルド中..."
        $platformStartTime = Get-Date
        
        $executionResult = Measure-ExecutionTime -ScriptBlock {
            Set-EnvironmentVariable -Name "GOOS" -Value $p.OS
            Set-EnvironmentVariable -Name "GOARCH" -Value $p.ARCH
            $outputName = "gopier-$($p.OS)-$($p.ARCH)$($p.Ext)"
            go build -ldflags $LDFlags -o "$BuildDir\$outputName"
        } -Description "$($p.OS) $($p.ARCH) ビルド"
        
        if ($LASTEXITCODE -eq 0) {
            $fileInfo = Test-FileWithInfo -Path "$BuildDir\gopier-$($p.OS)-$($p.ARCH)$($p.Ext)" -GetInfo
            if ($fileInfo) {
                $size = Format-FileSize -SizeInBytes $fileInfo.Length
                Write-InfoLog "  ✓ gopier-$($p.OS)-$($p.ARCH)$($p.Ext) ($size) - $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
            } else {
                Write-ErrorLog "  ✗ $($p.OS) $($p.ARCH) ビルドファイルが生成されませんでした"
                $buildSuccess = $false
            }
        } else {
            Write-ErrorLog "  ✗ $($p.OS) $($p.ARCH) ビルド失敗"
            $buildSuccess = $false
        }
    }
    
    $totalDuration = (Get-Date) - $totalStartTime
    
    if ($buildSuccess) {
        Write-InfoLog "クロスプラットフォームビルド完了 - 総実行時間: $($totalDuration.TotalSeconds.ToString('F2'))秒"
        exit 0
    } else {
        Write-ErrorLog "クロスプラットフォームビルド失敗 - 総実行時間: $($totalDuration.TotalSeconds.ToString('F2'))秒"
        exit 1
    }
}

# テスト実行
function Test-Project {
    Write-InfoLog "テスト実行中..."
    
    if (-not (Test-GoCommand)) { 
        Write-ErrorLog "Goがインストールされていないか、PATHに設定されていません"
        exit 1
    }
    
    $testConfig = Get-TestConfig
    $totalStartTime = Get-Date
    
    try {
        Set-EnvironmentVariable -Name "TESTING" -Value "1"
        
        # 通常テスト
        Write-InfoLog "通常テスト実行中..."
        $normalTestResult = Measure-ExecutionTime -ScriptBlock {
            go test -v ./...
        } -Description "通常テスト"
        
        if ($LASTEXITCODE -eq 0) {
            Write-InfoLog "通常テスト成功 - $($normalTestResult.Duration.TotalSeconds.ToString('F2'))秒"
        } else {
            Write-ErrorLog "通常テスト失敗"
            throw "通常テストが失敗しました"
        }
        
        # 統合テスト
        Write-InfoLog "統合テスト実行中..."
        $integrationTestResult = Measure-ExecutionTime -ScriptBlock {
            go test -v ./tests/...
        } -Description "統合テスト"
        
        if ($LASTEXITCODE -eq 0) {
            Write-InfoLog "統合テスト成功 - $($integrationTestResult.Duration.TotalSeconds.ToString('F2'))秒"
        } else {
            Write-ErrorLog "統合テスト失敗"
            throw "統合テストが失敗しました"
        }
        
        $totalDuration = (Get-Date) - $totalStartTime
        Write-InfoLog "テスト完了 - 総実行時間: $($totalDuration.TotalSeconds.ToString('F2'))秒"
        exit 0
    }
    catch {
        Write-ErrorLog "テストエラー: $($_.Exception.Message)"
        exit 1
    }
}

# テストカバレッジ
function Test-Coverage {
    Write-InfoLog "テストカバレッジ実行中..."
    
    if (-not (Test-GoCommand)) { 
        Write-ErrorLog "Goがインストールされていないか、PATHに設定されていません"
        exit 1
    }
    
    $testConfig = Get-TestConfig
    $totalStartTime = Get-Date
    
    try {
        Set-EnvironmentVariable -Name "TESTING" -Value "1"
        
        # 通常テストカバレッジ
        Write-InfoLog "通常テストカバレッジ実行中..."
        $normalCoverageResult = Measure-ExecutionTime -ScriptBlock {
            go test -v -coverprofile=$($testConfig.CoverageOutput) ./...
        } -Description "通常テストカバレッジ"
        
        if ($LASTEXITCODE -eq 0) {
            Write-InfoLog "通常テストカバレッジ成功 - $($normalCoverageResult.Duration.TotalSeconds.ToString('F2'))秒"
        } else {
            Write-ErrorLog "通常テストカバレッジ失敗"
            throw "通常テストカバレッジが失敗しました"
        }
        
        # 統合テストカバレッジ
        Write-InfoLog "統合テストカバレッジ実行中..."
        $integrationCoverageResult = Measure-ExecutionTime -ScriptBlock {
            go test -v -coverprofile=coverage-integration.out ./tests/...
        } -Description "統合テストカバレッジ"
        
        if ($LASTEXITCODE -eq 0) {
            Write-InfoLog "統合テストカバレッジ成功 - $($integrationCoverageResult.Duration.TotalSeconds.ToString('F2'))秒"
        } else {
            Write-ErrorLog "統合テストカバレッジ失敗"
            throw "統合テストカバレッジが失敗しました"
        }
        
        # カバレッジファイルをマージ
        if (Test-Path $testConfig.CoverageOutput -and Test-Path "coverage-integration.out") {
            Write-InfoLog "カバレッジファイルをマージ中..."
            $coverageContent = Get-Content $testConfig.CoverageOutput
            $integrationContent = Get-Content "coverage-integration.out" | Select-Object -Skip 1
            $coverageContent + $integrationContent | Set-Content $testConfig.CoverageOutput
            Remove-Item "coverage-integration.out"
        }
        
        # HTMLレポート生成
        Write-InfoLog "HTMLレポート生成中..."
        go tool cover -html=$testConfig.CoverageOutput -o $testConfig.CoverageHTML
        
        $totalDuration = (Get-Date) - $totalStartTime
        Write-InfoLog "カバレッジレポート: $testConfig.CoverageHTML - 総実行時間: $($totalDuration.TotalSeconds.ToString('F2'))秒"
        exit 0
    }
    catch {
        Write-ErrorLog "カバレッジテストエラー: $($_.Exception.Message)"
        exit 1
    }
}

# クリーンアップ
function Clean-Project {
    Write-InfoLog "クリーンアップ中..."
    
    $filesToRemove = @($BinaryName, "coverage.out", "coverage.html")
    foreach ($file in $filesToRemove) {
        if (Test-Path $file) {
            Remove-Item $file -Force
            Write-InfoLog "削除: $file"
        }
    }
    
    if (Test-Path $BuildDir) {
        Remove-Item $BuildDir -Recurse -Force
        Write-InfoLog "削除: $BuildDir"
    }
    
    Write-InfoLog "クリーンアップ完了"
}

# インストール
function Install-Project {
    Write-InfoLog "インストール中..."
    
    if (-not (Test-Path $BinaryName)) {
        Write-ErrorLog "ビルドファイルが見つかりません。先にビルドを実行してください。"
        return
    }
    
    $goPath = Get-EnvironmentVariable -Name "GOPATH" -DefaultValue "$env:USERPROFILE\go"
    $installPath = "$goPath\bin"
    
    Test-DirectoryWithCreate -Path $installPath -CreateIfNotExists
    
    try {
        Copy-Item $BinaryName "$installPath\" -Force
        Write-InfoLog "インストール完了: $installPath\$BinaryName"
    }
    catch {
        Write-ErrorLog "インストールエラー: $($_.Exception.Message)"
    }
}

# ヘルプ表示
function Show-Help {
    Write-ColorOutput "Gopier ビルドスクリプト" "Cyan"
    Write-ColorOutput "========================" "Cyan"
    Write-ColorOutput ""
    Write-ColorOutput "使用方法:" "White"
    Write-ColorOutput "  .\build.ps1 [Action] [Options]" "White"
    Write-ColorOutput ""
    Write-ColorOutput "アクション:" "White"
    Write-ColorOutput "  build        - 通常ビルド" "White"
    Write-ColorOutput "  release      - リリースビルド（最適化）" "White"
    Write-ColorOutput "  cross-build  - クロスプラットフォームビルド" "White"
    Write-ColorOutput "  test         - テスト実行" "White"
    Write-ColorOutput "  test-coverage - テストカバレッジ実行" "White"
    Write-ColorOutput "  clean        - クリーンアップ" "White"
    Write-ColorOutput "  install      - インストール" "White"
    Write-ColorOutput "  help         - このヘルプを表示" "White"
    Write-ColorOutput ""
    Write-ColorOutput "オプション:" "White"
    Write-ColorOutput "  -Platform     - プラットフォーム (windows, linux, darwin, all)" "White"
    Write-ColorOutput "  -Architecture - アーキテクチャ (amd64, arm64)" "White"
    Write-ColorOutput "  -Output       - 出力ファイル名" "White"
    Write-ColorOutput ""
    Write-ColorOutput "例:" "White"
    Write-ColorOutput "  .\build.ps1 build" "White"
    Write-ColorOutput "  .\build.ps1 cross-build -Platform all" "White"
    Write-ColorOutput "  .\build.ps1 test" "White"
    Write-ColorOutput "  .\build.ps1 release" "White"
}

# メイン処理
Write-InfoLog "Gopier ビルドスクリプト開始"
Write-InfoLog "アクション: $Action"
Write-InfoLog "プラットフォーム: $Platform"
Write-InfoLog "アーキテクチャ: $Architecture"

switch ($Action.ToLower()) {
    "build" { Build-Project }
    "release" { Build-Release }
    "cross-build" { Build-CrossPlatform }
    "test" { Test-Project }
    "test-coverage" { Test-Coverage }
    "clean" { Clean-Project }
    "install" { Install-Project }
    "help" { Show-Help }
    default { 
        Write-ErrorLog "不明なアクション: $Action"
        Show-Help
    }
} 