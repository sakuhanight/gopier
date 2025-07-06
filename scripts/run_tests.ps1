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

# 共通モジュールを読み込み
$scriptPath = $MyInvocation.MyCommand.Path
$scriptDir = Split-Path -Parent $scriptPath
$projectRoot = Split-Path -Parent $scriptDir
Import-Module (Join-Path $projectRoot "scripts\common\GopierCommon.psm1") -Force

# プロジェクトルートに移動
if (-not (Set-ProjectRoot)) {
    Write-ErrorLog "プロジェクトルートディレクトリが見つかりません"
    exit 1
}

# 設定を取得
$testConfig = Get-TestConfig
$logConfig = Get-LogConfig
$adminConfig = Get-AdminConfig

# ログ設定を適用
Set-LogConfig -Level $logConfig.Level -EnableFileLog $logConfig.EnableFileLog -LogDirectory $logConfig.LogDirectory

function Show-Help {
    Write-ColorOutput "Gopier テスト実行スクリプト" "Green"
    Write-ColorOutput ""
    Write-ColorOutput "使用方法:" "Yellow"
    Write-ColorOutput "  .\scripts\run_tests.ps1 [オプション]" "White"
    Write-ColorOutput ""
    Write-ColorOutput "オプション:" "Yellow"
    Write-ColorOutput "  -Admin        管理者権限が必要なテストのみ実行" "White"
    Write-ColorOutput "  -Short        短時間テストのみ実行（管理者権限不要）" "White"
    Write-ColorOutput "  -All          すべてのテストを実行" "White"
    Write-ColorOutput "  -AutoConfirm  管理者権限テスト実行時の確認をスキップ" "White"
    Write-ColorOutput "  -ShowLog      テスト実行後にログファイルを表示" "White"
    Write-ColorOutput "  -Help         このヘルプを表示" "White"
    Write-ColorOutput ""
    Write-ColorOutput "例:" "Yellow"
    Write-ColorOutput "  .\scripts\run_tests.ps1 -Short" "White"
    Write-ColorOutput "  .\scripts\run_tests.ps1 -Admin" "White"
    Write-ColorOutput "  .\scripts\run_tests.ps1 -Admin -AutoConfirm" "White"
    Write-ColorOutput "  .\scripts\run_tests.ps1 -Admin -ShowLog" "White"
    Write-ColorOutput "  .\scripts\run_tests.ps1 -All" "White"
    Write-ColorOutput ""
}

function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Invoke-ShortTests {
    Write-InfoLog "短時間テストを実行中..."
    Write-InfoLog "（管理者権限不要なテストのみ）"
    
    $executionResult = Measure-ExecutionTime -ScriptBlock {
        Set-EnvironmentVariable -Name "TESTING" -Value "1"
        go test ./internal/permissions/... -short -v
    } -Description "短時間テスト"
    
    $result = $LASTEXITCODE
    
    if ($result -eq 0) {
        Write-InfoLog "短時間テストが成功しました - $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
    } else {
        Write-ErrorLog "短時間テストが失敗しました - $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
    }
    
    return $result
}

