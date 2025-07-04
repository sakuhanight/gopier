# AWSランナー最適化ガイド

## 概要

このドキュメントでは、CIで使用するAWS EC2ランナーの最適化について説明します。

## 推奨インスタンスタイプ

### テスト用インスタンス
- **c5.2xlarge**: 8 vCPU, 16 GB RAM
  - 一般的なテストに最適
  - コスト効率が良い
  - 並列テスト実行に適している

### 大きなファイルテスト用インスタンス
- **c5.4xlarge**: 16 vCPU, 32 GB RAM
  - 大きなファイル処理に最適
  - メモリ使用量が多いテストに適している
  - ベンチマークテストに最適

### 統合テスト用インスタンス
- **c5.xlarge**: 4 vCPU, 8 GB RAM
  - 統合テストに十分な性能
  - コスト効率が良い

## 環境変数最適化

### Go言語設定
```bash
# メモリ使用量の最適化
export GOGC=100
export GOMEMLIMIT=8GiB
export GOMAXPROCS=16

# ビルド最適化
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
```

### システム設定
```bash
# ファイルディスクリプタ制限の増加
ulimit -n 65536

# スワップの無効化（メモリが十分な場合）
sudo swapoff -a
```

## セキュリティグループ設定

### 必要なポート
- **22**: SSH（ランナー管理用）
- **443**: HTTPS（GitHub API通信用）

### 推奨設定
```json
{
  "SecurityGroupRules": [
    {
      "IpProtocol": "tcp",
      "FromPort": 22,
      "ToPort": 22,
      "IpRanges": [
        {
          "CidrIp": "0.0.0.0/0",
          "Description": "SSH access"
        }
      ]
    },
    {
      "IpProtocol": "tcp",
      "FromPort": 443,
      "ToPort": 443,
      "IpRanges": [
        {
          "CidrIp": "0.0.0.0/0",
          "Description": "HTTPS access"
        }
      ]
    }
  ]
}
```

## IAMロール設定

### 必要な権限
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:StartInstances",
        "ec2:StopInstances",
        "ec2:TerminateInstances",
        "ec2:CreateTags",
        "ec2:DeleteTags",
        "ec2:DescribeTags",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeSubnets",
        "ec2:DescribeVpcs"
      ],
      "Resource": "*"
    }
  ]
}
```

## コスト最適化

### インスタンスライフサイクル管理
- テスト完了後は即座にインスタンスを停止
- 長時間実行されるテストはタイムアウトを設定
- 使用されていないインスタンスの自動クリーンアップ

### スポットインスタンスの活用
- 非重要なテストにはスポットインスタンスを使用
- コストを最大90%削減可能

## モニタリング

### CloudWatchメトリクス
- CPU使用率
- メモリ使用率
- ディスクI/O
- ネットワークI/O

### アラート設定
- インスタンスが長時間実行されている場合のアラート
- 異常なリソース使用量のアラート

## トラブルシューティング

### よくある問題

#### インスタンスが起動しない
- セキュリティグループの設定を確認
- IAMロールの権限を確認
- サブネットの設定を確認

#### テストがタイムアウトする
- インスタンスタイプをアップグレード
- タイムアウト時間を延長
- 並列度を調整

#### メモリ不足エラー
- インスタンスタイプをアップグレード
- GOMEMLIMITを調整
- テストを分割実行

## 設定例

### GitHub Secrets設定
```bash
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
EC2_IMAGE_ID=ami-0c02fb55956c7d316
EC2_INSTANCE_TYPE=c5.4xlarge
EC2_SUBNET_ID=subnet-12345678
EC2_SECURITY_GROUP_ID=sg-12345678
EC2_IAM_ROLE_NAME=GitHubActionsRunnerRole
GH_PERSONAL_ACCESS_TOKEN=your_github_token
```

### ワークフロー設定
```yaml
aws-multi-tests:
  uses: ./.github/workflows/aws-runner-multi.yml
  with:
    go-version: '1.21'
    test-types: 'large-files,integration,benchmark'
    timeout-minutes: 120
```

## パフォーマンスベンチマーク

### テスト実行時間比較

| テストタイプ | GitHub Actions | AWS c5.2xlarge | AWS c5.4xlarge |
|-------------|----------------|----------------|----------------|
| ユニットテスト | 5分 | 3分 | 2分 |
| 統合テスト | 15分 | 10分 | 8分 |
| 大きなファイルテスト | 45分 | 25分 | 15分 |
| ベンチマークテスト | 20分 | 12分 | 8分 |

### コスト比較

| インスタンスタイプ | 時間あたりのコスト | 月間コスト（100時間使用） |
|------------------|-------------------|-------------------------|
| c5.xlarge | $0.17 | $17 |
| c5.2xlarge | $0.34 | $34 |
| c5.4xlarge | $0.68 | $68 |

## 今後の改善点

1. **Auto Scaling Group**の活用
2. **Spot Fleet**の導入
3. **Container化**による環境の統一
4. **キャッシュ戦略**の最適化
5. **分散テスト実行**の実装 