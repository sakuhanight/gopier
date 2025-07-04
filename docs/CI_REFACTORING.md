# CI 改善

## 概要

このドキュメントは、gopierプロジェクトのCI（Continuous Integration）システムの改善について説明します。

## 変更内容

### 1. 新しいワークフローファイル

#### `ci.yml`
- **統合CIワークフロー**: すべてのCI処理を1つのファイルに統合
- **マトリックス戦略**: 効率的なテスト実行のためのマトリックス戦略を採用
- **依存関係の最適化**: 共通のセットアップジョブで依存関係を事前ダウンロード
- **セキュリティスキャン**: 包括的なセキュリティチェックを追加

#### `test.yml`
- **再利用可能なテストワークフロー**: 他のワークフローから呼び出し可能
- **柔軟な設定**: テストタイプ、カバレッジ、タイムアウトを設定可能
- **プラットフォーム対応**: Linux/macOS/Windows対応

#### `benchmark.yml`
- **効率的なベンチマーク**: 軽量で高速なベンチマーク実行
- **詳細レポート**: ベンチマーク結果の詳細分析
- **アーティファクト保存**: 結果の永続化

#### `security.yml`
- **多層セキュリティ**: Nancy、govulncheck、gosecを使用
- **依存関係監査**: 脆弱性の包括的チェック
- **レポート生成**: セキュリティレポートの自動生成

#### `lint.yml`
- **コード品質チェック**: 複数のリンティングツールを使用
- **フォーマットチェック**: gofmt、goimportsによる一貫性確保
- **複雑度チェック**: サイクロマチック複雑度の監視

### 2. Makefileの改善

#### 改善されたターゲット
- `test-ci`: CI用テスト（並列実行・カバレッジ付き）
- カバレッジレポート生成
- より詳細なエラーメッセージ

## 改善点

### 1. パフォーマンス向上
- **効率的なキャッシュ戦略**: 共通のキャッシュキーを使用
- **並列実行**: マトリックス戦略による並列処理
- **メモリ最適化**: Windows環境でのメモリ制限を緩和

### 2. 保守性向上
- **モジュラー設計**: 再利用可能なワークフロー
- **設定の一元化**: 環境変数とパラメータの統一
- **ドキュメント化**: 各ワークフローの目的と使用方法を明確化

### 3. セキュリティ強化
- **多層防御**: 複数のセキュリティツールを使用
- **自動脆弱性チェック**: 依存関係の自動監査
- **セキュリティレポート**: 詳細なセキュリティ分析

### 4. 品質向上
- **包括的テスト**: ユニットテスト、統合テスト、ベンチマーク
- **コード品質**: 複数のリンティングツールによる品質チェック
- **カバレッジ追跡**: 詳細なカバレッジレポート

## 使用方法

### 1. 新しいCIワークフローの実行

```bash
# CI用テストを実行（カバレッジ付き）
make test-ci
```

### 2. 個別ワークフローの実行

各ワークフローは独立して実行可能です：

```yaml
# test.ymlの使用例
- name: Run tests
  uses: ./.github/workflows/test.yml
  with:
    go-version: '1.21'
    platform: 'self-hosted'
    test-type: 'unit'
    coverage: true
    timeout-minutes: 20
```

### 3. セキュリティスキャンの実行

```yaml
# security.ymlの使用例
- name: Security scan
  uses: ./.github/workflows/security.yml
  with:
    go-version: '1.21'
    platform: 'self-hosted'
    timeout-minutes: 15
```

## 設定

### 1. 環境変数

```yaml
env:
  CI: true
  GITHUB_ACTIONS: true
  TESTING: 1
  GOGC: 100          # ガベージコレクション最適化
  GOMAXPROCS: 4      # 並列処理数
```

### 2. キャッシュ設定

```yaml
- name: Cache Go modules
  uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ inputs.go-version }}-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-${{ inputs.go-version }}-
      ${{ runner.os }}-go-
```

### 3. タイムアウト設定

各ジョブに適切なタイムアウトを設定：

- **テスト**: 10-20分
- **ベンチマーク**: 30分
- **セキュリティスキャン**: 15分
- **リント**: 10分

## 移行ガイド

### 1. 既存ワークフローからの移行

1. **古いワークフローファイルの削除**:
   ```bash
   rm .github/workflows/ci-optimized.yml
   rm .github/workflows/ci-unified.yml
   rm .github/workflows/ci-simple.yml
   rm .github/workflows/test-common.yml
   rm .github/workflows/windows-optimization.yml
   rm .github/workflows/test-permissions.yml
   ```

2. **新しいワークフローの有効化**:
   - `ci.yml`をメインのCIワークフローとして設定
   - 必要に応じて個別ワークフローを使用

3. **設定の更新**:
   - シークレットの確認（CODECOV_TOKEN、GIST_SECRET）
   - ランナー設定の確認

### 2. 段階的移行

1. **フェーズ1**: 新しいワークフローを並行実行
2. **フェーズ2**: 古いワークフローを無効化
3. **フェーズ3**: 古いファイルを削除

## トラブルシューティング

### 1. よくある問題

#### メモリ不足エラー
```yaml
# Windows環境でのメモリ最適化
env:
  GOGC: 100
  GOMAXPROCS: 4
  CGO_ENABLED: 0
```

#### タイムアウトエラー
```yaml
# タイムアウトの延長
timeout-minutes: 30
```

#### キャッシュエラー
```yaml
# キャッシュキーの確認
key: ${{ runner.os }}-go-${{ inputs.go-version }}-${{ hashFiles('**/go.sum') }}
```

### 2. デバッグ方法

1. **ログの確認**: GitHub Actionsのログを詳細に確認
2. **ローカルテスト**: `make test-ci`でローカル実行
3. **設定の確認**: 環境変数とパラメータの設定を確認

## 今後の改善計画

### 1. 短期的改善
- [ ] カバレッジバッジの自動更新
- [ ] ベンチマーク結果の可視化
- [ ] セキュリティレポートの自動通知

### 2. 長期的改善
- [ ] マルチアーキテクチャ対応
- [ ] コンテナ化対応
- [ ] パフォーマンス監視の統合

## 貢献

CIシステムの改善に貢献する場合：

1. **Issueの作成**: 改善提案やバグ報告
2. **Pull Request**: 具体的な改善実装
3. **ドキュメント更新**: 使用方法や設定の更新

## 参考資料

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Testing](https://golang.org/pkg/testing/)
- [Codecov Documentation](https://docs.codecov.io/)
- [Security Tools](https://github.com/securecodewarrior/gosec) 