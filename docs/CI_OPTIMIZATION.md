# CI最適化ガイド

## 概要

このドキュメントでは、GopierプロジェクトのCI最適化について説明します。PowerShellスクリプトのリファクタリングと統合により、効率的で保守しやすいCIシステムを構築しました。

## 最適化の背景

### 元の問題点
1. **Windowsビルドでのタイムアウト**: 大きなテストスイートによる実行時間の超過
2. **メモリ使用量の過多**: Windows環境でのメモリ制限による失敗
3. **重複したテスト実行**: 効率的でないテスト戦略
4. **保守性の低さ**: 分散した設定とスクリプト

### 解決策
1. **PowerShellスクリプトのリファクタリング**: 共通モジュールによる統一
2. **テストの分割実行**: 効率的な並列実行
3. **メモリ最適化**: 環境に応じた設定調整
4. **設定の一元化**: JSONファイルによる管理

## ワークフロー構成

### 1. 標準CIワークフロー (`ci-simple.yml`)
- **用途**: 日常的な開発とPR
- **特徴**: 高速、効率的、リソース使用量が少ない
- **PowerShell統合**: リファクタリングされたスクリプトを使用

### 2. 包括的CIワークフロー (`ci-unified.yml`)
- **用途**: 重要なリリース前の包括的テスト
- **特徴**: ベンチマーク、セキュリティスキャン、カバレッジバッジ更新
- **PowerShell統合**: 設定ファイルによる最適化

### 3. ベンチマークテストワークフロー (`benchmark.yml`)
- **用途**: パフォーマンス測定専用
- **特徴**: メモリ使用量を増加させてベンチマーク実行
- **結果保存**: アーティファクトとして保存

### 4. AWSセルフホステッドランナーワークフロー (`aws-runner.yml`)
- **用途**: 大きなファイルテスト専用
- **特徴**: AWS EC2インスタンスで実行、コスト最適化
- **PowerShell統合**: リモート環境でのスクリプト実行

## PowerShellスクリプト統合

### 共通モジュールの活用

```powershell
# 共通モジュールの読み込み
Import-Module "scripts\common\GopierCommon.psm1" -Force

# 設定の取得
$buildConfig = Get-BuildConfig
$testConfig = Get-TestConfig
$logConfig = Get-LogConfig
```

### 設定ファイルによる最適化

```json
{
  "Build": {
    "BinaryName": "gopier.exe",
    "BuildDir": "build",
    "EnableOptimization": true,
    "MemoryLimit": "256MiB",
    "GarbageCollection": "25"
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

## メモリ最適化設定

### 環境変数

#### 通常テスト用
```bash
GOGC=50              # ガベージコレクション頻度を上げる
GOMEMLIMIT=256MiB    # メモリ使用量を制限
GOMAXPROCS=2         # 並行処理を制限
CGO_ENABLED=0        # CGOを無効化
GOFLAGS="-buildvcs=false"  # VCS情報を無効化
```

#### PowerShell設定
```powershell
# 設定ファイルによる最適化
{
  "Build": {
    "MemoryLimit": "256MiB",
    "GarbageCollection": "25"
  }
}
```

#### ベンチマークテスト用
```bash
GOGC=100             # ガベージコレクション頻度を下げる
GOMEMLIMIT=1GiB      # メモリ使用量を増加
GOMAXPROCS=4         # 並行処理を増加
BENCHMARK_MODE=true  # ベンチマークモード
```

#### AWSセルフホステッドランナー用
```bash
GOGC=100             # ガベージコレクション頻度を下げる
GOMEMLIMIT=4GiB      # 大きなメモリ使用量
GOMAXPROCS=8         # 並行処理を最大
AWS_RUNNER=true      # AWSランナーモード
```

### テスト実行オプション

#### PowerShellスクリプトによる実行
```powershell
# 短時間テスト
.\scripts\run_tests.ps1 -Short

# すべてのテスト
.\scripts\run_tests.ps1 -All

# 管理者権限テスト
.\scripts\run_tests.ps1 -Admin
```

#### 通常テスト
```bash
go test -v -timeout=8m -parallel=2 ./package/...
```

#### ベンチマークテスト
```bash
go test -v -bench=. -benchmem -timeout=15m ./package/...
```

## 使用方法

### 1. 標準ワークフローを使用
```yaml
# .github/workflows/ci-simple.yml を使用
# 自動的に実行されます
```

### 2. PowerShellスクリプトによるローカルテスト
```powershell
# 短時間テスト（推奨）
.\scripts\run_tests.ps1 -Short

# すべてのテスト
.\scripts\run_tests.ps1 -All

