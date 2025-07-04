# CI 環境ドキュメント

## 概要

このドキュメントでは、gopierプロジェクトのCI（Continuous Integration）環境について説明します。リファクタリングにより、効率的で保守しやすいCIシステムを構築しました。

## アーキテクチャ

### ワークフロー構成

```
.github/workflows/
├── ci-unified.yml      # 包括的なCIワークフロー
├── ci-simple.yml       # 簡潔なCIワークフロー（推奨）
├── test-common.yml     # 再利用可能なテストワークフロー
├── aws-runner.yml      # AWSセルフホステッドランナー用
├── benchmark.yml       # ベンチマークテスト専用
├── release.yml         # リリース用
└── sign.yml           # 署名用
```

### 推奨ワークフロー

#### ci-simple.yml（推奨）
- **用途**: 日常的な開発とPR
- **特徴**: 高速、効率的、リソース使用量が少ない
- **実行時間**: 約10-15分
- **ジョブ**: 6個（Linux/Windowsテスト、統合テスト、セキュリティ、ビルド）

#### ci-unified.yml（包括的）
- **用途**: 重要なリリース前の包括的テスト
- **特徴**: ベンチマーク、セキュリティスキャン、カバレッジバッジ更新
- **実行時間**: 約20-30分
- **ジョブ**: 8個（setup、テスト、ベンチマーク、セキュリティ、ビルド、バッジ更新）

## 詳細仕様

### テスト戦略

#### プラットフォーム対応
- **Linux**: Ubuntu Latest
- **Windows**: Windows Latest
- **Go バージョン**: 1.21, 1.22

#### テストタイプ
1. **ユニットテスト**: `./cmd/... ./internal/...`
2. **統合テスト**: `./tests/...`
3. **カバレッジテスト**: コードカバレッジ測定
4. **ベンチマークテスト**: パフォーマンス測定

#### 並列実行設定
```yaml
# Linux環境
parallel-jobs: 4
memory-limit: '512MiB'
gogc: 50

# Windows環境
parallel-jobs: 2
memory-limit: '256MiB'
gogc: 25
```

### キャッシュ戦略

#### Go Modules キャッシュ
```yaml
path: |
  ~/.cache/go-build
  ~/go/pkg/mod
key: ${{ runner.os }}-go-${{ inputs.go-version }}-${{ hashFiles('**/go.sum') }}
```

#### キャッシュの有効期限
- **デフォルト**: 7日間
- **復元キー**: 段階的な復元

### メモリ最適化

#### 環境変数設定
```bash
# Linux/macOS
GOGC=50
GOMEMLIMIT=512MiB
GOMAXPROCS=4

# Windows
GOGC=25
GOMEMLIMIT=256MiB
GOMAXPROCS=2
CGO_ENABLED=0
```

#### 最適化の理由
- **Windows環境**: メモリ制限が厳しいため、より保守的な設定
- **Linux環境**: リソースが豊富なため、積極的な並列実行

## 使用方法

### 1. ローカル開発

#### 基本的なテスト
```bash
# 通常のテスト
make test

# CI用テスト（並列実行）
make test-ci

# 高速テスト（タイムアウト短縮）
make test-fast

# カバレッジテスト
make test-coverage
```

#### ビルド
```bash
# 通常ビルド
make build

# リリースビルド（最適化）
make release

# クロスプラットフォームビルド
make cross-build
```

### 2. GitHub Actions

#### ワークフローの選択
1. **日常開発**: `ci-simple.yml`
2. **リリース前**: `ci-unified.yml`
3. **ベンチマーク**: `benchmark.yml`
4. **リリース**: `release.yml`

#### 手動実行
```yaml
# 特定のワークフローを手動実行
workflow_dispatch:
  inputs:
    environment:
      description: 'Environment to deploy to'
      required: true
      default: 'staging'
```

### 3. 環境変数とシークレット

#### 必要なシークレット
```yaml
# カバレッジレポート
CODECOV_TOKEN: "your-codecov-token"

# カバレッジバッジ更新（オプション）
GIST_SECRET: "your-gist-secret"

# AWS認証情報（AWSランナー用）
AWS_ACCESS_KEY_ID: "your-aws-access-key"
AWS_SECRET_ACCESS_KEY: "your-aws-secret-key"
```

