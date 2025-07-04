# EC2 Self-Hosted Runner Management

このドキュメントでは、`machulav/ec2-github-runner`を使用せずに独自実装したEC2セルフホステッドランナー管理システムについて説明します。

## 概要

このシステムは以下のコンポーネントで構成されています：

- `scripts/ec2-runner-manager.sh` - メインのランナー管理スクリプト
- `scripts/ec2-runner-helper.sh` - ランナー監視・管理のヘルパースクリプト
- `.github/workflows/aws-runner-custom.yml` - カスタム実装を使用するGitHub Actionsワークフロー

## 機能

### メイン機能
- EC2インスタンスの自動作成・起動
- GitHub Actionsランナーの自動設定
- テスト実行後の自動クリーンアップ
- ランナーの状態監視
- コスト管理

### ヘルパー機能
- ランナーの健全性チェック
- 古いランナーの自動クリーンアップ
- コストレポートの生成
- ランナー一覧の表示

## セットアップ

### 1. 必要な環境変数

以下の環境変数を設定してください：

```bash
# AWS認証情報
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# GitHub認証情報
export GITHUB_TOKEN="your-personal-access-token"
export GITHUB_REPOSITORY="owner/repo"
```

### 2. IAM Instance Profileの自動設定

統合されたスクリプトを使用してIAM Instance Profileを自動設定できます：

```bash
# IAM設定の実行
./scripts/ec2-runner-manager.sh setup-iam --role-name EC2RunnerRole --profile-name EC2RunnerRole

# または、ヘルパースクリプトを使用
./scripts/ec2-runner-helper.sh setup-iam --role-name EC2RunnerRole --profile-name EC2RunnerRole
```

このコマンドは以下を自動的に実行します：
- IAMロールの作成
- 必要なポリシーの作成とアタッチ
- インスタンスプロファイルの作成
- ロールとインスタンスプロファイルの関連付け

### 3. IAM設定の確認

```bash
# IAM設定の確認
./scripts/ec2-runner-manager.sh verify-iam

# または、ヘルパースクリプトを使用
./scripts/ec2-runner-helper.sh verify-iam
```

### 4. GitHub Secretsの設定

スクリプト実行後、GitHubリポジトリのSettings > Secrets and variables > Actionsで以下のシークレットを設定：

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`
- `GH_PERSONAL_ACCESS_TOKEN`
- `EC2_SUBNET_ID`
- `EC2_SECURITY_GROUP_ID`
- `EC2_IAM_ROLE_NAME`: `EC2RunnerRole`（作成したインスタンスプロファイル名）

### 5. AWSリソースの準備

#### セキュリティグループ
以下のルールを含むセキュリティグループを作成：

- SSH (ポート22) - 必要に応じて
- HTTPS (ポート443) - アウトバウンド
- HTTP (ポート80) - アウトバウンド

#### サブネット
EC2インスタンスを配置するサブネットを準備

### 6. 手動設定（オプション）

自動設定ができない場合は、以下の手動コマンドを使用：

```bash
# IAMロールの作成
aws iam create-role \
  --role-name EC2RunnerRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": {
          "Service": "ec2.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
      }
    ]
  }'

# インスタンスプロファイルの作成
aws iam create-instance-profile \
  --instance-profile-name EC2RunnerRole

# ロールをインスタンスプロファイルに追加
aws iam add-role-to-instance-profile \
  --instance-profile-name EC2RunnerRole \
  --role-name EC2RunnerRole

# 必要なポリシーのアタッチ
aws iam attach-role-policy \
  --role-name EC2RunnerRole \
  --policy-arn arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess
```

## 使用方法

### メインスクリプト

#### ランナーの起動
```bash
./scripts/ec2-runner-manager.sh start \
  --label "my-runner" \
  --type "t3.medium" \
  --subnet "subnet-12345678" \
  --sg "sg-12345678" \
  --role "EC2RunnerRole"
```

#### ランナーの停止
```bash
./scripts/ec2-runner-manager.sh stop --label "my-runner"
```

#### ランナーのステータス確認
```bash
./scripts/ec2-runner-manager.sh status --label "my-runner"
```

#### 古いランナーのクリーンアップ
```bash
./scripts/ec2-runner-manager.sh cleanup
```

#### IAM設定
```bash
# IAM設定の実行
./scripts/ec2-runner-manager.sh setup-iam --role-name EC2RunnerRole --profile-name EC2RunnerRole

