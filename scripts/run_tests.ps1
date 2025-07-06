# Gopier テスト実行スクリプト
# 管理者権限が必要なテストとそうでないテストを分離して実行

param(
    [switch]$Admin,
    [switch]$Short,
    [switch]$All,
    [switch]$Help,
    [switch]$AutoConfirm,
    [switch]$ShowLog
)

function Show-Help {
    Write-Host "Gopier テスト実行スクリプト" -ForegroundColor Green
    Write-Host ""
    Write-Host "使用方法:" -ForegroundColor Yellow
    Write-Host "  .\scripts\run_tests.ps1 [オプション]" -ForegroundColor White
    Write-Host ""
    Write-Host "オプション:" -ForegroundColor Yellow
    Write-Host "  -Admin        管理者権限が必要なテストのみ実行" -ForegroundColor White
    Write-Host "  -Short        短時間テストのみ実行（管理者権限不要）" -ForegroundColor White
    Write-Host "  -All          すべてのテストを実行" -ForegroundColor White
    Write-Host "  -AutoConfirm  管理者権限テスト実行時の確認をスキップ" -ForegroundColor White
    Write-Host "  -ShowLog      テスト実行後にログファイルを表示" -ForegroundColor White
    Write-Host "  -Help         このヘルプを表示" -ForegroundColor White
    Write-Host ""
    Write-Host "例:" -ForegroundColor Yellow
    Write-Host "  .\scripts\run_tests.ps1 -Short" -ForegroundColor White
    Write-Host "  .\scripts\run_tests.ps1 -Admin" -ForegroundColor White
    Write-Host "  .\scripts\run_tests.ps1 -Admin -AutoConfirm" -ForegroundColor White
    Write-Host "  .\scripts\run_tests.ps1 -Admin -ShowLog" -ForegroundColor White
    Write-Host "  .\scripts\run_tests.ps1 -All" -ForegroundColor White
    Write-Host ""
}

function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Run-ShortTests {
    Write-Host "短時間テストを実行中..." -ForegroundColor Green
    Write-Host "（管理者権限不要なテストのみ）" -ForegroundColor Gray
    
    $env:TESTING = "1"
    go test ./internal/permissions/... -short -v
    $result = $LASTEXITCODE
    
    if ($result -eq 0) {
        Write-Host "短時間テストが成功しました" -ForegroundColor Green
    } else {
        Write-Host "短時間テストが失敗しました" -ForegroundColor Red
    }
    
    return $result
}

