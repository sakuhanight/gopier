# AWS/EC2 全自動統合管理スクリプト

## 概要

`scripts/aws-ec2-automation.sh` は、AWS/EC2管理を完全自動化する統合スクリプトです。既存の複数のスクリプトの機能を統合し、ユーザーの手動入力を最小限に抑えて、AWS/EC2環境の設定から管理までを一括で行います。

## 主な特徴

### 🚀 完全自動化
- **AWS認証情報の自動検出**: AWS CLIの設定から自動取得
- **GitHub認証情報の自動検出**: GitHub CLIの設定から自動取得
- **リポジトリ情報の自動検出**: 複数の方法でGitHubリポジトリを自動検出
- **リソースの自動選択・作成**: サブネット、セキュリティグループ、IAMロールを自動設定
- **GitHub Secretsの自動設定**: 必要なSecretsを自動でGitHubに登録

### 🔧 統合機能
- **EC2ランナー管理**: 起動、停止、ステータス確認
- **リソース管理**: セキュリティグループ、IAMロール、サブネットの自動作成・管理
- **コスト監視**: CloudWatchダッシュボードの自動設定
- **設定管理**: 設定ファイルの自動生成・保存

### 🛡️ 安全性
- **最小権限の原則**: 必要最小限のIAM権限のみを付与
- **セキュリティグループ**: 適切なネットワークアクセス制御
- **自動クリーンアップ**: 不要なリソースの自動削除

## 前提条件

### 必要なツール
```bash
# AWS CLI
brew install awscli  # macOS
# または
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && unzip awscliv2.zip && sudo ./aws/install  # Linux

# GitHub CLI
brew install gh  # macOS
# または
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg  # Linux

# jq
brew install jq  # macOS
# または
sudo apt-get install jq  # Linux
```

### 事前設定
```bash
# AWS認証情報の設定
aws configure
# または
aws sso login

# GitHub認証の設定
gh auth login
```

## 使用方法

### 1. 完全自動設定（推奨）

初回実行時は、このコマンドで全ての設定を自動化します：

```bash
./scripts/aws-ec2-automation.sh auto-setup
```

このコマンドで以下が自動実行されます：
- 前提条件チェック
- 環境変数の自動設定
- AWS情報の自動取得
- サブネットの自動選択・作成
- セキュリティグループの自動作成
- IAMロールの自動作成
- GitHub Secretsの自動設定
- 設定ファイルの保存

### 2. EC2ランナーの管理

#### ランナー起動
```bash
# デフォルト設定で起動
./scripts/aws-ec2-automation.sh start

# カスタムラベルで起動
./scripts/aws-ec2-automation.sh start --label my-custom-runner

# カスタムインスタンスタイプで起動
./scripts/aws-ec2-automation.sh start --type c5.4xlarge
```

#### ランナー停止
```bash
./scripts/aws-ec2-automation.sh stop
```

#### ステータス確認
```bash
./scripts/aws-ec2-automation.sh status
```

### 3. リソース管理

#### リソースクリーンアップ
```bash
./scripts/aws-ec2-automation.sh cleanup
```

このコマンドで以下が削除されます：
- 実行中のEC2インスタンス
- セキュリティグループ
- IAMロール・ポリシー
- 設定ファイル

#### コスト監視設定
```bash
./scripts/aws-ec2-automation.sh monitor
```

CloudWatchダッシュボードが自動作成されます。

## オプション

| オプション | 説明 | デフォルト値 |
|-----------|------|-------------|
| `--label` | ランナーラベル | `gopier-runner-{timestamp}` |
| `--type` | EC2インスタンスタイプ | `c5.2xlarge` |
| `--region` | AWSリージョン | `us-east-1` |
| `--timeout` | タイムアウト時間（分） | `60` |
| `--help` | ヘルプ表示 | - |

## 設定ファイル

自動設定後、`.aws-ec2-config.env` ファイルが生成されます：

```bash
# AWS/EC2自動設定ファイル
# 生成日時: 2024-01-01 12:00:00

# AWS設定
AWS_REGION=us-east-1
EC2_INSTANCE_TYPE=c5.2xlarge

# EC2設定
EC2_IMAGE_ID=ami-12345678
EC2_SUBNET_ID=subnet-12345678
EC2_VPC_ID=vpc-12345678
EC2_AVAILABILITY_ZONE=us-east-1a
EC2_SECURITY_GROUP_ID=sg-12345678
EC2_IAM_ROLE_NAME=gopier-ec2-role

# GitHub設定
GITHUB_REPOSITORY=owner/repo

# プロジェクト設定
PROJECT_NAME=gopier
```

## 自動検出機能

### GitHubリポジトリの自動検出

スクリプトは以下の順序でGitHubリポジトリを自動検出します：

