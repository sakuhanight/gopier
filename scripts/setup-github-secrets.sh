#!/bin/bash

# GitHub Secrets自動設定スクリプト
# 使用方法: ./scripts/setup-github-secrets.sh

set -e

    # グローバル変数の宣言
    AWS_ACCESS_KEY_ID=""
    AWS_SECRET_ACCESS_KEY=""
    GH_PERSONAL_ACCESS_TOKEN=""
    AWS_REGION=""
    EC2_SECURITY_GROUP_ID=""
    EC2_SUBNET_ID=""
    EC2_IMAGE_ID=""
    EC2_INSTANCE_TYPE=""
    EC2_IAM_ROLE_NAME=""
    SETUP_COST_MONITORING=""

echo "🚀 GitHub Secrets自動設定を開始します..."

# 色付きログ関数
log_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

log_warn() {
    echo -e "\033[33m[WARN]\033[0m $1"
}

log_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

# 前提条件チェック
check_prerequisites() {
    log_info "前提条件をチェックしています..."
    
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLIがインストールされていません"
        exit 1
    fi
    
    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLIがインストールされていません"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jqがインストールされていません"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curlがインストールされていません"
        exit 1
    fi
    
    # AWS認証チェック
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS認証が設定されていません"
        exit 1
    fi
    
    # GitHub認証チェック
    if ! gh auth status &> /dev/null; then
        log_warn "GitHub CLIが認証されていません"
        log_info "GitHub CLIの認証を行います..."
        if ! gh auth login --web; then
            log_error "GitHub CLIの認証に失敗しました"
            exit 1
        fi
    fi
    
    log_info "前提条件チェック完了 ✅"
}

# AWS情報を取得
get_aws_info() {
    log_info "AWS情報を取得しています..."
    
    # アカウントID
    AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    log_info "AWS Account ID: $AWS_ACCOUNT_ID"
    
    # リージョン
    AWS_REGION=$(aws configure get region || echo "ap-northeast-1")
    log_info "AWS Region: $AWS_REGION"
    
    # 最新のAmazon Linux 2 AMIを取得
    log_info "最新のAmazon Linux 2 AMIを取得しています..."
    EC2_IMAGE_ID=$(aws ec2 describe-images \
        --owners amazon \
        --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" "Name=state,Values=available" \
        --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
        --output text \
        --region "$AWS_REGION")
    
    if [ -z "$EC2_IMAGE_ID" ]; then
        log_error "AMI IDの取得に失敗しました"
        exit 1
    fi
    log_info "EC2 Image ID: $EC2_IMAGE_ID"
    
    # セキュリティグループID
    setup_security_group
    
    # サブネットID
    log_info "利用可能なサブネットを表示します:"
    aws ec2 describe-subnets --query 'Subnets[*].[SubnetId,VpcId,AvailabilityZone]' --output table --region "$AWS_REGION"
    read -p "使用するサブネットIDを入力してください: " EC2_SUBNET_ID
    
    # インスタンスタイプ
    EC2_INSTANCE_TYPE="c5.2xlarge"
    log_info "EC2 Instance Type: $EC2_INSTANCE_TYPE"
    
    # IAMロール名
    EC2_IAM_ROLE_NAME="gopier-ec2-role"
    log_info "EC2 IAM Role Name: $EC2_IAM_ROLE_NAME"
    
    # EC2 IAMロールの作成または確認
    setup_ec2_iam_role
}