function Run-AdminTests {
    Write-Host "管理者権限テストを実行中..." -ForegroundColor Green
    Write-Host "（管理者権限が必要なテストのみ）" -ForegroundColor Gray
    
    $adminArgs = "-Admin"
    if ($AutoConfirm) {
        $adminArgs += " -AutoConfirm"
    }
    if ($ShowLog) {
        $adminArgs += " -ShowLog"
    }
    
    if (-not (Test-AdminPrivileges)) {
        Write-Host "管理者権限が必要です。UACダイアログが表示されます..." -ForegroundColor Yellow
        Write-Host "管理者権限でPowerShellを起動してテストを実行します" -ForegroundColor Gray
        
        $scriptPath = $MyInvocation.MyCommand.Path
        $projectRoot = Split-Path -Parent (Split-Path -Parent $scriptPath)
        
        # 現在の環境変数を取得
        $envVars = @{
            "GOPATH" = $env:GOPATH
            "GOROOT" = $env:GOROOT
            "PATH" = $env:PATH
            "GOOS" = $env:GOOS
            "GOARCH" = $env:GOARCH
        }
        
        # 環境変数を文字列として構築
        $envString = ""
        foreach ($key in $envVars.Keys) {
            if ($envVars[$key]) {
                $envString += "`$env:$key = '$($envVars[$key])'; "
            }
        }
        
        $arguments = @(
            "-NoProfile",
            "-ExecutionPolicy", "Bypass",
            "-Command", "$envString Set-Location '$projectRoot'; & '$scriptPath' $adminArgs"
        )
        
        try {
            Write-Host "管理者権限でPowerShellを起動中..." -ForegroundColor Gray
            Write-Host "コマンド: $($arguments -join ' ')" -ForegroundColor Gray
            
            $process = Start-Process powershell -ArgumentList $arguments -Verb RunAs -Wait -PassThru
            
            Write-Host "管理者権限プロセスの終了コード: $($process.ExitCode)" -ForegroundColor Gray
            return $process.ExitCode
        } catch {
            Write-Host "管理者権限でのテスト実行に失敗しました: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host "詳細: $($_.Exception)" -ForegroundColor Red
            return 1
        }
    }
    
    # 管理者権限で実行されている場合
    Write-Host "管理者権限で実行中: $(Test-AdminPrivileges)" -ForegroundColor Green
    Write-Host "現在のディレクトリ: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Go環境: $env:GOROOT" -ForegroundColor Gray
    
    # Goコマンドの存在確認
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "エラー: Goコマンドが見つかりません" -ForegroundColor Red
        Write-Host "Goがインストールされているか、PATHに設定されているか確認してください" -ForegroundColor Yellow
        return 1
    }
    
    # モジュールの依存関係を確認
    Write-Host "モジュールの依存関係を確認中..." -ForegroundColor Gray
    try {
        go mod tidy
        Write-Host "モジュールの依存関係を更新しました" -ForegroundColor Gray
    } catch {
        Write-Host "警告: モジュールの依存関係の更新に失敗しました: $($_.Exception.Message)" -ForegroundColor Yellow
    }
    
    # テストファイルの存在確認
    $testFiles = @(
        "internal/permissions/permissions_windows_admin_test.go",
        "internal/permissions/uac_elevation_windows_test.go"
    )
    
    foreach ($testFile in $testFiles) {
        if (-not (Test-Path $testFile)) {
            Write-Host "警告: テストファイルが見つかりません: $testFile" -ForegroundColor Yellow
        } else {
            Write-Host "テストファイル確認: $testFile" -ForegroundColor Gray
        }
    }
    
    $env:TESTING = "1"
    Write-Host "テスト実行中: go test ./internal/permissions/... -run 'WithAdmin' -v" -ForegroundColor Gray
    
    $startTime = Get-Date
    try {
        # テスト出力をログファイルにも記録
        $testOutput = go test ./internal/permissions/... -run "WithAdmin" -v 2>&1
        $result = $LASTEXITCODE
        
        # ログファイルに出力
        $logFile = "admin_tests_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"
        $logFile = Join-Path (Get-Location) $logFile
        $logContent = @"
=== 管理者権限テスト実行ログ ===
実行日時: $(Get-Date)
管理者権限: $(Test-AdminPrivileges)
現在のディレクトリ: $(Get-Location)
Go環境: $env:GOROOT

=== テスト出力 ===
$testOutput

=== 実行結果 ===
終了コード: $result
実行時間: $($duration.TotalSeconds.ToString('F2'))秒
"@
        $logContent | Out-File -FilePath $logFile -Encoding UTF8
        
    } catch {
        Write-Host "テスト実行中にエラーが発生しました: $($_.Exception.Message)" -ForegroundColor Red
        $result = 1
        
        # エラーもログファイルに記録
        $errorContent = @"
=== 管理者権限テスト実行エラー ===
実行日時: $(Get-Date)
エラー: $($_.Exception.Message)
詳細: $($_.Exception)
"@
        $errorContent | Out-File -FilePath $logFile -Encoding UTF8
    }
    $endTime = Get-Date
    $duration = $endTime - $startTime
    
    Write-Host "テスト実行時間: $($duration.TotalSeconds.ToString('F2'))秒" -ForegroundColor Gray
    
    if ($result -eq 0) {
        Write-Host "管理者権限テストが成功しました" -ForegroundColor Green
        Write-Host "✓ すべての管理者権限テストが正常に完了しました" -ForegroundColor Green
        Write-Host "ログファイル: $logFile" -ForegroundColor Gray
    } else {
        Write-Host "管理者権限テストが失敗しました（終了コード: $result）" -ForegroundColor Red

        # 失敗の原因を分析
        switch ($result) {
            1 { Write-Host "原因: テストの実行に失敗しました" -ForegroundColor Yellow }
            2 { Write-Host "原因: テストのコンパイルに失敗しました" -ForegroundColor Yellow }
            default { Write-Host "原因: 不明なエラー（終了コード: $result）" -ForegroundColor Yellow }
        }

        Write-Host "対処法:" -ForegroundColor Yellow
        Write-Host "  1. 管理者権限で実行されているか確認してください" -ForegroundColor White
        Write-Host "  2. Goの環境が正しく設定されているか確認してください" -ForegroundColor White
        Write-Host "  3. テストファイルが存在するか確認してください" -ForegroundColor White
        Write-Host "ログファイル: $logFile" -ForegroundColor Gray
    }

    # ログファイルを表示するオプション
    if ($ShowLog) {
        if (Test-Path $logFile) {
            Write-Host "`n=== ログファイルの内容 ===" -ForegroundColor Cyan
            try {
                $logContent = Get-Content $logFile -Encoding UTF8 -ErrorAction Stop
                if ($logContent.Count -gt 0) {
                    foreach ($line in $logContent) {
                        if ($line -match "===.*===") {
                            Write-Host $line -ForegroundColor Magenta
                        } elseif ($line -match "PASS|成功") {
                            Write-Host $line -ForegroundColor Green
                        } elseif ($line -match "FAIL|失敗|エラー") {
                            Write-Host $line -ForegroundColor Red
                        } elseif ($line -match "WARNING|警告") {
                            Write-Host $line -ForegroundColor Yellow
                        } elseif ($line -match "DEBUG|デバッグ") {
                            Write-Host $line -ForegroundColor Gray
                        } else {
                            Write-Host $line
                        }
                    }
                } else {
                    Write-Host "ログファイルは空です" -ForegroundColor Gray
                }
            } catch {
                Write-Host "ログファイルの読み込みに失敗しました: $($_.Exception.Message)" -ForegroundColor Red
            }
        } else {
            Write-Host "警告: ログファイルが見つかりません: $logFile" -ForegroundColor Yellow
        }
    }
    
    return $result
}