function Invoke-AdminTests {
    Write-InfoLog "管理者権限テストを実行中..."
    Write-InfoLog "（管理者権限が必要なテストのみ）"
    
    $adminArgs = "-Admin"
    if ($AutoConfirm) {
        $adminArgs += " -AutoConfirm"
    }
    if ($ShowLog) {
        $adminArgs += " -ShowLog"
    }
    
    if (-not (Test-AdminPrivileges)) {
        Write-WarningLog "管理者権限が必要です。UACダイアログが表示されます..."
        Write-InfoLog "管理者権限でPowerShellを起動してテストを実行します"
        
        $scriptPath = $MyInvocation.MyCommand.Path
        $projectRoot = Split-Path -Parent (Split-Path -Parent $scriptPath)
        
        # 現在の環境変数を取得
        $envVars = @{
            "GOPATH" = Get-EnvironmentVariable -Name "GOPATH"
            "GOROOT" = Get-EnvironmentVariable -Name "GOROOT"
            "PATH" = Get-EnvironmentVariable -Name "PATH"
            "GOOS" = Get-EnvironmentVariable -Name "GOOS"
            "GOARCH" = Get-EnvironmentVariable -Name "GOARCH"
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
            Write-InfoLog "管理者権限でPowerShellを起動中..."
            Write-DebugLog "コマンド: $($arguments -join ' ')"
            
            $processResult = Invoke-ProcessWithResult -FilePath "powershell" -ArgumentList $arguments -TimeoutSeconds $testConfig.TimeoutSeconds
            
            Write-DebugLog "管理者権限プロセスの終了コード: $($processResult.ExitCode)"
            return $processResult.ExitCode
        } catch {
            Write-ErrorLog "管理者権限でのテスト実行に失敗しました: $($_.Exception.Message)"
            return 1
        }
    }
    
    # 管理者権限で実行されている場合
    Write-InfoLog "管理者権限で実行中: $(Test-AdminPrivileges)"
    Write-DebugLog "現在のディレクトリ: $(Get-Location)"
    Write-DebugLog "Go環境: $(Get-EnvironmentVariable -Name 'GOROOT')"
    
    # Goコマンドの存在確認
    if (-not (Test-GoCommand)) {
        Write-ErrorLog "Goコマンドが見つかりません"
        Write-InfoLog "Goがインストールされているか、PATHに設定されているか確認してください"
        return 1
    }
    
    # モジュールの依存関係を確認
    Write-InfoLog "モジュールの依存関係を確認中..."
    $moduleResult = Invoke-WithErrorHandling -ScriptBlock {
        go mod tidy
        Write-InfoLog "モジュールの依存関係を更新しました"
    } -ErrorMessage "モジュールの依存関係の更新に失敗しました" -ContinueOnError $true
    
    # テストファイルの存在確認
    $testFiles = @(
        "internal/permissions/permissions_windows_admin_test.go",
        "internal/permissions/uac_elevation_windows_test.go"
    )
    
    foreach ($testFile in $testFiles) {
        if (-not (Test-Path $testFile)) {
            Write-WarningLog "テストファイルが見つかりません: $testFile"
        } else {
            Write-DebugLog "テストファイル確認: $testFile"
        }
    }
    
    Set-EnvironmentVariable -Name "TESTING" -Value "1"
    Write-InfoLog "テスト実行中: go test ./internal/permissions/... -run 'WithAdmin' -v"
    
    $executionResult = Measure-ExecutionTime -ScriptBlock {
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
Go環境: $(Get-EnvironmentVariable -Name 'GOROOT')

=== テスト出力 ===
$testOutput

=== 実行結果 ===
終了コード: $result
実行時間: $($executionResult.Duration.TotalSeconds.ToString('F2'))秒
"@
        $logContent | Out-File -FilePath $logFile -Encoding UTF8
        
        return @{
            ExitCode = $result
            LogFile = $logFile
            Output = $testOutput
        }
    } -Description "管理者権限テスト"
    
    $result = $executionResult.Result.ExitCode
    $logFile = $executionResult.Result.LogFile
    
    Write-InfoLog "テスト実行時間: $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
    
    if ($result -eq 0) {
        Write-InfoLog "管理者権限テストが成功しました"
        Write-InfoLog "✓ すべての管理者権限テストが正常に完了しました"
        Write-InfoLog "ログファイル: $logFile"
    } else {
        Write-ErrorLog "管理者権限テストが失敗しました（終了コード: $result）"

        # 失敗の原因を分析
        switch ($result) {
            1 { Write-WarningLog "原因: テストの実行に失敗しました" }
            2 { Write-WarningLog "原因: テストのコンパイルに失敗しました" }
            default { Write-WarningLog "原因: 不明なエラー（終了コード: $result）" }
        }

        Write-InfoLog "対処法:"
        Write-InfoLog "  1. 管理者権限で実行されているか確認してください"
        Write-InfoLog "  2. Goの環境が正しく設定されているか確認してください"
        Write-InfoLog "  3. テストファイルが存在するか確認してください"
        Write-InfoLog "ログファイル: $logFile"
    }

    # ログファイルを表示するオプション
    if ($ShowLog) {
        if (Test-Path $logFile) {
            Write-InfoLog "`n=== ログファイルの内容 ==="
            try {
                $logContent = Get-Content $logFile -Encoding UTF8 -ErrorAction Stop
                if ($logContent.Count -gt 0) {
                    foreach ($line in $logContent) {
                        if ($line -match "===.*===") {
                            Write-ColorOutput $line "Magenta"
                        } elseif ($line -match "PASS|成功") {
                            Write-ColorOutput $line "Green"
                        } elseif ($line -match "FAIL|失敗|エラー") {
                            Write-ColorOutput $line "Red"
                        } elseif ($line -match "WARNING|警告") {
                            Write-ColorOutput $line "Yellow"
                        } elseif ($line -match "DEBUG|デバッグ") {
                            Write-ColorOutput $line "Gray"
                        } else {
                            Write-Host $line
                        }
                    }
                } else {
                    Write-InfoLog "ログファイルは空です"
                }
            } catch {
                Write-ErrorLog "ログファイルの読み込みに失敗しました: $($_.Exception.Message)"
            }
        } else {
            Write-WarningLog "ログファイルが見つかりません: $logFile"
        }
    }
    
    return $result
}