# IAMユーザーの作成または選択
setup_iam_user() {
    log_info "IAMユーザーの設定を行います..."
    
    # 既存のIAMユーザーを確認
    log_info "既存のIAMユーザーを確認しています..."
    EXISTING_USERS=$(aws iam list-users --query 'Users[?contains(UserName, `gopier`) || contains(UserName, `ci`) || contains(UserName, `runner`)].UserName' --output text 2>/dev/null || echo "")
    
    if [ -n "$EXISTING_USERS" ]; then
        echo "既存の関連ユーザーが見つかりました:"
        for user in $EXISTING_USERS; do
            echo "  - $user"
        done
        
        read -p "既存のユーザーを使用しますか？ (y/n): " USE_EXISTING
        if [[ $USE_EXISTING =~ ^[Yy]$ ]]; then
            echo "使用可能なユーザー:"
            for user in $EXISTING_USERS; do
                echo "  - $user"
            done
            read -p "使用するユーザー名を入力してください: " SELECTED_USER
            
            # 選択されたユーザーのアクセスキーを確認
            EXISTING_KEYS=$(aws iam list-access-keys --user-name "$SELECTED_USER" --query 'AccessKeyMetadata[?Status==`Active`].AccessKeyId' --output text 2>/dev/null || echo "")
            
            if [ -n "$EXISTING_KEYS" ]; then
                log_info "既存のアクセスキーが見つかりました: $EXISTING_KEYS"
                read -p "既存のアクセスキーを使用しますか？ (y/n): " USE_EXISTING_KEY
                if [[ $USE_EXISTING_KEY =~ ^[Yy]$ ]]; then
                    AWS_ACCESS_KEY_ID="$EXISTING_KEYS"
                    read -s -p "AWS Secret Access Key: " AWS_SECRET_ACCESS_KEY
                    echo
                else
                    create_new_access_key "$SELECTED_USER"
                fi
            else
                create_new_access_key "$SELECTED_USER"
            fi
        else
            create_new_user
        fi
    else
        create_new_user
    fi
}

# 新しいIAMユーザーを作成
create_new_user() {
    log_info "新しいIAMユーザーを作成します..."
    
    read -p "作成するユーザー名 (デフォルト: gopier-ci): " USER_NAME
    USER_NAME=${USER_NAME:-gopier-ci}
    
    # ユーザーが既に存在するかチェック
    if aws iam get-user --user-name "$USER_NAME" &>/dev/null; then
        log_warn "ユーザー '$USER_NAME' は既に存在します"
        read -p "このユーザーを使用しますか？ (y/n): " USE_EXISTING_USER
        if [[ $USE_EXISTING_USER =~ ^[Yy]$ ]]; then
            SELECTED_USER="$USER_NAME"
        else
            read -p "新しいユーザー名を入力してください: " USER_NAME
            SELECTED_USER="$USER_NAME"
        fi
    else
        # 新しいユーザーを作成
        log_info "ユーザー '$USER_NAME' を作成しています..."
        aws iam create-user --user-name "$USER_NAME"
        SELECTED_USER="$USER_NAME"
        
        # ポリシーを作成
        create_iam_policy "$USER_NAME"
        
        # ポリシーをアタッチ
        attach_iam_policy "$USER_NAME"
    fi
    
    # アクセスキーを作成
    create_new_access_key "$SELECTED_USER"
}

# IAMポリシーを作成
create_iam_policy() {
    local user_name="$1"
    local policy_name="${user_name}-policy"
    
    log_info "IAMポリシー '$policy_name' を作成しています..."
    
    # ポリシードキュメントを作成
    cat > /tmp/gopier-policy.json << EOF
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
EOF
    
    # ポリシーを作成
    aws iam create-policy --policy-name "$policy_name" --policy-document file:///tmp/gopier-policy.json
    
    # 一時ファイルを削除
    rm -f /tmp/gopier-policy.json
    
    log_info "IAMポリシー作成完了 ✅"
}

# IAMポリシーをアタッチ
attach_iam_policy() {
    local user_name="$1"
    local policy_name="${user_name}-policy"
    
    log_info "IAMポリシーをユーザーにアタッチしています..."
    
    # アカウントIDを取得
    local account_id=$(aws sts get-caller-identity --query Account --output text)
    
    # ポリシーをアタッチ
    aws iam attach-user-policy --user-name "$user_name" --policy-arn "arn:aws:iam::${account_id}:policy/${policy_name}"
    
    log_info "IAMポリシーアタッチ完了 ✅"
}