# IAM設定の確認
./scripts/ec2-runner-manager.sh verify-iam
```

### ヘルパースクリプト

#### ランナーの監視
```bash
./scripts/ec2-runner-helper.sh monitor --repository "owner/repo"
```

#### 健全性チェック
```bash
./scripts/ec2-runner-helper.sh health-check --repository "owner/repo"
```

#### コストレポート
```bash
./scripts/ec2-runner-helper.sh cost-report --repository "owner/repo"
```

#### ランナー一覧
```bash
./scripts/ec2-runner-helper.sh list --repository "owner/repo"
```

#### IAM設定
```bash
# IAM設定の実行
./scripts/ec2-runner-helper.sh setup-iam --role-name EC2RunnerRole --profile-name EC2RunnerRole

# IAM設定の確認
./scripts/ec2-runner-helper.sh verify-iam
```

### GitHub Actionsでの使用

#### ワークフローの呼び出し
```yaml
name: Test with Custom EC2 Runner

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    uses: ./.github/workflows/aws-runner-custom.yml
    with:
      go-version: '1.22.0'
      test-type: 'large-files'
      instance-type: 't3.large'
      timeout-minutes: 120
    secrets: inherit
```

## 設定オプション

### EC2インスタンス設定

| オプション | デフォルト | 説明 |
|-----------|-----------|------|
| `--type` | `t3.medium` | EC2インスタンスタイプ |
| `--image` | Amazon Linux 2 | AMI ID |
| `--subnet` | 必須 | サブネットID |
| `--sg` | 必須 | セキュリティグループID |
| `--role` | 必須 | IAMロール名 |

### テスト設定

| オプション | デフォルト | 説明 |
|-----------|-----------|------|
| `test-type` | `large-files` | テストタイプ（large-files, integration, benchmark） |
| `timeout-minutes` | 60 | タイムアウト時間（分） |

## トラブルシューティング

### よくある問題

#### 1. IAMロールエラー
```
Value (*) for parameter iamInstanceProfile.name is invalid. Invalid IAM Instance Profile name
```

**解決策**: 
```bash
# IAM設定を確認
./scripts/ec2-runner-manager.sh verify-iam

# 問題がある場合は再設定
./scripts/ec2-runner-manager.sh setup-iam --role-name EC2RunnerRole --profile-name EC2RunnerRole
```

#### 2. ランナー登録タイムアウト
```
ランナーの登録がタイムアウトしました
```

**解決策**: 
- セキュリティグループでHTTPSアウトバウンドが許可されているか確認
- インスタンスの起動ログを確認（`/var/log/user-data.log`）

#### 3. 認証エラー
```
AWS認証情報が無効です
```

**解決策**: AWS認証情報が正しく設定されているか確認

### ログの確認

#### EC2インスタンスのログ
```bash
# インスタンスにSSH接続してログを確認
sudo tail -f /var/log/user-data.log
sudo journalctl -u actions.runner.* -f
```

#### GitHub Actionsのログ
GitHub Actionsの実行ログで詳細なエラー情報を確認できます。

## コスト最適化

### 推奨設定

1. **インスタンスタイプの選択**
   - 軽量テスト: `t3.small`
   - 通常テスト: `t3.medium`
   - 重いテスト: `t3.large`

2. **自動クリーンアップ**
   - 24時間経過したインスタンスを自動停止
   - 定期的なクリーンアップの実行

3. **コスト監視**
   - 月次コストレポートの確認
   - 使用量の監視

### コスト削減のヒント

- テスト完了後の即座なインスタンス停止
- 適切なインスタンスタイプの選択
- 不要なインスタンスの早期削除
- スポットインスタンスの検討（安定性を重視する場合は避ける）

## セキュリティ

### 推奨事項

1. **最小権限の原則**
   - IAMロールには必要最小限の権限のみ付与
   - セキュリティグループは必要最小限のポートのみ開放

2. **認証情報の管理**
   - GitHub Personal Access Tokenは適切な権限で作成
   - AWS認証情報は定期的にローテーション

3. **ネットワークセキュリティ**
   - プライベートサブネットの使用を検討
   - セキュリティグループの厳格な設定

## 更新履歴

- 2024-01-XX: 初回リリース
- カスタム実装によるmachulav/ec2-github-runnerからの移行
- ヘルパースクリプトの追加
- コスト管理機能の実装
- IAM設定機能の統合 