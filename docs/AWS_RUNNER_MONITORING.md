# AWSランナー監視スクリプト

## 概要

`monitor-aws-runners.sh`は、CIで使用されるAWSセルフホステッドランナーの使用状況を包括的に監視するスクリプトです。

## 機能

### 1. EC2インスタンス監視
- 実行中・停止中のインスタンスの確認
- インスタンスタイプとリソース使用状況
- 起動時刻とタグ情報

### 2. ランナー使用状況
- GitHubランナーインスタンスの特定
- テストタイプ別の使用状況
- ワークフロー実行履歴

### 3. コスト分析
- 月間コストの確認
- サービス別コスト内訳
- コスト予測と警告

### 4. パフォーマンスメトリクス
- CPU使用率（過去1時間）
- メモリ使用率
- ネットワーク使用量
- CloudWatchメトリクス

### 5. セキュリティ状態
- セキュリティグループの確認
- IAMロールの検証
- アクセス権限の確認

## 前提条件

### 必要なツール
```bash
# AWS CLI
aws --version

# jq (JSONパーサー)
jq --version

# bc (計算ツール)
bc --version
```

### AWS認証設定
```bash
# AWS認証情報の設定
aws configure

# または環境変数で設定
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="ap-northeast-1"
```

### 必要なIAM権限
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeInstances",
                "ec2:DescribeSecurityGroups",
                "cloudwatch:GetMetricStatistics",
                "ce:GetCostAndUsage",
                "iam:ListRoles",
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
```

## 使用方法

### 基本的な使用方法
```bash
# すべての監視を実行（デフォルト）
./scripts/monitor-aws-runners.sh

# ヘルプを表示
./scripts/monitor-aws-runners.sh --help
```

### オプション指定
```bash
# コスト分析のみ実行
./scripts/monitor-aws-runners.sh --cost-analysis

# パフォーマンスメトリクスのみ表示
./scripts/monitor-aws-runners.sh --performance

# セキュリティ状態のみ確認
./scripts/monitor-aws-runners.sh --security

# 特定の監視項目を組み合わせ
./scripts/monitor-aws-runners.sh --cost-analysis --performance

# JSON形式で出力
./scripts/monitor-aws-runners.sh --json
```

## 出力例

### テキスト形式の出力
```
🔍 AWSランナー監視スクリプトを開始します...

[INFO] 前提条件をチェックしています...
[SUCCESS] 前提条件チェック完了

[INFO] AWS情報を取得しています...
=== AWS情報 ===
アカウントID: 123456789012
リージョン: ap-northeast-1
ユーザー: arn:aws:iam::123456789012:user/ci-user

[INFO] EC2インスタンスの状態を確認しています...
=== EC2インスタンス状態 ===
インスタンスID: i-1234567890abcdef0
状態: running
タイプ: c5.2xlarge
起動時刻: 2024-01-15T10:30:00.000Z
タグ: Name=gopier-test-runner, GitHubRepository=sakuha/gopier, TestType=large-files
---

[INFO] GitHubランナーの使用状況を確認しています...
=== ランナー使用状況 ===
インスタンスID: i-1234567890abcdef0
状態: running
起動時刻: 2024-01-15T10:30:00.000Z
テストタイプ: large-files
---

[INFO] コスト分析を実行しています...
=== コスト分析 (2024-01-01 〜 2024-01-15) ===
総コスト: 45.67 USD

サービス別コスト:
Amazon EC2: 42.30 USD
Amazon CloudWatch: 2.15 USD
AWS Cost Explorer: 1.22 USD

[INFO] パフォーマンスメトリクスを取得しています...
=== パフォーマンスメトリクス (過去1時間) ===
インスタンスID: i-1234567890abcdef0
  CPU使用率: 75.2%
  ネットワーク受信: 1024000 bytes
  ネットワーク送信: 512000 bytes
---

[INFO] セキュリティ状態を確認しています...
=== セキュリティ状態 ===
セキュリティグループID: sg-1234567890abcdef0
名前: gopier-test-runner
説明: Security group for gopier test runner
---

=== IAMロール確認 ===
関連するIAMロール:
  - gopier-test-runner-role

[INFO] リソース使用量の要約を生成しています...

=== 監視サマリー ===
監視時刻: Mon Jan 15 19:30:00 JST 2024
実行中インスタンス数: 1
停止中インスタンス数: 0
今月の推定コスト: $45.67

=== 推奨事項 ===
[WARN] 実行中のインスタンスがあります。使用後は停止してください。

