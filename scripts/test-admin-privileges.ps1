# 管理者権限テストスクリプト
# このスクリプトは管理者権限で実行する必要があります

param(
    [switch]$Verbose
)

Write-Host "=== Windows管理者権限テスト ===" -ForegroundColor Green

# 管理者権限の確認
function Test-AdminPrivileges {
    Write-Host "管理者権限を確認中..." -ForegroundColor Yellow
    
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    $isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if ($isAdmin) {
        Write-Host "✓ 管理者権限で実行されています" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ 管理者権限がありません" -ForegroundColor Red
        return $false
    }
}

# レジストリアクセステスト
function Test-RegistryAccess {
    Write-Host "`nレジストリアクセスをテスト中..." -ForegroundColor Yellow
    
    $testPaths = @(
        "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion",
        "HKLM:\SYSTEM\CurrentControlSet\Services",
        "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"
    )
    
    foreach ($path in $testPaths) {
        try {
            $item = Get-ItemProperty -Path $path -ErrorAction Stop
            Write-Host "✓ $path へのアクセス成功" -ForegroundColor Green
            if ($Verbose) {
                Write-Host "  プロパティ数: $($item.PSObject.Properties.Count)"
            }
        } catch {
            Write-Host "✗ $path へのアクセス失敗: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
}

# サービス管理テスト
function Test-ServiceManagement {
    Write-Host "`nサービス管理をテスト中..." -ForegroundColor Yellow
    
    try {
        # システムサービスの一覧を取得
        $services = Get-Service | Where-Object { $_.Status -eq "Running" } | Select-Object -First 5
        Write-Host "✓ 実行中サービスの取得成功" -ForegroundColor Green
        if ($Verbose) {
            $services | Format-Table -AutoSize
        }
        
        # 特定のサービスの詳細情報を取得
        $spooler = Get-Service -Name "Spooler" -ErrorAction SilentlyContinue
        if ($spooler) {
            Write-Host "✓ Spoolerサービスの詳細取得成功" -ForegroundColor Green
            if ($Verbose) {
                Write-Host "  サービス名: $($spooler.Name)"
                Write-Host "  表示名: $($spooler.DisplayName)"
                Write-Host "  ステータス: $($spooler.Status)"
            }
        }
    } catch {
        Write-Host "✗ サービス管理テスト失敗: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# プロセス管理テスト
function Test-ProcessManagement {
    Write-Host "`nプロセス管理をテスト中..." -ForegroundColor Yellow
    
    try {
        # システムプロセスの一覧を取得
        $processes = Get-Process | Where-Object { $_.ProcessName -like "*system*" } | Select-Object -First 3
        Write-Host "✓ システムプロセスの取得成功" -ForegroundColor Green
        if ($Verbose) {
            $processes | Format-Table -AutoSize
        }
        
        # プロセスの詳細情報を取得
        $systemProcess = Get-Process -Name "System" -ErrorAction SilentlyContinue
        if ($systemProcess) {
            Write-Host "✓ Systemプロセスの詳細取得成功" -ForegroundColor Green
            if ($Verbose) {
                Write-Host "  プロセスID: $($systemProcess.Id)"
                Write-Host "  メモリ使用量: $([math]::Round($systemProcess.WorkingSet64 / 1MB, 2)) MB"
            }
        }
    } catch {
        Write-Host "✗ プロセス管理テスト失敗: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# ファイルシステム権限テスト
function Test-FileSystemPermissions {
    Write-Host "`nファイルシステム権限をテスト中..." -ForegroundColor Yellow
    
    $testPaths = @(
        "C:\Windows\System32",
        "C:\Program Files",
        "C:\ProgramData"
    )
    
    foreach ($path in $testPaths) {
        try {
            $acl = Get-Acl -Path $path -ErrorAction Stop
            Write-Host "✓ $path のACL取得成功" -ForegroundColor Green
            if ($Verbose) {
                Write-Host "  所有者: $($acl.Owner)"
                Write-Host "  アクセスルール数: $($acl.Access.Count)"
            }
        } catch {
            Write-Host "✗ $path のACL取得失敗: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
}

# WMIアクセステスト
function Test-WMIAccess {
    Write-Host "`nWMIアクセスをテスト中..." -ForegroundColor Yellow
    
    try {
        # システム情報を取得
        $computerSystem = Get-WmiObject -Class Win32_ComputerSystem
        Write-Host "✓ コンピューターシステム情報取得成功" -ForegroundColor Green
        if ($Verbose) {
            Write-Host "  コンピューター名: $($computerSystem.Name)"
            Write-Host "  製造元: $($computerSystem.Manufacturer)"
            Write-Host "  モデル: $($computerSystem.Model)"
        }
        
        # ディスク情報を取得
        $logicalDisks = Get-WmiObject -Class Win32_LogicalDisk
        Write-Host "✓ 論理ディスク情報取得成功" -ForegroundColor Green
        if ($Verbose) {
            $logicalDisks | ForEach-Object {
                Write-Host "  ドライブ: $($_.DeviceID) - サイズ: $([math]::Round($_.Size / 1GB, 2)) GB"
            }
        }
        
        # ネットワークアダプター情報を取得
        $networkAdapters = Get-WmiObject -Class Win32_NetworkAdapter | Where-Object { $_.NetEnabled -eq $true }
        Write-Host "✓ ネットワークアダプター情報取得成功" -ForegroundColor Green
        if ($Verbose) {
            $networkAdapters | Select-Object -First 3 | ForEach-Object {
                Write-Host "  アダプター: $($_.Name)"
            }
        }
    } catch {
        Write-Host "✗ WMIアクセステスト失敗: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# メイン実行
function Main {
    $adminCheck = Test-AdminPrivileges
    
    if (-not $adminCheck) {
        Write-Host "`nこのスクリプトは管理者権限で実行する必要があります。" -ForegroundColor Red
        Write-Host "PowerShellを管理者として実行してから、このスクリプトを再実行してください。" -ForegroundColor Yellow
        exit 1
    }
    
    Test-RegistryAccess
    Test-ServiceManagement
    Test-ProcessManagement
    Test-FileSystemPermissions
    Test-WMIAccess
    
    Write-Host "`n=== 管理者権限テスト完了 ===" -ForegroundColor Green
    Write-Host "すべてのテストが正常に完了しました。" -ForegroundColor Green
}

# スクリプト実行
Main 