# 新しいアクセスキーを作成
create_new_access_key() {
    local user_name="$1"
    
    log_info "ユーザー '$user_name' の新しいアクセスキーを作成しています..."
    
    # 新しいアクセスキーを作成
    local key_output=$(aws iam create-access-key --user-name "$user_name")
    
    # アクセスキー情報を抽出
    AWS_ACCESS_KEY_ID=$(echo "$key_output" | jq -r '.AccessKey.AccessKeyId')
    AWS_SECRET_ACCESS_KEY=$(echo "$key_output" | jq -r '.AccessKey.SecretAccessKey')
    
    log_info "アクセスキー作成完了 ✅"
    log_info "Access Key ID: $AWS_ACCESS_KEY_ID"
    log_warn "Secret Access Key は安全に保管してください"
    
    # 変数が正しく設定されたか確認
    if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
        log_error "アクセスキーの設定に失敗しました"
        exit 1
    fi
}

# EC2 IAMロールの作成または確認
setup_ec2_iam_role() {
    log_info "EC2 IAMロールの設定を行います..."
    
    # ロールが既に存在するかチェック
    if aws iam get-role --role-name "$EC2_IAM_ROLE_NAME" &>/dev/null; then
        log_info "EC2 IAMロール '$EC2_IAM_ROLE_NAME' は既に存在します"
        return 0
    fi
    
    log_info "EC2 IAMロール '$EC2_IAM_ROLE_NAME' を作成しています..."
    
    # 信頼ポリシーを作成
    cat > /tmp/ec2-trust-policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com" },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
    
    # ロールを作成
    aws iam create-role --role-name "$EC2_IAM_ROLE_NAME" --assume-role-policy-document file:///tmp/ec2-trust-policy.json
    
    # EC2インスタンスプロファイルを作成
    aws iam create-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME"
    aws iam add-role-to-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME" --role-name "$EC2_IAM_ROLE_NAME"
    
    # EC2用ポリシーを作成
    cat > /tmp/ec2-policy.json << EOF
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
EOF
    
    # ポリシーを作成してアタッチ
    aws iam create-policy --policy-name "${EC2_IAM_ROLE_NAME}-policy" --policy-document file:///tmp/ec2-policy.json
    
    local account_id=$(aws sts get-caller-identity --query Account --output text)
    aws iam attach-role-policy --role-name "$EC2_IAM_ROLE_NAME" --policy-arn "arn:aws:iam::${account_id}:policy/${EC2_IAM_ROLE_NAME}-policy"
    
    # 一時ファイルを削除
    rm -f /tmp/ec2-trust-policy.json /tmp/ec2-policy.json
    
    log_info "EC2 IAMロール作成完了 ✅"
}

# セキュリティグループの作成または確認
setup_security_group() {
    log_info "セキュリティグループの設定を行います..."
    
    # 既存のセキュリティグループを確認
    EC2_SECURITY_GROUP_ID=$(aws ec2 describe-security-groups \
        --group-names gopier-test-runner \
        --query 'SecurityGroups[0].GroupId' \
        --output text \
        --region "$AWS_REGION" 2>/dev/null || echo "")
    
    if [ -n "$EC2_SECURITY_GROUP_ID" ]; then
        log_info "セキュリティグループ 'gopier-test-runner' が見つかりました: $EC2_SECURITY_GROUP_ID"
        return 0
    fi
    
    log_info "セキュリティグループ 'gopier-test-runner' を作成しています..."
    
    # セキュリティグループを作成
    local sg_output=$(aws ec2 create-security-group \
        --group-name gopier-test-runner \
        --description "Security group for gopier test runner" \
        --region "$AWS_REGION")
    
    EC2_SECURITY_GROUP_ID=$(echo "$sg_output" | jq -r '.GroupId')
    
    # アウトバウンドルールを設定
    aws ec2 authorize-security-group-egress \
        --group-id "$EC2_SECURITY_GROUP_ID" \
        --protocol -1 \
        --port -1 \
        --cidr 0.0.0.0/0 \
        --region "$AWS_REGION"
    
    log_info "セキュリティグループ作成完了 ✅"
    log_info "Security Group ID: $EC2_SECURITY_GROUP_ID"
}