function Run-AllTests {
    Write-Host "すべてのテストを実行中..." -ForegroundColor Green
    
    $env:TESTING = "1"
    go test ./internal/permissions/... -v
    $result = $LASTEXITCODE
    
    if ($result -eq 0) {
        Write-Host "すべてのテストが成功しました" -ForegroundColor Green
    } else {
        Write-Host "一部のテストが失敗しました" -ForegroundColor Red
    }
    
    return $result
}

# メイン処理
if ($Help) {
    Show-Help
    exit 0
}

# プロジェクトルートディレクトリに移動
$scriptPath = $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent (Split-Path -Parent $scriptPath)

if ($projectRoot -ne (Get-Location).Path) {
    Write-Host "プロジェクトルートディレクトリに移動中: $projectRoot" -ForegroundColor Gray
    Set-Location $projectRoot
}

# 現在のディレクトリを確認
if (-not (Test-Path "go.mod")) {
    Write-Host "エラー: go.modファイルが見つかりません" -ForegroundColor Red
    Write-Host "プロジェクトのルートディレクトリで実行してください" -ForegroundColor Yellow
    exit 1
}

# オプションの確認
if ($Admin -and $Short) {
    Write-Host "エラー: -Adminと-Shortは同時に指定できません" -ForegroundColor Red
    exit 1
}

if ($Admin -and $All) {
    Write-Host "エラー: -Adminと-Allは同時に指定できません" -ForegroundColor Red
    exit 1
}

if ($Short -and $All) {
    Write-Host "エラー: -Shortと-Allは同時に指定できません" -ForegroundColor Red
    exit 1
}

# デフォルトは短時間テスト
if (-not $Admin -and -not $Short -and -not $All) {
    $Short = $true
}

# テスト実行
$exitCode = 0

if ($Short) {
    $exitCode = Run-ShortTests
} elseif ($Admin) {
    $exitCode = Run-AdminTests
} elseif ($All) {
    $exitCode = Run-AllTests
}

exit $exitCode 