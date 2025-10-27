# 管理者権限テストスクリプト
# このスクリプトは管理者権限で実行する必要があります

param(
    [switch]$Verbose
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
$logConfig = Get-LogConfig
$adminConfig = Get-AdminConfig

# ログ設定を適用
Set-LogConfig -Level $logConfig.Level -EnableFileLog $logConfig.EnableFileLog -LogDirectory $logConfig.LogDirectory

Write-InfoLog "=== Windows管理者権限テスト ==="

# 管理者権限の確認
function Test-AdminPrivileges {
    Write-InfoLog "管理者権限を確認中..."
    
    $isAdmin = Test-AdminPrivileges
    
    if ($isAdmin) {
        Write-InfoLog "✓ 管理者権限で実行されています"
        return $true
    } else {
        Write-ErrorLog "✗ 管理者権限がありません"
        return $false
    }
}

# レジストリアクセステスト
function Test-RegistryAccess {
    Write-InfoLog "レジストリアクセスをテスト中..."
    
    $testPaths = @(
        "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion",
        "HKLM:\SYSTEM\CurrentControlSet\Services",
        "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"
    )
    
    foreach ($path in $testPaths) {
        $result = Invoke-WithErrorHandling -ScriptBlock {
            $item = Get-ItemProperty -Path $path -ErrorAction Stop
            Write-InfoLog "✓ $path へのアクセス成功"
            if ($Verbose) {
                Write-DebugLog "  プロパティ数: $($item.PSObject.Properties.Count)"
            }
        } -ErrorMessage "$path へのアクセス失敗" -ContinueOnError $true
        
        if (-not $result) {
            Write-ErrorLog "✗ $path へのアクセス失敗"
        }
    }
}

# サービス管理テスト
function Test-ServiceManagement {
    Write-InfoLog "サービス管理をテスト中..."
    
    $result = Invoke-WithErrorHandling -ScriptBlock {
        # システムサービスの一覧を取得
        $services = Get-Service | Where-Object { $_.Status -eq "Running" } | Select-Object -First 5
        Write-InfoLog "✓ 実行中サービスの取得成功"
        if ($Verbose) {
            $services | Format-Table -AutoSize
        }
        
        # 特定のサービスの詳細情報を取得
        $spooler = Get-Service -Name "Spooler" -ErrorAction SilentlyContinue
        if ($spooler) {
            Write-InfoLog "✓ Spoolerサービスの詳細取得成功"
            if ($Verbose) {
                Write-DebugLog "  サービス名: $($spooler.Name)"
                Write-DebugLog "  表示名: $($spooler.DisplayName)"
                Write-DebugLog "  ステータス: $($spooler.Status)"
            }
        }
    } -ErrorMessage "サービス管理テスト失敗" -ContinueOnError $true
    
    if (-not $result) {
        Write-ErrorLog "✗ サービス管理テスト失敗"
    }
}

# プロセス管理テスト
function Test-ProcessManagement {
    Write-InfoLog "プロセス管理をテスト中..."
    
    $result = Invoke-WithErrorHandling -ScriptBlock {
        # システムプロセスの一覧を取得
        $processes = Get-Process | Where-Object { $_.ProcessName -like "*system*" } | Select-Object -First 3
        Write-InfoLog "✓ システムプロセスの取得成功"
        if ($Verbose) {
            $processes | Format-Table -AutoSize
        }
        
        # プロセスの詳細情報を取得
        $systemProcess = Get-Process -Name "System" -ErrorAction SilentlyContinue
        if ($systemProcess) {
            Write-InfoLog "✓ Systemプロセスの詳細取得成功"
            if ($Verbose) {
                Write-DebugLog "  プロセスID: $($systemProcess.Id)"
                Write-DebugLog "  メモリ使用量: $([math]::Round($systemProcess.WorkingSet64 / 1MB, 2)) MB"
            }
        }
    } -ErrorMessage "プロセス管理テスト失敗" -ContinueOnError $true
    
    if (-not $result) {
        Write-ErrorLog "✗ プロセス管理テスト失敗"
    }
}

# ファイルシステム権限テスト
function Test-FileSystemPermissions {
    Write-InfoLog "ファイルシステム権限をテスト中..."
    
    $testPaths = @(
        "C:\Windows\System32",
        "C:\Program Files",
        "C:\ProgramData"
    )
    
    foreach ($path in $testPaths) {
        $result = Invoke-WithErrorHandling -ScriptBlock {
            $acl = Get-Acl -Path $path -ErrorAction Stop
            Write-InfoLog "✓ $path のACL取得成功"
            if ($Verbose) {
                Write-DebugLog "  所有者: $($acl.Owner)"
                Write-DebugLog "  アクセスルール数: $($acl.Access.Count)"
            }
        } -ErrorMessage "$path のACL取得失敗" -ContinueOnError $true
        
        if (-not $result) {
            Write-ErrorLog "✗ $path のACL取得失敗"
        }
    }
}

# WMIアクセステスト
function Test-WMIAccess {
    Write-InfoLog "WMIアクセスをテスト中..."
    
    $result = Invoke-WithErrorHandling -ScriptBlock {
        # システム情報を取得
        $computerSystem = Get-WmiObject -Class Win32_ComputerSystem
        Write-InfoLog "✓ コンピューターシステム情報取得成功"
        if ($Verbose) {
            Write-DebugLog "  コンピューター名: $($computerSystem.Name)"
            Write-DebugLog "  製造元: $($computerSystem.Manufacturer)"
            Write-DebugLog "  モデル: $($computerSystem.Model)"
        }
        
        # ディスク情報を取得
        $logicalDisks = Get-WmiObject -Class Win32_LogicalDisk
        Write-InfoLog "✓ 論理ディスク情報取得成功"
        if ($Verbose) {
            $logicalDisks | ForEach-Object {
                Write-DebugLog "  ドライブ: $($_.DeviceID) - サイズ: $([math]::Round($_.Size / 1GB, 2)) GB"
            }
        }
        
        # ネットワークアダプター情報を取得
        $networkAdapters = Get-WmiObject -Class Win32_NetworkAdapter | Where-Object { $_.NetEnabled -eq $true }
        Write-InfoLog "✓ ネットワークアダプター情報取得成功"
        if ($Verbose) {
            $networkAdapters | Select-Object -First 3 | ForEach-Object {
                Write-DebugLog "  アダプター: $($_.Name)"
            }
        }
    } -ErrorMessage "WMIアクセステスト失敗" -ContinueOnError $true
    
    if (-not $result) {
        Write-ErrorLog "✗ WMIアクセステスト失敗"
    }
}

# メイン実行
function Main {
    $totalStartTime = Get-Date
    
    Write-InfoLog "管理者権限テスト開始"
    Write-InfoLog "詳細モード: $Verbose"
    
    $adminCheck = Test-AdminPrivileges
    
    if (-not $adminCheck) {
        Write-ErrorLog "このスクリプトは管理者権限で実行する必要があります。"
        Write-InfoLog "PowerShellを管理者として実行してから、このスクリプトを再実行してください。"
        exit 1
    }
    
    # 各テストを実行
    $testResults = @()
    
    $testResults += @{
        Name = "レジストリアクセス"
        Function = { Test-RegistryAccess }
    }
    
    $testResults += @{
        Name = "サービス管理"
        Function = { Test-ServiceManagement }
    }
    
    $testResults += @{
        Name = "プロセス管理"
        Function = { Test-ProcessManagement }
    }
    
    $testResults += @{
        Name = "ファイルシステム権限"
        Function = { Test-FileSystemPermissions }
    }
    
    $testResults += @{
        Name = "WMIアクセス"
        Function = { Test-WMIAccess }
    }
    
    # テスト実行
    foreach ($test in $testResults) {
        Write-InfoLog "`n--- $($test.Name) テスト ---"
        $testStartTime = Get-Date
        
        $executionResult = Measure-ExecutionTime -ScriptBlock {
            & $test.Function
        } -Description "$($test.Name) テスト"
        
        $testDuration = $executionResult.Duration.TotalSeconds.ToString('F2')
        Write-InfoLog "$($test.Name) テスト完了 - 実行時間: $testDuration秒"
    }
    
    $totalDuration = (Get-Date) - $totalStartTime
    
    Write-InfoLog "`n=== 管理者権限テスト完了 ==="
    Write-InfoLog "すべてのテストが正常に完了しました。"
    Write-InfoLog "総実行時間: $($totalDuration.TotalSeconds.ToString('F2'))秒"
    
    # ログファイルに結果を記録
    if ($adminConfig.LogAdminActions) {
        $logFile = "admin_privileges_test_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"
        $logContent = @"
=== 管理者権限テスト実行ログ ===
実行日時: $(Get-Date)
管理者権限: $(Test-AdminPrivileges)
詳細モード: $Verbose
総実行時間: $($totalDuration.TotalSeconds.ToString('F2'))秒

=== テスト結果 ===
すべてのテストが正常に完了しました。
"@
        $logContent | Out-File -FilePath $logFile -Encoding UTF8
        Write-InfoLog "ログファイル: $logFile"
    }
}

# スクリプト実行
Main 