# GitHub Personal Access Tokenの設定
setup_github_token() {
    log_info "GitHub Personal Access Tokenの設定を行います..."
    
    # 既存のトークンを確認
    log_info "既存のGitHub Personal Access Tokenを確認しています..."
    EXISTING_TOKENS=$(gh api user/tokens --jq '.[] | select(.note | contains("gopier") or contains("ci") or contains("runner")) | {id: .id, note: .note, created_at: .created_at}' 2>/dev/null || echo "")
    
    if [ -n "$EXISTING_TOKENS" ]; then
        echo "既存の関連トークンが見つかりました:"
        echo "$EXISTING_TOKENS" | jq -r '.note + " (作成日: " + .created_at + ")"'
        
        read -p "既存のトークンを使用しますか？ (y/n): " USE_EXISTING_TOKEN
        if [[ $USE_EXISTING_TOKEN =~ ^[Yy]$ ]]; then
            echo "使用可能なトークン:"
            echo "$EXISTING_TOKENS" | jq -r '.note'
            read -p "使用するトークンのメモ名を入力してください: " SELECTED_TOKEN_NOTE
            
            # 選択されたトークンのIDを取得
            TOKEN_ID=$(echo "$EXISTING_TOKENS" | jq -r "select(.note == \"$SELECTED_TOKEN_NOTE\") | .id")
            
            if [ -n "$TOKEN_ID" ] && [ "$TOKEN_ID" != "null" ]; then
                log_info "既存のトークンを使用します: $SELECTED_TOKEN_NOTE"
                read -s -p "トークンの値を入力してください: " GH_PERSONAL_ACCESS_TOKEN
                echo
                return 0
            else
                log_warn "指定されたトークンが見つかりませんでした"
            fi
        fi
    fi
    
    # 新しいトークンを作成
    create_new_github_token
}

