# EC2 Self-Hosted Runner Migration Guide

## 概要

`machulav/ec2-github-runner@v2`のサポート終了に伴い、独自のEC2ランナー自動作成・削除ワークフローに移行しました。

## 変更点

### 1. 新しいワークフロー

- **`ec2-runner-manager.yml`**: 完全にカスタムなEC2ランナー管理ワークフロー
- **`ci-ec2-runner.yml`**: 既存のCIワークフローを更新して独自実装を使用

### 2. 主な改善点

#### 自動化スクリプトの強化
- 最新のGitHub Actionsランナーバージョンを自動取得
- Go 1.21/1.22の自動インストール
- Windowsクロスコンパイルツールチェーンの自動インストール
- 詳細なログ出力とエラーハンドリング

#### ワークフローの改善
- 手動実行可能なワークフロー（workflow_dispatch）
- ランナー登録の自動待機機能
- テスト完了後の自動ランナー停止
- リソースクリーンアップの自動化

## 使用方法

### 1. 手動実行

GitHub Actionsの「Actions」タブから手動でワークフローを実行できます：

```bash
# ランナー起動
./scripts/aws-ec2-automation.sh start --label my-runner --type c5.2xlarge

# ランナー停止
./scripts/aws-ec2-automation.sh stop

# ステータス確認
./scripts/aws-ec2-automation.sh status

# リソースクリーンアップ
./scripts/aws-ec2-automation.sh cleanup
```

### 2. 自動実行

プッシュやプルリクエスト時に自動的に実行されます。

## 設定

### 必要なGitHub Secrets

```yaml
AWS_ACCESS_KEY_ID: AWSアクセスキーID
AWS_SECRET_ACCESS_KEY: AWSシークレットアクセスキー
AWS_REGION: AWSリージョン（例: us-east-1）
EC2_IMAGE_ID: EC2 AMI ID
EC2_SUBNET_ID: サブネットID
EC2_SECURITY_GROUP_ID: セキュリティグループID
EC2_IAM_ROLE_NAME: IAMロール名
EC2_INSTANCE_TYPE: インスタンスタイプ（例: c5.2xlarge）
GITHUB_TOKEN: GitHubトークン
CODECOV_TOKEN: Codecovトークン（オプション）
```

### 初回セットアップ

```bash
# 完全自動設定
./scripts/aws-ec2-automation.sh auto-setup
```

## ワークフロー詳細

### ec2-runner-manager.yml

完全にカスタムなEC2ランナー管理ワークフロー：

- **手動実行**: workflow_dispatchで手動実行可能
- **ランナー起動**: カスタムスクリプトでEC2インスタンスを作成
- **ランナー登録**: GitHub Actionsランナーを自動登録
- **テスト実行**: 起動したランナーでテストを実行
- **自動停止**: テスト完了後にランナーを自動停止

### ci-ec2-runner.yml

既存のCIワークフローを更新：

- **machulav/ec2-github-runner@v2**を独自実装に置き換え
- **自動ランナー管理**: テスト実行中のみランナーを起動
- **コスト最適化**: 使用時のみリソースを消費

## トラブルシューティング

### よくある問題

1. **ランナーが起動しない**
   ```bash
   # ログを確認
   ./scripts/aws-ec2-automation.sh status
   
   # AWS認証情報を確認
   aws sts get-caller-identity
   ```

2. **ランナーが登録されない**
   ```bash
   # GitHubトークンを確認
   gh auth status
   
   # ランナートークンを手動取得
   gh api repos/OWNER/REPO/actions/runners/registration-token
   ```

3. **インスタンスが停止しない**
   ```bash
   # 強制停止
   aws ec2 terminate-instances --instance-ids i-1234567890abcdef0
   ```

### ログ確認

```bash
# EC2インスタンスのログ
aws ec2 get-console-output --instance-id i-1234567890abcdef0

# ユーザーデータスクリプトのログ
ssh ec2-user@PUBLIC_IP "sudo cat /var/log/user-data.log"

# GitHub Actionsランナーのログ
ssh ec2-user@PUBLIC_IP "sudo cat /var/log/actions-runner.log"
```

## パフォーマンス最適化

### インスタンスタイプの選択

```yaml
# 推奨設定
c5.2xlarge: 8 vCPU, 16 GB RAM - 一般的なCI/CD
c5.4xlarge: 16 vCPU, 32 GB RAM - 大規模ビルド
c5.9xlarge: 36 vCPU, 72 GB RAM - 高負荷テスト
```

### タイムアウト設定

```yaml
# デフォルト: 60分
timeout_minutes: 120  # 大規模プロジェクト用
```

## セキュリティ

### IAMポリシー

最小権限の原則に従ったIAMポリシー：

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

```bash
# SSHアクセス（自分のIPのみ）
aws ec2 authorize-security-group-ingress \
  --group-id sg-1234567890abcdef0 \
  --protocol tcp \
  --port 22 \
  --cidr YOUR_IP/32

# HTTPSアクセス
aws ec2 authorize-security-group-ingress \
  --group-id sg-1234567890abcdef0 \
  --protocol tcp \
  --port 443 \
  --cidr 0.0.0.0/0
```

## コスト管理

### 監視設定

```bash
# CloudWatchダッシュボードの作成
./scripts/aws-ec2-automation.sh monitor
```

### コスト最適化

1. **スポットインスタンスの使用**
2. **自動停止の確実な実行**
3. **リソースクリーンアップの定期実行**

## 移行チェックリスト

- [ ] 新しいワークフローファイルの配置
- [ ] GitHub Secretsの設定確認
- [ ] AWS認証情報の確認
- [ ] 初回テスト実行
- [ ] 既存ランナーの停止
- [ ] 古いワークフローの無効化

## サポート

問題が発生した場合は、以下を確認してください：

1. GitHub Actionsのログ
2. EC2インスタンスのログ
3. AWS CloudWatchログ
4. スクリプトの実行ログ

詳細なトラブルシューティングについては、プロジェクトのIssuesページをご確認ください。 