function Invoke-AllTests {
    Write-InfoLog "すべてのテストを実行中..."
    
    $executionResult = Measure-ExecutionTime -ScriptBlock {
        Set-EnvironmentVariable -Name "TESTING" -Value "1"
        go test ./internal/permissions/... -v
    } -Description "すべてのテスト"
    
    $result = $LASTEXITCODE
    
    if ($result -eq 0) {
        Write-InfoLog "すべてのテストが成功しました - $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
    } else {
        Write-ErrorLog "一部のテストが失敗しました - $($executionResult.Duration.TotalSeconds.ToString('F2'))秒"
    }
    
    return $result
}

# メイン処理
if ($Help) {
    Show-Help
    exit 0
}

# 現在のディレクトリを確認
if (-not (Test-Path "go.mod")) {
    Write-ErrorLog "go.modファイルが見つかりません"
    Write-InfoLog "プロジェクトのルートディレクトリで実行してください"
    exit 1
}

# オプションの確認
if ($Admin -and $Short) {
    Write-ErrorLog "-Adminと-Shortは同時に指定できません"
    exit 1
}

if ($Admin -and $All) {
    Write-ErrorLog "-Adminと-Allは同時に指定できません"
    exit 1
}

if ($Short -and $All) {
    Write-ErrorLog "-Shortと-Allは同時に指定できません"
    exit 1
}

# デフォルトは短時間テスト
if (-not $Admin -and -not $Short -and -not $All) {
    $Short = $true
}

Write-InfoLog "Gopier テスト実行スクリプト開始"
Write-InfoLog "テストタイプ: $(if ($Short) { 'Short' } elseif ($Admin) { 'Admin' } else { 'All' })"
Write-InfoLog "自動確認: $AutoConfirm"
Write-InfoLog "ログ表示: $ShowLog"

# テスト実行
$exitCode = 0

if ($Short) {
    $exitCode = Invoke-ShortTests
} elseif ($Admin) {
    $exitCode = Invoke-AdminTests
} elseif ($All) {
    $exitCode = Invoke-AllTests
}

Write-InfoLog "テスト実行スクリプト終了 - 終了コード: $exitCode"
exit $exitCode 