# 新しいGitHub Personal Access Tokenを作成
create_new_github_token() {
    log_info "新しいGitHub Personal Access Tokenを作成します..."
    
    # トークン名を決定
    TOKEN_NOTE="gopier-ci-$(date +%Y%m%d-%H%M%S)"
    
    log_info "トークン名: $TOKEN_NOTE"
    log_info "必要な権限: repo, workflow"
    
    # トークンを作成
    log_info "GitHub Personal Access Tokenを作成しています..."
    
    # GitHub CLIでFine-grained Personal Access Tokenを作成を試行
    log_info "GitHub CLIでFine-grained Personal Access Tokenを作成を試行しています..."
    
    # 現在のリポジトリを取得
    CURRENT_REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
    
    # Fine-grained tokenを作成（複数の方法を試行）
    log_info "Fine-grained Personal Access Tokenを作成しています..."
    
    # 方法1: 基本的なFine-grained token作成
    TOKEN_OUTPUT=$(gh auth token create --scopes repo,workflow --expiry 90d --note "$TOKEN_NOTE" 2>/dev/null || echo "")
    
    if [ -n "$TOKEN_OUTPUT" ]; then
        GH_PERSONAL_ACCESS_TOKEN="$TOKEN_OUTPUT"
        log_info "Fine-grained Personal Access Token作成成功 ✅"
        return 0
    fi
    
    # 方法2: より詳細な権限でFine-grained token作成
    log_info "詳細な権限でFine-grained token作成を試行しています..."
    TOKEN_OUTPUT=$(gh auth token create \
        --scopes repo:status,repo_deployment,public_repo,repo:invite,security_events,workflow \
        --expiry 90d \
        --note "$TOKEN_NOTE" 2>/dev/null || echo "")
    
    if [ -n "$TOKEN_OUTPUT" ]; then
        GH_PERSONAL_ACCESS_TOKEN="$TOKEN_OUTPUT"
        log_info "Fine-grained Personal Access Token作成成功 ✅"
        return 0
    fi
    
    # 方法3: Classic token作成を試行
    log_info "Classic Personal Access Token作成を試行しています..."
    TOKEN_OUTPUT=$(gh auth token create \
        --scopes repo,workflow \
        --expiry 90d \
        --note "$TOKEN_NOTE" \
        --classic 2>/dev/null || echo "")
    
    if [ -n "$TOKEN_OUTPUT" ]; then
        GH_PERSONAL_ACCESS_TOKEN="$TOKEN_OUTPUT"
        log_info "Classic Personal Access Token作成成功 ✅"
        return 0
    fi
    
    # すべての自動作成に失敗した場合
    log_warn "自動トークン作成に失敗しました。手動作成が必要です"
    log_info "以下の手順でトークンを作成してください:"
    echo ""
    echo "1. GitHub.com にログイン"
    echo "2. Settings → Developer settings → Personal access tokens → Tokens (classic)"
    echo "3. 'Generate new token (classic)' をクリック"
    echo "4. Note: $TOKEN_NOTE"
    echo "5. Expiration: 90 days (推奨)"
    echo "6. Select scopes:"
    echo "   - [x] repo (Full control of private repositories)"
    echo "   - [x] workflow (Update GitHub Action workflows)"
    echo "7. 'Generate token' をクリック"
    echo ""
    
    read -s -p "作成したトークンの値を入力してください: " GH_PERSONAL_ACCESS_TOKEN
    echo
    
    if [ -z "$GH_PERSONAL_ACCESS_TOKEN" ]; then
        log_error "トークンが入力されていません"
        exit 1
    fi
    
    # トークンの検証
    log_info "トークンを検証しています..."
    if curl -s -H "Authorization: token $GH_PERSONAL_ACCESS_TOKEN" https://api.github.com/user | jq -e '.login' >/dev/null 2>&1; then
        log_info "トークン検証成功 ✅"
        
        # トークンの権限を確認
        log_info "トークンの権限を確認しています..."
        TOKEN_SCOPES=$(curl -s -H "Authorization: token $GH_PERSONAL_ACCESS_TOKEN" https://api.github.com/user | jq -r '.login' 2>/dev/null || echo "")
        
        if [ -n "$TOKEN_SCOPES" ]; then
            log_info "トークン権限確認成功 ✅"
        else
            log_warn "トークン権限の確認に失敗しましたが、基本的な認証は成功しています"
        fi
    else
        log_error "トークン検証に失敗しました"
        exit 1
    fi
}

# コスト監視設定
setup_cost_monitoring() {
    log_info "コスト監視設定を行います..."
    
    read -p "コスト監視とアラートを設定しますか？ (y/n): " SETUP_COST_MONITORING
    
    if [[ $SETUP_COST_MONITORING =~ ^[Yy]$ ]]; then
        log_info "コスト監視設定を開始します..."
        create_cost_monitoring
    else
        log_info "コスト監視設定をスキップします"
    fi
}

# コスト監視とアラートを作成
create_cost_monitoring() {
    log_info "コスト監視とアラートを作成しています..."
    
    # 1. SNSトピックの作成（アラート通知用）
    log_info "SNSトピックを作成しています..."
    aws sns create-topic --name "gopier-ci-alerts" --region "$AWS_REGION" 2>/dev/null || {
        log_info "SNSトピック 'gopier-ci-alerts' は既に存在します"
    }
    
    # 2. CloudWatchメトリクスアラーム（CPU使用率）
    log_info "CloudWatch CPU使用率アラームを作成しています..."
    aws cloudwatch put-metric-alarm \
        --alarm-name "gopier-ci-cpu-usage" \
        --alarm-description "CPU usage alert for gopier CI EC2 instances" \
        --metric-name CPUUtilization \
        --namespace AWS/EC2 \
        --statistic Average \
        --period 300 \
        --threshold 80 \
        --comparison-operator GreaterThanThreshold \
        --evaluation-periods 2 \
        --region "$AWS_REGION" 2>/dev/null || {
            log_warn "CPU使用率アラームの作成に失敗しました"
        }
    
    # 3. コストアラートの設定
    log_info "コストアラートを設定しています..."
    setup_cost_alerts
    
    # 4. CloudWatchダッシュボードの作成
    log_info "CloudWatchダッシュボードを作成しています..."
    create_cloudwatch_dashboard
    
    log_info "コスト監視設定完了 ✅"
}

# コストアラートを設定
setup_cost_alerts() {
    log_info "コストアラートを設定しています..."
    
    # 予算アラートの作成
    create_budget_alerts
}

# 予算アラートを作成
create_budget_alerts() {
    log_info "予算アラートを作成しています..."
    
    # 月間予算$100のアラート
    cat > /tmp/budget-alert.json << EOF
{
    "BudgetName": "gopier-ci-monthly-budget",
    "BudgetLimit": {
        "Amount": "100",
        "Unit": "USD"
    },
    "TimeUnit": "MONTHLY",
    "BudgetType": "COST",
    "NotificationsWithSubscribers": [
        {
            "Notification": {
                "ComparisonOperator": "GREATER_THAN",
                "NotificationType": "ACTUAL",
                "Threshold": 80,
                "ThresholdType": "PERCENTAGE"
            },
            "Subscribers": [
                {
                    "Address": "admin@example.com",
                    "SubscriptionType": "EMAIL"
                }
            ]
        },
        {
            "Notification": {
                "ComparisonOperator": "GREATER_THAN",
                "NotificationType": "ACTUAL",
                "Threshold": 100,
                "ThresholdType": "PERCENTAGE"
            },
            "Subscribers": [
                {
                    "Address": "admin@example.com",
                    "SubscriptionType": "EMAIL"
                }
            ]
        }
    ]
}
EOF
    
    aws budgets create-budget --account-id $(aws sts get-caller-identity --query Account --output text) --budget file:///tmp/budget-alert.json 2>/dev/null || {
        log_warn "予算アラートの作成に失敗しました（権限不足またはメールアドレス未設定の可能性があります）"
    }
    
    rm -f /tmp/budget-alert.json
}

