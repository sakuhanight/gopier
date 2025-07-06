# Gopier PowerShell スクリプト

このディレクトリには、GopierプロジェクトのPowerShellスクリプトが含まれています。

## ディレクトリ構造

```
scripts/
├── common/                    # 共通モジュール
│   ├── Logger.ps1            # ログ機能
│   ├── Utils.ps1             # ユーティリティ機能
│   ├── Config.ps1            # 設定管理
│   └── GopierCommon.psm1     # メインモジュール
├── run_tests.ps1             # テスト実行スクリプト
├── test-admin-privileges.ps1 # 管理者権限テストスクリプト
├── config.example.json       # 設定ファイル例
└── README.md                 # このファイル
```

## 共通モジュール

### Logger.ps1
統一されたログ出力とエラーハンドリングを提供します。

**主な機能:**
- ログレベルの管理（Debug, Info, Warning, Error, Fatal）
- 色付きコンソール出力
- ファイルログ出力
- エラーハンドリング関数

**使用例:**
```powershell
Write-InfoLog "情報メッセージ"
Write-ErrorLog "エラーメッセージ"
Write-WarningLog "警告メッセージ"
```

### Utils.ps1
プロジェクト全体で使用される共通機能を提供します。

**主な機能:**
- 管理者権限チェック
- Goコマンド確認
- プロジェクトルート検出
- ファイルサイズフォーマット
- 実行時間計測
- 環境変数管理
- プロセス実行

**使用例:**
```powershell
if (Test-AdminPrivileges) {
    Write-InfoLog "管理者権限で実行中"
}

$executionResult = Measure-ExecutionTime -ScriptBlock {
    # 実行したい処理
} -Description "処理の説明"
```

### Config.ps1
プロジェクト全体の設定を一元管理します。

**主な機能:**
- JSON設定ファイルの読み込み/保存
- 設定の検証
- デフォルト設定の管理
- 設定セクション別の取得

**使用例:**
```powershell
$buildConfig = Get-BuildConfig
$testConfig = Get-TestConfig
$logConfig = Get-LogConfig
```

### GopierCommon.psm1
上記の共通モジュールを統合するメインモジュールです。

**使用方法:**
```powershell
Import-Module "scripts\common\GopierCommon.psm1" -Force
```

## スクリプト

### build.ps1
プロジェクトのビルド、テスト、リリース用のメインスクリプトです。

**使用方法:**
```powershell
# 通常ビルド
.\build.ps1 build

# リリースビルド
.\build.ps1 release

# クロスプラットフォームビルド
.\build.ps1 cross-build -Platform all

# テスト実行
.\build.ps1 test

# テストカバレッジ
.\build.ps1 test-coverage
```

### run_tests.ps1
テスト実行を管理するスクリプトです。

**使用方法:**
```powershell
# 短時間テスト（管理者権限不要）
.\scripts\run_tests.ps1 -Short

# 管理者権限テスト
.\scripts\run_tests.ps1 -Admin

# すべてのテスト
.\scripts\run_tests.ps1 -All

# ログ表示付き
.\scripts\run_tests.ps1 -Admin -ShowLog
```

### test-admin-privileges.ps1
管理者権限のテストを実行するスクリプトです。

**使用方法:**
```powershell
# 基本テスト
.\scripts\test-admin-privileges.ps1

# 詳細モード
.\scripts\test-admin-privileges.ps1 -Verbose
```

## 設定

### config.json
プロジェクトの設定を管理するJSONファイルです。

**設定項目:**
- **Build**: ビルド関連の設定
- **Test**: テスト関連の設定
- **Log**: ログ関連の設定
- **Platform**: プラットフォーム関連の設定
- **Admin**: 管理者権限関連の設定

**設定ファイルの作成:**
```powershell
# 設定ファイル例をコピー
Copy-Item "scripts\config.example.json" "scripts\config.json"

# 設定を編集
notepad "scripts\config.json"
```

## リファクタリングの改善点

### 1. 共通機能の分離
- 重複していた機能を共通モジュールに分離
- コードの再利用性が向上
- 保守性が向上

### 2. 統一されたログ機能
- すべてのスクリプトで統一されたログ出力
- ログレベルの管理
- ファイルログとコンソールログの両方に対応

### 3. 設定の一元管理
- JSONファイルによる設定管理
- デフォルト設定の提供
- 設定の検証機能

### 4. エラーハンドリングの改善
- 統一されたエラーハンドリング方式
- より詳細なエラー情報
- エラーの継続処理オプション

### 5. 実行時間の計測
- 各処理の実行時間を計測
- パフォーマンスの可視化
- ボトルネックの特定

### 6. 環境変数の安全な管理
- 環境変数の安全な設定/取得
- デフォルト値の提供
- 一時的/永続的な設定の選択

## 使用方法

### 初回セットアップ
1. 設定ファイルを作成
```powershell
Copy-Item "scripts\config.example.json" "scripts\config.json"
```

2. 設定を必要に応じて編集
```powershell
notepad "scripts\config.json"
```

### 日常的な使用
```powershell
# ビルド
.\build.ps1 build

# テスト
.\scripts\run_tests.ps1 -Short

# 管理者権限テスト
.\scripts\run_tests.ps1 -Admin
```

## トラブルシューティング

### 共通モジュールが読み込めない
```powershell
# モジュールパスを確認
Get-Module -ListAvailable | Where-Object { $_.Name -like "*Gopier*" }

# 手動でモジュールを読み込み
Import-Module ".\scripts\common\GopierCommon.psm1" -Force
```

### 設定ファイルが見つからない
```powershell
# 設定ファイルの存在確認
Test-Path "scripts\config.json"

# 設定ファイル例から作成
Copy-Item "scripts\config.example.json" "scripts\config.json"
```

### ログファイルが生成されない
```powershell
# ログディレクトリの確認
Test-Path "logs"

# ログ設定の確認
Get-LogConfig
```

## 開発者向け情報

### 新しいスクリプトの作成
1. 共通モジュールを読み込み
```powershell
Import-Module "scripts\common\GopierCommon.psm1" -Force
```

2. 設定を取得
```powershell
$config = Get-ProjectConfig
```

3. ログ機能を使用
```powershell
Write-InfoLog "処理開始"
Write-ErrorLog "エラーが発生"
```

### 新しい共通機能の追加
1. 適切なモジュールファイルに機能を追加
2. `GopierCommon.psm1`の`Export-ModuleMember`に追加
3. ドキュメントを更新

### テスト
```powershell
# 短時間テスト
.\scripts\run_tests.ps1 -Short

# 管理者権限テスト
.\scripts\run_tests.ps1 -Admin

# すべてのテスト
.\scripts\run_tests.ps1 -All
``` 