[SUCCESS] 監視完了
```

### JSON形式の出力
```json
{
  "account_id": "123456789012",
  "region": "ap-northeast-1",
  "user": "arn:aws:iam::123456789012:user/ci-user",
  "ec2_instances": [
    {
      "InstanceId": "i-1234567890abcdef0",
      "State": {"Name": "running"},
      "InstanceType": "c5.2xlarge",
      "LaunchTime": "2024-01-15T10:30:00.000Z",
      "Tags": [
        {"Key": "Name", "Value": "gopier-test-runner"},
        {"Key": "GitHubRepository", "Value": "sakuha/gopier"},
        {"Key": "TestType", "Value": "large-files"}
      ]
    }
  ],
  "runner_usage": [
    ["i-1234567890abcdef0", "running", "2024-01-15T10:30:00.000Z", "large-files"]
  ],
  "cost_analysis": {
    "ResultsByTime": [
      {
        "Total": {
          "BlendedCost": {
            "Amount": "45.67",
            "Unit": "USD"
          }
        },
        "Groups": [
          {
            "Keys": ["Amazon EC2"],
            "Metrics": {
              "BlendedCost": {
                "Amount": "42.30",
                "Unit": "USD"
              }
            }
          }
        ]
      }
    ]
  },
  "performance": {
    "i-1234567890abcdef0": {
      "cpu": {
        "Datapoints": [
          {
            "Average": 75.2,
            "Timestamp": "2024-01-15T19:00:00.000Z"
          }
        ]
      },
      "memory": {
        "Datapoints": []
      }
    }
  },
  "security": [
    ["sg-1234567890abcdef0", "gopier-test-runner", "Security group for gopier test runner", []]
  ]
}
```

## 定期実行

### cronでの定期実行
```bash
# 毎日午前9時に実行
0 9 * * * /path/to/gopier/scripts/monitor-aws-runners.sh >> /var/log/aws-runner-monitor.log 2>&1

# 毎時間実行
0 * * * * /path/to/gopier/scripts/monitor-aws-runners.sh --json > /tmp/runner-status.json
```

### GitHub Actionsでの定期実行
```yaml
name: AWS Runner Monitoring

on:
  schedule:
    - cron: '0 9 * * *'  # 毎日午前9時
  workflow_dispatch:     # 手動実行も可能

jobs:
  monitor:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ secrets.AWS_REGION }}
      
      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq bc
      
      - name: Run monitoring script
        run: ./scripts/monitor-aws-runners.sh --json
      
      - name: Upload monitoring results
        uses: actions/upload-artifact@v4
        with:
          name: aws-runner-monitoring-$(date +%Y%m%d)
          path: runner-status.json
          retention-days: 30
```

## トラブルシューティング

### よくあるエラー

#### AWS認証エラー
```
[ERROR] AWS認証が設定されていません
```
**解決方法:**
```bash
aws configure
# または環境変数を設定
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
```

#### 権限不足エラー
```
An error occurred (UnauthorizedOperation) when calling the DescribeInstances operation
```
**解決方法:** IAMユーザーに必要な権限を追加

#### jqが見つからないエラー
```
[ERROR] jqがインストールされていません
```
**解決方法:**
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# CentOS/RHEL
sudo yum install jq
```

### デバッグモード
```bash
# デバッグ情報を表示
bash -x ./scripts/monitor-aws-runners.sh

# 特定の関数のみデバッグ
bash -x ./scripts/monitor-aws-runners.sh --cost-analysis
```

## カスタマイズ

### プロジェクト名の変更
```bash
# スクリプト内のPROJECT_NAME変数を変更
PROJECT_NAME="your-project-name"
```

### リージョンの変更
```bash
# デフォルトリージョンを変更
DEFAULT_REGION="us-east-1"
```

### 監視項目の追加
新しい監視機能を追加する場合：

1. 新しい関数を作成
2. オプション解析に追加
3. メイン実行部分に追加

```bash
# 新しい監視関数の例
check_custom_metric() {
    log_info "カスタムメトリクスを確認しています..."
    # 実装
}

# オプション解析に追加
--custom-metric)
    CUSTOM_METRIC=true
    shift
    ;;

# メイン実行部分に追加
if [[ "$CUSTOM_METRIC" == true || "$ALL_MONITORING" == true ]]; then
    check_custom_metric
fi
```

## ベストプラクティス

### 1. 定期実行
- 毎日1回は実行してリソース状況を確認
- コストが高い場合は毎時間実行

### 2. アラート設定
- コストが予算を超えた場合の通知
- 実行中インスタンスが長時間残っている場合の警告

### 3. ログ管理
- 監視結果をログファイルに保存
- 古いログは定期的に削除

### 4. セキュリティ
- 最小権限の原則に従う
- アクセスキーは定期的にローテーション

## 関連ドキュメント

- [AWSランナー設定ガイド](./AWS_RUNNER_SETUP.md)
- [CI最適化ガイド](./CI_OPTIMIZATION.md)
- [コスト監視設定](./CI_ENVIRONMENT.md) 