# AWSセルフホステッドランナー設定ガイド

## 概要

[machulav/ec2-github-runner](https://github.com/machulav/ec2-github-runner)を使用して、大きなファイルテストを実行するためのAWSセルフホステッドランナーを設定します。

## 前提条件

- AWSアカウント
- GitHub Personal Access Token
- AWS IAMユーザー（EC2権限付き）

## 1. AWS設定

### 1.1 IAMユーザーの作成

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:RunInstances",
                "ec2:CreateTags",
                "ec2:DescribeInstances",
                "ec2:TerminateInstances",
                "ec2:DescribeInstanceStatus",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcs",
                "iam:PassRole"
            ],
            "Resource": "*"
        }
    ]
}
```

### 1.2 EC2インスタンス用のIAMロール

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeTags",
                "ec2:DescribeInstances"
            ],
            "Resource": "*"
        }
    ]
}
```

### 1.3 セキュリティグループの設定

```bash
# セキュリティグループを作成
aws ec2 create-security-group \
    --group-name gopier-test-runner \
    --description "Security group for gopier test runner"

# アウトバウンドルールを設定
aws ec2 authorize-security-group-egress \
    --group-name gopier-test-runner \
    --protocol -1 \
    --port -1 \
    --cidr 0.0.0.0/0
```

### 1.4 AMIの準備

推奨インスタンスタイプ：
- **c5.2xlarge**: 8 vCPU, 16 GB RAM
- **c5.4xlarge**: 16 vCPU, 32 GB RAM
- **r5.2xlarge**: 8 vCPU, 64 GB RAM

## 2. GitHub Secrets設定

以下のシークレットをGitHubリポジトリに設定：

| Secret名 | 説明 |
|----------|------|
| `AWS_ACCESS_KEY_ID` | AWSアクセスキーID |
| `AWS_SECRET_ACCESS_KEY` | AWSシークレットアクセスキー |
| `AWS_REGION` | AWSリージョン（例: us-east-1） |
| `GH_PERSONAL_ACCESS_TOKEN` | GitHub Personal Access Token |
| `EC2_IMAGE_ID` | EC2 AMI ID |
| `EC2_INSTANCE_TYPE` | EC2インスタンスタイプ |
| `EC2_SUBNET_ID` | EC2サブネットID |
| `EC2_SECURITY_GROUP_ID` | EC2セキュリティグループID |
| `EC2_IAM_ROLE_NAME` | EC2 IAMロール名 |

## 3. コスト最適化

### 3.1 インスタンスタイプの選択

```yaml
# 軽量テスト用
ec2-instance-type: c5.large    # 2 vCPU, 4 GB RAM

# 中程度のテスト用
ec2-instance-type: c5.xlarge   # 4 vCPU, 8 GB RAM

# 大きなファイルテスト用
ec2-instance-type: c5.2xlarge  # 8 vCPU, 16 GB RAM
```

### 3.2 Spot インスタンスの使用

```yaml
market-type: spot  # オンデマンドより最大90%割引
```

### 3.3 自動停止の設定

ワークフローで自動的にインスタンスを停止：
- テスト完了後すぐに停止
- エラー時も確実に停止（`if: always()`）

## 4. セキュリティ考慮事項

### 4.1 プライベートリポジトリでの使用

セルフホステッドランナーはプライベートリポジトリでの使用を推奨：
- フォークからの悪意のあるコード実行を防止
- リソースへの不正アクセスを防止

### 4.2 ネットワーク分離

```yaml
# プライベートサブネットでの実行
subnet-id: subnet-private-123
security-group-id: sg-private-456
```

## 5. トラブルシューティング

### 5.1 インスタンス起動エラー

```bash
# インスタンス状態を確認
aws ec2 describe-instances --instance-ids i-1234567890abcdef0

# セキュリティグループを確認
aws ec2 describe-security-groups --group-ids sg-1234567890abcdef0
```

### 5.2 メモリ不足エラー

```yaml
# より大きなインスタンスタイプに変更
ec2-instance-type: r5.2xlarge  # 64 GB RAM
```

### 5.3 タイムアウトエラー

```yaml
# タイムアウト時間を延長
timeout-minutes: 120
```

## 6. 監視とログ

### 6.1 CloudWatch監視

```bash
# CloudWatchメトリクスを有効化
aws cloudwatch put-metric-alarm \
    --alarm-name "EC2RunnerCPU" \
    --alarm-description "CPU usage for EC2 runner" \
    --metric-name CPUUtilization \
    --namespace AWS/EC2 \
    --statistic Average \
    --period 300 \
    --threshold 80 \
    --comparison-operator GreaterThanThreshold
```

### 6.2 コスト監視

```bash
# コスト分析を確認
aws ce get-cost-and-usage \
    --time-period Start=2024-01-01,End=2024-01-31 \
    --granularity MONTHLY \
    --metrics BlendedCost \
    --group-by Type=DIMENSION,Key=SERVICE
```

## 7. ベストプラクティス

### 7.1 リソース管理

- 使用後は必ずインスタンスを停止
- 不要なEBSボリュームを削除
- 定期的にコストを監視

### 7.2 パフォーマンス最適化

- 適切なインスタンスタイプを選択
- 並行テスト数を調整
- キャッシュを活用

### 7.3 セキュリティ強化

- 最小権限の原則を適用
- 定期的にIAMポリシーをレビュー
- セキュリティグループを厳格に設定

## 8. 参考資料

- [machulav/ec2-github-runner](https://github.com/machulav/ec2-github-runner)
- [GitHub Actions Self-hosted Runners](https://docs.github.com/en/actions/hosting-your-own-runners)
- [AWS EC2 Pricing](https://aws.amazon.com/ec2/pricing/)
- [AWS IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html) 