# CloudWatchダッシュボードを作成
create_cloudwatch_dashboard() {
    log_info "CloudWatchダッシュボードを作成しています..."
    
    cat > /tmp/dashboard.json << EOF
{
    "widgets": [
        {
            "type": "metric",
            "x": 0,
            "y": 0,
            "width": 12,
            "height": 6,
            "properties": {
                "metrics": [
                    ["AWS/EC2", "CPUUtilization"]
                ],
                "view": "timeSeries",
                "stacked": false,
                "region": "$AWS_REGION",
                "title": "gopier CI - CPU Utilization",
                "period": 300
            }
        },
        {
            "type": "metric",
            "x": 12,
            "y": 0,
            "width": 12,
            "height": 6,
            "properties": {
                "metrics": [
                    ["AWS/EC2", "NetworkIn"]
                ],
                "view": "timeSeries",
                "stacked": false,
                "region": "$AWS_REGION",
                "title": "gopier CI - Network In",
                "period": 300
            }
        }
    ]
}
EOF
    
    aws cloudwatch put-dashboard \
        --dashboard-name "gopier-ci-monitoring" \
        --dashboard-body file:///tmp/dashboard.json \
        --region "$AWS_REGION" 2>/dev/null || {
        log_warn "CloudWatchダッシュボードの作成に失敗しました"
    }
    
    rm -f /tmp/dashboard.json
}

# ユーザー入力を受け取る
get_user_input() {
    log_info "必要な情報を入力してください..."
    
    # IAMユーザーの設定
    setup_iam_user
    
    # GitHub Personal Access Token
    setup_github_token
    
    # コスト監視設定
    setup_cost_monitoring
    
    # 入力値の検証
    log_info "設定された変数を確認しています..."
    log_info "AWS_ACCESS_KEY_ID: ${AWS_ACCESS_KEY_ID:0:10}..."
    log_info "AWS_SECRET_ACCESS_KEY: ${AWS_SECRET_ACCESS_KEY:0:10}..."
    log_info "GH_PERSONAL_ACCESS_TOKEN: ${GH_PERSONAL_ACCESS_TOKEN:0:10}..."
    
    if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ] || [ -z "$GH_PERSONAL_ACCESS_TOKEN" ]; then
        log_error "必要な情報が入力されていません"
        log_error "AWS_ACCESS_KEY_ID: $([ -z "$AWS_ACCESS_KEY_ID" ] && echo "未設定" || echo "設定済み")"
        log_error "AWS_SECRET_ACCESS_KEY: $([ -z "$AWS_SECRET_ACCESS_KEY" ] && echo "未設定" || echo "設定済み")"
        log_error "GH_PERSONAL_ACCESS_TOKEN: $([ -z "$GH_PERSONAL_ACCESS_TOKEN" ] && echo "未設定" || echo "設定済み")"
        exit 1
    fi
}

