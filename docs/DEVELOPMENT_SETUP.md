# 開発環境セットアップガイド

## 概要

このドキュメントでは、Gopierプロジェクトの開発環境セットアップについて説明します。リファクタリングされたPowerShellスクリプトを使用して、効率的で保守しやすい開発環境を構築できます。

## 前提条件

### 必要なソフトウェア

- **Go**: 1.23以上
- **PowerShell**: 5.1以上（Windows）
- **Git**: 最新版
- **Make**: 最新版（オプション）

### 推奨環境

- **OS**: Windows 10/11, Linux (Ubuntu 20.04+), macOS 12+
- **メモリ**: 8GB以上
- **ディスク**: 10GB以上の空き容量

## クイックスタート

### 1. リポジトリのクローン

```bash
git clone https://github.com/sakuhanight/gopier.git
cd gopier
```

### 2. 設定ファイルの作成

```powershell
# PowerShellで実行
Copy-Item "scripts\config.example.json" "scripts\config.json"
```

### 3. 基本的なビルドとテスト

```powershell
# ビルド
.\build.ps1 build

# テスト
.\scripts\run_tests.ps1 -Short
```

## 詳細セットアップ

### PowerShellスクリプトの構成

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
└── README.md                 # スクリプト詳細説明
```

### 設定ファイルのカスタマイズ

#### config.json の主要設定

```json
{
  "Build": {
    "BinaryName": "gopier.exe",
    "BuildDir": "build",
    "EnableOptimization": true,
    "MemoryLimit": "512MiB",
    "GarbageCollection": "50"
  },
  "Test": {
    "EnableCoverage": true,
    "CoverageOutput": "coverage.out",
    "TimeoutSeconds": 300,
    "ParallelTests": true
  },
  "Log": {
    "Level": "Info",
    "EnableFileLog": true,
    "LogDirectory": "logs",
    "EnableColor": true
  }
}
```

#### 設定の説明

- **Build**: ビルド関連の設定
- **Test**: テスト関連の設定
- **Log**: ログ関連の設定
- **Platform**: プラットフォーム関連の設定
- **Admin**: 管理者権限関連の設定

## 開発ワークフロー

### 日常的な開発

#### 1. コードの変更

```bash
# 新しいブランチを作成
git checkout -b feature/new-feature

# コードを編集
# ...

# 変更をコミット
git add .
git commit -m "feat: add new feature"
```

#### 2. ローカルテスト

```powershell
# 短時間テスト（推奨）
.\scripts\run_tests.ps1 -Short

# すべてのテスト
.\scripts\run_tests.ps1 -All

# 管理者権限テスト（必要に応じて）
.\scripts\run_tests.ps1 -Admin
```

#### 3. ビルドとインストール

```powershell
# 通常ビルド
.\build.ps1 build

# リリースビルド
.\build.ps1 release

# インストール
.\build.ps1 install
```

### 高度な開発

#### クロスプラットフォームビルド

```powershell
# すべてのプラットフォーム
.\build.ps1 cross-build -Platform all

# 特定のプラットフォーム
.\build.ps1 cross-build -Platform linux -Architecture amd64
```

#### テストカバレッジ

```powershell
# カバレッジテスト実行
.\build.ps1 test-coverage

# カバレッジレポートの確認
Start-Process "coverage.html"
```

#### 管理者権限テスト

```powershell
# 管理者権限テストの実行
.\scripts\test-admin-privileges.ps1

# 詳細モード
.\scripts\test-admin-privileges.ps1 -Verbose
```

## トラブルシューティング

### よくある問題

#### 1. Goコマンドが見つからない

```powershell
# Goのインストール確認
go version

# PATHの確認
$env:PATH -split ';' | Where-Object { $_ -like "*go*" }

# Goの再インストール
# https://golang.org/dl/ からダウンロード
```

#### 2. PowerShell実行ポリシーの問題

```powershell
# 実行ポリシーの確認
Get-ExecutionPolicy