1. **GitHub CLI**: `gh repo view` コマンドから取得
2. **Gitリモート**: `git remote get-url origin` から取得
3. **設定ファイル**: `.github/config` から読み込み
4. **package.json**: Node.jsプロジェクトの場合
5. **go.mod**: Goプロジェクトの場合

### AWSリソースの自動選択

#### サブネット
- デフォルトサブネットを優先選択
- デフォルトサブネットがない場合は最初のサブネットを使用

#### セキュリティグループ
- 既存のセキュリティグループを確認
- 存在しない場合は新規作成
- 適切なインバウンド・アウトバウンドルールを自動設定

#### IAMロール
- 既存のIAMロールを確認
- 存在しない場合は新規作成
- 必要最小限の権限を自動付与

## トラブルシューティング

### よくある問題

#### 1. AWS認証エラー
```bash
ERROR: AWS認証が設定されていません
```
**解決方法**:
```bash
aws configure
# または
aws sso login
```

#### 2. GitHub認証エラー
```bash
ERROR: GitHub CLIが認証されていません
```
**解決方法**:
```bash
gh auth login
```

#### 3. サブネットが見つからない
```bash
ERROR: サブネットが見つかりません
```
**解決方法**:
- AWSコンソールでVPCとサブネットが作成されているか確認
- デフォルトVPCが存在するか確認

#### 4. IAM権限不足
```bash
ERROR: Access Denied
```
**解決方法**:
- 使用するAWSユーザーに必要なIAM権限が付与されているか確認
- 以下の権限が必要：
  - EC2関連: `ec2:*`
  - IAM関連: `iam:*`
  - CloudWatch関連: `cloudwatch:*`

### ログの確認

スクリプトは詳細なログを出力します。エラーが発生した場合は、ログメッセージを確認してください：

```bash
[INFO] 前提条件をチェックしています...
[SUCCESS] 前提条件チェック完了
[STEP] 環境変数を自動設定中...
[INFO] AWS認証情報をAWS CLIから取得しました
[SUCCESS] 環境変数設定完了
```

## セキュリティ考慮事項

### IAM権限
スクリプトで作成されるIAMロールには、必要最小限の権限のみが付与されます：

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeTags",
                "ec2:DescribeInstances",
                "ec2:DescribeInstanceStatus",
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
```

### セキュリティグループ
作成されるセキュリティグループには以下のルールが設定されます：

- **SSH (22)**: 自分のIPアドレスからのみアクセス許可
- **HTTPS (443)**: 全IPアドレスからのアクセス許可
- **HTTP (80)**: 全IPアドレスからのアクセス許可
- **アウトバウンド**: 全トラフィック許可

### 設定ファイル
`.aws-ec2-config.env` ファイルには機密情報が含まれるため、Gitにコミットしないでください。

## 既存スクリプトとの比較

| 機能 | 既存スクリプト | 統合スクリプト |
|------|---------------|---------------|
| 初期設定 | 複数スクリプトが必要 | 1コマンドで完了 |
| 手動入力 | 多数の手動入力 | 最小限の手動入力 |
| リソース管理 | 個別管理 | 統合管理 |
| エラーハンドリング | 基本的 | 詳細 |
| ログ出力 | 標準的 | 色付き・詳細 |
| 設定ファイル | 手動管理 | 自動管理 |

## 移行ガイド

既存のスクリプトから統合スクリプトへの移行：

### 1. 既存リソースの確認
```bash
# 既存のEC2インスタンスを確認
aws ec2 describe-instances --filters "Name=tag:Name,Values=*GitHub-Runner*"

# 既存のセキュリティグループを確認
aws ec2 describe-security-groups --filters "Name=group-name,Values=*gopier*"

# 既存のIAMロールを確認
aws iam list-roles --query 'Roles[?contains(RoleName, `gopier`)].RoleName'
```

### 2. 統合スクリプトの実行
```bash
# 完全自動設定を実行
./scripts/aws-ec2-automation.sh auto-setup
```

### 3. 既存リソースのクリーンアップ
```bash
# 不要になった既存リソースを削除
./scripts/aws-ec2-automation.sh cleanup
```

## 今後の拡張予定

- [ ] 複数リージョン対応
- [ ] 自動スケーリング機能
- [ ] コスト最適化機能
- [ ] 監視・アラート機能の強化
- [ ] バックアップ・復旧機能
- [ ] マルチアカウント対応

## サポート

問題が発生した場合は、以下の手順で調査してください：

1. スクリプトのログを確認
2. AWS CloudTrailでAPI呼び出しを確認
3. GitHub Actionsのログを確認
4. 必要に応じてGitHubのIssuesで報告

## ライセンス

このスクリプトはMITライセンスの下で提供されています。 