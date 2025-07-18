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
    [ValidateSet("build", "test", "release", "clean", "install", "cross-build", "help")]
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

# 設定
$ErrorActionPreference = "Stop"
$BinaryName = $Output
$BuildDir = "build"
$Version = if (git describe --tags --always --dirty 2>$null) { git describe --tags --always --dirty } else { "dev" }
$BuildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$LDFlags = "-ldflags `"-X main.Version=$Version -X main.BuildTime=$BuildTime`""

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
    Write-ColorOutput "ビルド中..." "Green"
    
    if (-not (Test-GoCommand)) { return }
    
    try {
        go build $LDFlags -o $BinaryName
        if (Test-Path $BinaryName) {
            $size = (Get-Item $BinaryName).Length / 1MB
            Write-ColorOutput "ビルド完了: $BinaryName ($([math]::Round($size, 2)) MB)" "Green"
        } else {
            Write-ColorOutput "エラー: ビルドに失敗しました" "Red"
        }
    }
    catch {
        Write-ColorOutput "ビルドエラー: $_" "Red"
    }
}

# リリースビルド
function Build-Release {
    Write-ColorOutput "リリースビルド中..." "Green"
    
    if (-not (Test-GoCommand)) { return }
    
    try {
        go build $LDFlags -ldflags "-s -w" -o $BinaryName
        if (Test-Path $BinaryName) {
            $size = (Get-Item $BinaryName).Length / 1MB
            Write-ColorOutput "リリースビルド完了: $BinaryName ($([math]::Round($size, 2)) MB)" "Green"
        } else {
            Write-ColorOutput "エラー: リリースビルドに失敗しました" "Red"
        }
    }
    catch {
        Write-ColorOutput "リリースビルドエラー: $_" "Red"
    }
}

# クロスプラットフォームビルド
function Build-CrossPlatform {
    Write-ColorOutput "クロスプラットフォームビルド中..." "Green"
    
    if (-not (Test-GoCommand)) { return }
    
    if (-not (Test-Path $BuildDir)) {
        New-Item -ItemType Directory -Path $BuildDir | Out-Null
    }
    
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
    
    foreach ($p in $platforms) {
        Write-ColorOutput "$($p.OS) $($p.ARCH)..." "Yellow"
        try {
            $env:GOOS = $p.OS
            $env:GOARCH = $p.ARCH
            $outputName = "gopier-$($p.OS)-$($p.ARCH)$($p.Ext)"
            go build $LDFlags -o "$BuildDir\$outputName"
            Write-ColorOutput "  ✓ $outputName" "Green"
        }
        catch {
            Write-ColorOutput "  ✗ $($p.OS) $($p.ARCH) ビルド失敗: $_" "Red"
        }
    }
    
    Write-ColorOutput "クロスプラットフォームビルド完了" "Green"
}

# テスト実行
function Test-Project {
    Write-ColorOutput "テスト実行中..." "Green"
    
    if (-not (Test-GoCommand)) { return }
    
    try {
        go test -v ./...
        Write-ColorOutput "テスト完了" "Green"
    }
    catch {
        Write-ColorOutput "テストエラー: $_" "Red"
    }
}

# テストカバレッジ
function Test-Coverage {
    Write-ColorOutput "テストカバレッジ実行中..." "Green"
    
    if (-not (Test-GoCommand)) { return }
    
    try {
        go test -v -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
        Write-ColorOutput "カバレッジレポート: coverage.html" "Green"
    }
    catch {
        Write-ColorOutput "カバレッジテストエラー: $_" "Red"
    }
}

# クリーンアップ
function Clean-Project {
    Write-ColorOutput "クリーンアップ中..." "Yellow"
    
    $filesToRemove = @($BinaryName, "coverage.out", "coverage.html")
    foreach ($file in $filesToRemove) {
        if (Test-Path $file) {
            Remove-Item $file -Force
            Write-ColorOutput "削除: $file" "Yellow"
        }
    }
    
    if (Test-Path $BuildDir) {
        Remove-Item $BuildDir -Recurse -Force
        Write-ColorOutput "削除: $BuildDir" "Yellow"
    }
    
    Write-ColorOutput "クリーンアップ完了" "Green"
}

# インストール
function Install-Project {
    Write-ColorOutput "インストール中..." "Green"
    
    if (-not (Test-Path $BinaryName)) {
        Write-ColorOutput "エラー: ビルドファイルが見つかりません。先にビルドを実行してください。" "Red"
        return
    }
    
    $goPath = $env:GOPATH
    if (-not $goPath) {
        $goPath = "$env:USERPROFILE\go"
    }
    
    $installPath = "$goPath\bin"
    if (-not (Test-Path $installPath)) {
        New-Item -ItemType Directory -Path $installPath -Force | Out-Null
    }
    
    try {
        Copy-Item $BinaryName "$installPath\" -Force
        Write-ColorOutput "インストール完了: $installPath\$BinaryName" "Green"
    }
    catch {
        Write-ColorOutput "インストールエラー: $_" "Red"
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
switch ($Action.ToLower()) {
    "build" { Build-Project }
    "release" { Build-Release }
    "cross-build" { Build-CrossPlatform }
    "test" { Test-Project }
    "clean" { Clean-Project }
    "install" { Install-Project }
    "help" { Show-Help }
    default { 
        Write-ColorOutput "不明なアクション: $Action" "Red"
        Show-Help
    }
} 