# カバレッジテスト
.\build.ps1 test-coverage
```

### 3. 手動でWindows最適化ワークフローを実行
```yaml
# 特定のパッケージのみテスト
uses: ./.github/workflows/windows-optimization.yml
with:
  go-version: '1.21'
  test-packages: './internal/copier/...'
  timeout-minutes: 15
```

### 4. ベンチマークテストを実行
```yaml
# ベンチマークテスト専用
uses: ./.github/workflows/benchmark.yml
with:
  go-version: '1.21'
  platform: 'windows-latest'
  timeout-minutes: 35
```

### 5. AWSセルフホステッドランナーで大きなファイルテストを実行
```yaml
# 大きなファイルテスト専用
uses: ./.github/workflows/aws-runner.yml
with:
  go-version: '1.21'
  test-type: 'large-files'
  timeout-minutes: 60
```

## パフォーマンス改善

### 実行時間の短縮
- **従来**: 45分（タイムアウト）
- **最適化後**: 15分以内（各ジョブ）
- **PowerShell統合後**: 10分以内（効率的なスクリプト実行）

### メモリ使用量の削減
- **従来**: 512MB以上
- **最適化後**: 256MB以下（通常テスト）、1GB（ベンチマークテスト）、4GB（AWSランナー）
- **PowerShell統合後**: 設定ファイルによる動的調整

### 並行実行による効率化
- 4つのWindowsテストジョブを並行実行
- ベンチマークテストを独立して実行
- AWSセルフホステッドランナーで大きなファイルテストを実行
- PowerShellスクリプトによる効率的なテスト管理

## トラブルシューティング

### 1. メモリ不足エラー
```powershell
# 設定ファイルでメモリ制限を調整
{
  "Build": {
    "MemoryLimit": "128MiB"
  }
}

# または環境変数で直接設定
$env:GOMEMLIMIT = "128MiB"
$env:GOGC = "10"
```

### 2. タイムアウトエラー
```powershell
# 設定ファイルでタイムアウトを調整
{
  "Test": {
    "TimeoutSeconds": 600
  }
}
```

### 3. PowerShell実行エラー
```powershell
# 実行ポリシーの確認と変更
Get-ExecutionPolicy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### 4. テスト失敗
```powershell
# 特定のテストのみ実行
go test -v -run TestSpecific ./package/...

# ログの確認
Get-ChildItem "logs" -Filter "*.log" | Sort-Object LastWriteTime -Descending
```

## 監視とログ

### メモリ使用量の監視
PowerShellスクリプトによる監視：
- システムメモリ情報
- プロセスメモリ使用量
- ディスク容量
- モジュールキャッシュサイズ

### ログ出力
```powershell
# ログファイルの確認
Get-ChildItem "logs" -Filter "*.log" | Sort-Object LastWriteTime -Descending

# 最新のログを表示
Get-Content "logs\gopier_$(Get-Date -Format 'yyyyMMdd').log" -Tail 50
```

### パフォーマンスメトリクス
- テスト実行時間の計測
- メモリ使用量の追跡
- 成功率の監視
- キャッシュヒット率の確認

## 今後の改善案

### 短期計画（1-3ヶ月）
1. **テストデータの最適化**: 大きなテストファイルの削減
2. **キャッシュ戦略の改善**: より効率的なキャッシュ利用
3. **段階的ビルド**: 依存関係の段階的ダウンロード

### 中期計画（3-6ヶ月）
1. **分散テスト**: 複数のランナーでの分散実行
2. **自動スケーリング**: 負荷に応じたリソース自動調整
3. **セキュリティ強化**: セキュリティスキャンの自動化

### 長期計画（6ヶ月以上）
1. **AI支援テスト**: 機械学習によるテスト最適化
2. **予測分析**: テスト失敗の予測と予防
3. **統合開発環境**: IDEとの統合強化

## ベストプラクティス

### PowerShellスクリプト
1. **共通モジュールの活用**: 重複コードの排除
2. **設定ファイルの使用**: 環境に応じた動的調整
3. **エラーハンドリング**: 統一されたエラー処理
4. **ログ出力**: 詳細なログ記録

### CI/CD
1. **ワークフローの選択**: 用途に応じた適切な選択
2. **キャッシュの活用**: 効率的なキャッシュ戦略
3. **並列実行**: 適切な並列度の設定
4. **モニタリング**: 継続的なパフォーマンス監視

## 参考資料

- [CI環境ドキュメント](CI_ENVIRONMENT.md)
- [開発環境セットアップガイド](DEVELOPMENT_SETUP.md)
- [PowerShellスクリプトドキュメント](../scripts/README.md)
- [GitHub Actions の制限](https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#supported-runners-and-hardware-resources)
- [Go メモリ最適化](https://golang.org/doc/gc-guide)
- [Windows ランナーの仕様](https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#supported-software)

---

**最終更新**: 2025/07/06 