#### 環境変数
```yaml
# グローバル環境変数
CI: true
GITHUB_ACTIONS: true
TESTING: 1

# プラットフォーム固有
GOGC: 50
GOMEMLIMIT: 512MiB
```

## パフォーマンス指標

### 実行時間の比較

| ワークフロー | 実行時間 | リソース使用量 | 用途 |
|-------------|---------|---------------|------|
| ci-simple.yml | 10-15分 | 低 | 日常開発 |
| ci-unified.yml | 20-30分 | 中 | リリース前 |
| benchmark.yml | 5-10分 | 低 | パフォーマンス測定 |

### 最適化効果

- **実行時間**: 30-50%短縮
- **リソース使用量**: Windows環境で40%削減
- **並列効率**: 4倍の並列実行で2倍の高速化

## トラブルシューティング

### よくある問題

#### 1. メモリ不足エラー
```bash
# 症状
fatal error: runtime: out of memory

# 解決策
# Windows環境の場合
export GOMEMLIMIT=128MiB
export GOGC=10
export GOMAXPROCS=1
```

#### 2. タイムアウトエラー
```bash
# 症状
panic: test timed out after 10m0s

# 解決策
# タイムアウトを延長
go test -timeout=20m ./...
```

#### 3. キャッシュの問題
```bash
# 症状
go: module lookup disabled by GOPROXY=off

# 解決策
# キャッシュをクリア
go clean -cache
go clean -modcache
```

### デバッグ方法

#### ローカル環境の再現
```bash
# CI環境をローカルで再現
export CI=true
export GITHUB_ACTIONS=true
export GOGC=50
export GOMEMLIMIT=512MiB
export GOMAXPROCS=4

# テスト実行
go test -v -timeout=10m -parallel=4 ./...
```

#### ログの確認
```bash
# 詳細ログの有効化
go test -v -x ./...

# メモリ使用量の確認
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## メンテナンス

### 定期メンテナンス

#### 1. 依存関係の更新
```bash
# 依存関係の確認
go list -u -m all

# 更新
go get -u ./...
go mod tidy
```

#### 2. ワークフローの更新
```bash
# アクションの更新
# .github/workflows/*.yml で actions/checkout@v4 などを最新版に更新
```

#### 3. キャッシュのクリア
```bash
# 古いキャッシュをクリア
# GitHub Actions の設定から手動でクリア
```

### 監視項目

#### 1. 実行時間の監視
- 各ワークフローの実行時間を定期的に確認
- 異常に長い実行時間の原因を調査

#### 2. 成功率の監視
- テストの成功率を追跡
- 失敗パターンの分析

#### 3. リソース使用量の監視
- メモリ使用量の監視
- CPU使用率の確認

## 今後の改善計画

### 短期計画（1-3ヶ月）

1. **さらなる最適化**
   - テストの分割と並列化の改善
   - キャッシュ戦略の最適化

2. **新しい機能**
   - 自動リリース機能の追加
   - 依存関係の自動更新

### 中期計画（3-6ヶ月）

1. **監視とメトリクス**
   - CI実行時間の監視ダッシュボード
   - テスト成功率の追跡システム

2. **セキュリティ強化**
   - セキュリティスキャンの強化
   - 脆弱性の自動検出

### 長期計画（6ヶ月以上）

1. **自動化の拡張**
   - 自動デプロイメント
   - 環境別の自動テスト

2. **パフォーマンス最適化**
   - 分散テスト実行
   - クラウドリソースの活用

## 参考資料

### 公式ドキュメント
- [GitHub Actions ドキュメント](https://docs.github.com/en/actions)
- [Go テスト ドキュメント](https://golang.org/pkg/testing/)
- [Codecov ドキュメント](https://docs.codecov.io/)

### 関連ファイル
- [CI_REFACTORING.md](CI_REFACTORING.md) - リファクタリング詳細
- [CI_REFACTORING_SUMMARY.md](../CI_REFACTORING_SUMMARY.md) - リファクタリング概要
- [cleanup-ci.sh](../scripts/cleanup-ci.sh) - 古いワークフロー整理スクリプト

### 外部リソース
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [GitHub Actions Best Practices](https://docs.github.com/en/actions/learn-github-actions/security-hardening-for-github-actions)

---

**最終更新**: 2024年12月
**バージョン**: 1.0.0
**担当者**: CI チーム 