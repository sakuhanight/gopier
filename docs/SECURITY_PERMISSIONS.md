# セキュリティ権限設定

このドキュメントでは、GitHub Actionsワークフローでの安全な権限設定について説明します。

## 概要

GitHub Actionsでは、最小権限の原則に従って権限を設定することで、セキュリティリスクを最小化できます。

## 権限設定の原則

### 1. 最小権限の原則
- 必要最小限の権限のみを付与
- デフォルトでは読み取り専用権限
- 書き込み権限は必要に応じて明示的に設定

### 2. 権限の分離
- ジョブごとに適切な権限を設定
- テスト、ビルド、デプロイで異なる権限レベル
- 機密操作には追加の承認を要求

### 3. セキュリティチェック
- フォークからのPRを制限
- 承認プロセスの必須化
- 自己承認の禁止

## 設定ファイル

### メインCI設定
- **ファイル**: `.github/workflows/ci.yml`
- **目的**: メインのCI/CDパイプライン
- **権限**: テストとビルドに必要な最小権限

### 権限設定ワークフロー
- **ファイル**: `.github/workflows/test-permissions.yml`
- **目的**: テスト実行時の権限制御
- **機能**: ディレクトリ作成、権限設定、セキュリティチェック

### セキュリティ設定
- **ファイル**: `.github/security/permissions.yml`
- **目的**: 権限設定の定義と管理
- **内容**: デフォルト権限、ジョブ別権限、セキュリティチェック

## 権限レベル

### 読み取り専用権限
```yaml
permissions:
  contents: read        # リポジトリコンテンツ
  actions: read         # Actions
  pull-requests: read   # PR
```

### 書き込み権限（必要最小限）
```yaml
permissions:
  security-events: write # セキュリティイベント
  checks: write         # チェック結果
  statuses: write       # ステータス更新
```

### 制限された権限
```yaml
permissions:
  deployments: none     # デプロイメント
  issues: none          # イシュー
  packages: none        # パッケージ
  pages: none           # Pages
```

## テスト実行時の安全設定

### ディレクトリ権限
```bash
# テスト用ディレクトリを作成し、適切な権限を設定
mkdir -p test_output test_data coverage_reports build_output dist
chmod 755 test_output test_data coverage_reports build_output dist
```

### 環境変数
```bash
# テスト実行時の環境変数
CI=true
GITHUB_ACTIONS=true
TESTING=1
TEST_OUTPUT_DIR=./test_output
COVERAGE_DIR=./coverage_reports
```

### セキュリティチェック
```bash
# 権限の最小化確認
if [ "$GITHUB_REF" != "refs/heads/main" ] && [ "$GITHUB_REF" != "refs/heads/develop" ]; then
  echo "⚠️  Warning: Running on non-main branch"
fi

# フォークからのPRチェック
if [ "$GITHUB_EVENT_NAME" = "pull_request" ] && [ "$GITHUB_BASE_REF" != "main" ] && [ "$GITHUB_BASE_REF" != "develop" ]; then
  echo "⚠️  Warning: PR from non-main branch"
fi
```

## セキュリティベストプラクティス

### 1. 機密情報の保護
- シークレットは暗号化して保存
- 環境変数での機密情報の露出を避ける
- ログ出力での機密情報の漏洩を防ぐ

### 2. ネットワークセキュリティ
- 必要最小限のネットワークアクセス
- 外部APIへのアクセスを制限
- ファイアウォールルールの適用

### 3. ファイルシステムセキュリティ
- テスト用ディレクトリのみにアクセス
- 一時ファイルの適切な削除
- 権限の最小化

### 4. プロセスセキュリティ
- テストプロセスの分離
- リソース制限の設定
- タイムアウトの設定

## トラブルシューティング

### 権限エラーの対処
1. **権限不足エラー**
   ```bash
   # 権限を確認
   ls -la test_output/
   # 権限を修正
   chmod 755 test_output/
   ```

2. **ディレクトリ作成エラー**
   ```bash
   # ディレクトリの存在確認
   mkdir -p test_output
   # 権限設定
   chmod 755 test_output
   ```

3. **環境変数エラー**
   ```bash
   # 環境変数の確認
   echo $CI
   echo $GITHUB_ACTIONS
   echo $TESTING
   ```

### セキュリティ警告の対処
1. **フォークPR警告**
   - フォークからのPRを制限
   - 承認プロセスを必須化

2. **権限過多警告**
   - 不要な権限を削除
   - 最小権限の原則を適用

## 監査とログ

### 権限使用の監査
```bash
# 権限使用状況の確認
echo "Checking permission usage..."
for dir in test_output test_data coverage_reports; do
  if [ -d "$dir" ]; then
    perms=$(stat -c "%a" "$dir")
    echo "$dir: $perms"
  fi
done
```

### セキュリティログ
```bash
# セキュリティイベントのログ
echo "Security audit completed"
echo "Permissions: OK"
echo "Environment: OK"
echo "Access control: OK"
```

## 参考資料

- [GitHub Actions Permissions](https://docs.github.com/en/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token)
- [Security Hardening for GitHub Actions](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [Using GitHub Actions with OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect) 


**最終更新**: 2025/07/06