# GitHub Secretsを設定
set_github_secrets() {
    log_info "GitHub Secretsを設定しています..."
    
    # 現在のリポジトリを取得
    REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
    log_info "対象リポジトリ: $REPO"
    
    # Secrets設定
    log_info "AWS_ACCESS_KEY_ID を設定中..."
    echo "$AWS_ACCESS_KEY_ID" | gh secret set AWS_ACCESS_KEY_ID --repo "$REPO"
    
    log_info "AWS_SECRET_ACCESS_KEY を設定中..."
    echo "$AWS_SECRET_ACCESS_KEY" | gh secret set AWS_SECRET_ACCESS_KEY --repo "$REPO"
    
    log_info "AWS_REGION を設定中..."
    echo "$AWS_REGION" | gh secret set AWS_REGION --repo "$REPO"
    
    log_info "EC2_SECURITY_GROUP_ID を設定中..."
    echo "$EC2_SECURITY_GROUP_ID" | gh secret set EC2_SECURITY_GROUP_ID --repo "$REPO"
    
    log_info "EC2_SUBNET_ID を設定中..."
    echo "$EC2_SUBNET_ID" | gh secret set EC2_SUBNET_ID --repo "$REPO"
    
    log_info "EC2_IMAGE_ID を設定中..."
    echo "$EC2_IMAGE_ID" | gh secret set EC2_IMAGE_ID --repo "$REPO"
    
    log_info "EC2_INSTANCE_TYPE を設定中..."
    echo "$EC2_INSTANCE_TYPE" | gh secret set EC2_INSTANCE_TYPE --repo "$REPO"
    
    log_info "EC2_IAM_ROLE_NAME を設定中..."
    echo "$EC2_IAM_ROLE_NAME" | gh secret set EC2_IAM_ROLE_NAME --repo "$REPO"
    
    log_info "GH_PERSONAL_ACCESS_TOKEN を設定中..."
    echo "$GH_PERSONAL_ACCESS_TOKEN" | gh secret set GH_PERSONAL_ACCESS_TOKEN --repo "$REPO"
    
    log_info "GitHub Secrets設定完了 ✅"
}

# 設定確認
verify_secrets() {
    log_info "設定されたSecretsを確認しています..."
    
    gh secret list --repo "$REPO"
    
    log_info "設定完了！以下のSecretsが設定されました:"
    echo "  - AWS_ACCESS_KEY_ID"
    echo "  - AWS_SECRET_ACCESS_KEY"
    echo "  - AWS_REGION: $AWS_REGION"
    echo "  - EC2_SECURITY_GROUP_ID: $EC2_SECURITY_GROUP_ID"
    echo "  - EC2_SUBNET_ID: $EC2_SUBNET_ID"
    echo "  - EC2_IMAGE_ID: $EC2_IMAGE_ID"
    echo "  - EC2_INSTANCE_TYPE: $EC2_INSTANCE_TYPE"
    echo "  - EC2_IAM_ROLE_NAME: $EC2_IAM_ROLE_NAME"
    echo "  - GH_PERSONAL_ACCESS_TOKEN"
}

# メイン処理
main() {
    echo "=========================================="
    echo "  GitHub Secrets 自動設定スクリプト"
    echo "=========================================="
    
    check_prerequisites
    get_aws_info
    get_user_input
    set_github_secrets
    verify_secrets
    
    echo ""
    log_info "🎉 すべての設定が完了しました！"
    echo ""
    echo "次のステップ:"
    echo "1. GitHub ActionsワークフローでEC2ランナーを使用"
    echo "2. 大きなファイルテストを実行"
    if [[ $SETUP_COST_MONITORING =~ ^[Yy]$ ]]; then
        echo "3. コスト監視が設定されました"
        echo "   - CloudWatchダッシュボード: gopier-ci-monitoring"
        echo "   - 予算アラート: gopier-ci-monthly-budget"
        echo "   - CPU使用率アラーム: gopier-ci-cpu-usage"
    else
        echo "3. コスト監視を後から設定する場合:"
        echo "   ./scripts/setup-cost-monitoring.sh"
    fi
}

# スクリプト実行
main "$@" 