# 実行ポリシーの変更（管理者権限が必要）
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### 3. モジュールの依存関係エラー

```powershell
# モジュールの整理
go mod tidy

# 依存関係のダウンロード
go mod download

# キャッシュのクリア
go clean -cache
```

#### 4. メモリ不足エラー

```powershell
# 設定ファイルでメモリ制限を調整
# scripts/config.json の Build.MemoryLimit を変更
{
  "Build": {
    "MemoryLimit": "256MiB"
  }
}
```

#### 5. 管理者権限が必要なテストの失敗

```powershell
# PowerShellを管理者として実行
# または、管理者権限テストをスキップ
.\scripts\run_tests.ps1 -Short
```

### ログの確認

```powershell
# ログディレクトリの確認
Get-ChildItem "logs" -Filter "*.log" | Sort-Object LastWriteTime -Descending

# 最新のログを表示
Get-Content "logs\gopier_$(Get-Date -Format 'yyyyMMdd').log" -Tail 50
```

## パフォーマンス最適化

### ビルド最適化

#### 環境変数の設定

```powershell
# メモリ最適化
$env:GOGC = "50"
$env:GOMEMLIMIT = "512MiB"

# 並列処理の最適化
$env:GOMAXPROCS = "4"
```

#### キャッシュの活用

```powershell
# Goモジュールキャッシュの確認
go env GOMODCACHE

# ビルドキャッシュの確認
go env GOCACHE
```

### テスト最適化

#### 並列テストの設定

```powershell
# 設定ファイルで並列テストを有効化
{
  "Test": {
    "ParallelTests": true,
    "TimeoutSeconds": 300
  }
}
```

#### テストの分割実行

```powershell
# 特定のパッケージのみテスト
go test ./internal/copier/...

# 短時間テストのみ
go test -short ./...
```

## CI/CD統合

### GitHub Actions

#### ローカルでのCIテスト

```powershell
# CI用テスト実行
.\scripts\run_tests.ps1 -All

# カバレッジテスト
.\build.ps1 test-coverage
```

#### ワークフローの確認

```yaml
# .github/workflows/ci-simple.yml を使用
# 自動的に実行されます
```

### ローカルCI環境

#### Docker環境でのテスト

```bash
# Dockerfile.dev を使用
docker build -f Dockerfile.dev -t gopier-dev .

# コンテナ内でテスト実行
docker run --rm gopier-dev ./scripts/run_tests.ps1 -All
```

## 開発ツール

### 推奨IDE設定

#### VS Code

```json
{
  "go.toolsManagement.checkForUpdates": "local",
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports",
  "files.associations": {
    "*.ps1": "powershell"
  }
}
```

#### GoLand

- Go modules を有効化
- PowerShell プラグインをインストール
- コードインスペクションを有効化

### デバッグ設定

#### VS Code デバッグ設定

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Package",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/main.go"
    }
  ]
}
```

## ベストプラクティス

### コード品質

1. **テスト駆動開発**: 新機能は必ずテストを作成
2. **コードレビュー**: プルリクエストでレビューを依頼
3. **静的解析**: `golangci-lint` を使用
4. **フォーマット**: `go fmt` でコードを整形

### パフォーマンス

1. **プロファイリング**: `go test -bench` でベンチマーク
2. **メモリ監視**: `go test -memprofile` でメモリ使用量を監視
3. **CPU監視**: `go test -cpuprofile` でCPU使用量を監視

### セキュリティ

1. **依存関係の更新**: `go mod tidy` で定期的に更新
2. **セキュリティスキャン**: `govulncheck` を使用
3. **権限の最小化**: 必要最小限の権限で実行

## 参考資料

- [Go公式ドキュメント](https://golang.org/doc/)
- [PowerShell公式ドキュメント](https://docs.microsoft.com/powershell/)
- [GitHub Actionsドキュメント](https://docs.github.com/actions)
- [プロジェクトREADME](../README.md)
- [CI環境ドキュメント](CI_ENVIRONMENT.md)

---

**最終更新**: 2025/07/06 