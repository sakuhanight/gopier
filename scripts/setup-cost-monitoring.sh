#!/bin/bash

# コスト監視自動設定スクリプト
# 使用方法: ./scripts/setup-cost-monitoring.sh

set -e

echo "💰 コスト監視自動設定を開始します..."

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
    
    if ! command -v jq &> /dev/null; then
        log_error "jqがインストールされていません"
        exit 1
    fi
    
    # AWS認証チェック
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS認証が設定されていません"
        exit 1
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
    
    # プロジェクト名
    read -p "プロジェクト名 (デフォルト: gopier-ci): " PROJECT_NAME
    PROJECT_NAME=${PROJECT_NAME:-gopier-ci}
    log_info "Project Name: $PROJECT_NAME"
}

# コスト監視設定オプション
setup_cost_monitoring_options() {
    log_info "コスト監視設定オプションを選択してください..."
    
    echo ""
    echo "設定可能な監視項目:"
    echo "1. CloudWatch CPU使用率アラーム"
    echo "2. 月間予算アラート"
    echo "3. CloudWatchダッシュボード"
    echo "4. SNS通知設定"
    echo "5. すべて設定"
    echo ""
    
    read -p "設定する項目を選択してください (1-5): " MONITORING_OPTION
    
    case $MONITORING_OPTION in
        1)
            create_cpu_alarm
            ;;
        2)
            create_budget_alerts
            ;;
        3)
            create_cloudwatch_dashboard
            ;;
        4)
            setup_sns_notifications
            ;;
        5)
            create_all_monitoring
            ;;
        *)
            log_error "無効な選択です"
            exit 1
            ;;
    esac
}

# CPU使用率アラームを作成
create_cpu_alarm() {
    log_info "CloudWatch CPU使用率アラームを作成しています..."
    
    read -p "CPU使用率の閾値 (デフォルト: 80): " CPU_THRESHOLD
    CPU_THRESHOLD=${CPU_THRESHOLD:-80}
    
    aws cloudwatch put-metric-alarm \
        --alarm-name "${PROJECT_NAME}-cpu-usage" \
        --alarm-description "CPU usage alert for ${PROJECT_NAME} EC2 instances" \
        --metric-name CPUUtilization \
        --namespace AWS/EC2 \
        --statistic Average \
        --period 300 \
        --threshold "$CPU_THRESHOLD" \
        --comparison-operator GreaterThanThreshold \
        --evaluation-periods 2 \
        --region "$AWS_REGION" 2>/dev/null || {
            log_warn "CPU使用率アラームの作成に失敗しました"
            return 1
        }
    
    log_info "CPU使用率アラーム作成完了 ✅"
    log_info "アラーム名: ${PROJECT_NAME}-cpu-usage"
    log_info "閾値: ${CPU_THRESHOLD}%"
}

# 予算アラートを作成
create_budget_alerts() {
    log_info "予算アラートを作成しています..."
    
    read -p "月間予算額 (USD, デフォルト: 100): " BUDGET_AMOUNT
    BUDGET_AMOUNT=${BUDGET_AMOUNT:-100}
    
    read -p "通知メールアドレス: " NOTIFICATION_EMAIL
    
    if [ -z "$NOTIFICATION_EMAIL" ]; then
        log_warn "メールアドレスが入力されていません。デフォルトのadmin@example.comを使用します"
        NOTIFICATION_EMAIL="admin@example.com"
    fi
    
    # 予算アラートのJSONを作成
    cat > /tmp/budget-alert.json << EOF
{
    "BudgetName": "${PROJECT_NAME}-monthly-budget",
    "BudgetLimit": {
        "Amount": "${BUDGET_AMOUNT}",
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
                    "Address": "${NOTIFICATION_EMAIL}",
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
                    "Address": "${NOTIFICATION_EMAIL}",
                    "SubscriptionType": "EMAIL"
                }
            ]
        }
    ]
}
EOF
    
    aws budgets create-budget --account-id "$AWS_ACCOUNT_ID" --budget file:///tmp/budget-alert.json 2>/dev/null || {
        log_warn "予算アラートの作成に失敗しました（権限不足またはメールアドレス未設定の可能性があります）"
        rm -f /tmp/budget-alert.json
        return 1
    }
    
    rm -f /tmp/budget-alert.json
    log_info "予算アラート作成完了 ✅"
    log_info "予算名: ${PROJECT_NAME}-monthly-budget"
    log_info "予算額: $${BUDGET_AMOUNT}/月"
    log_info "通知先: ${NOTIFICATION_EMAIL}"
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
                "title": "${PROJECT_NAME} - CPU Utilization",
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
                "title": "${PROJECT_NAME} - Network In",
                "period": 300
            }
        },
        {
            "type": "metric",
            "x": 0,
            "y": 6,
            "width": 12,
            "height": 6,
            "properties": {
                "metrics": [
                    ["AWS/EC2", "NetworkOut"]
                ],
                "view": "timeSeries",
                "stacked": false,
                "region": "$AWS_REGION",
                "title": "${PROJECT_NAME} - Network Out",
                "period": 300
            }
        },
        {
            "type": "metric",
            "x": 12,
            "y": 6,
            "width": 12,
            "height": 6,
            "properties": {
                "metrics": [
                    ["AWS/EC2", "DiskReadOps"]
                ],
                "view": "timeSeries",
                "stacked": false,
                "region": "$AWS_REGION",
                "title": "${PROJECT_NAME} - Disk Read Ops",
                "period": 300
            }
        }
    ]
}
EOF
    
    aws cloudwatch put-dashboard \
        --dashboard-name "${PROJECT_NAME}-monitoring" \
        --dashboard-body file:///tmp/dashboard.json \
        --region "$AWS_REGION" 2>/dev/null || {
        log_warn "CloudWatchダッシュボードの作成に失敗しました"
        rm -f /tmp/dashboard.json
        return 1
    }
    
    rm -f /tmp/dashboard.json
    log_info "CloudWatchダッシュボード作成完了 ✅"
    log_info "ダッシュボード名: ${PROJECT_NAME}-monitoring"
}

# SNS通知設定
setup_sns_notifications() {
    log_info "SNS通知設定を行います..."
    
    # SNSトピックの作成
    log_info "SNSトピックを作成しています..."
    aws sns create-topic --name "${PROJECT_NAME}-alerts" --region "$AWS_REGION" 2>/dev/null || {
        log_info "SNSトピック '${PROJECT_NAME}-alerts' は既に存在します"
    }
    
    read -p "通知メールアドレス: " NOTIFICATION_EMAIL
    
    if [ -n "$NOTIFICATION_EMAIL" ]; then
        # SNSサブスクリプションを作成
        aws sns subscribe \
            --topic-arn "arn:aws:sns:${AWS_REGION}:${AWS_ACCOUNT_ID}:${PROJECT_NAME}-alerts" \
            --protocol email \
            --notification-endpoint "$NOTIFICATION_EMAIL" \
            --region "$AWS_REGION" 2>/dev/null || {
                log_warn "SNSサブスクリプションの作成に失敗しました"
                return 1
            }
        
        log_info "SNS通知設定完了 ✅"
        log_info "トピック名: ${PROJECT_NAME}-alerts"
        log_info "通知先: ${NOTIFICATION_EMAIL}"
        log_info "確認メールが送信されます。メール内のリンクをクリックしてサブスクリプションを有効にしてください。"
    else
        log_warn "メールアドレスが入力されていません。SNSトピックのみ作成しました。"
    fi
}

# すべての監視設定を作成
create_all_monitoring() {
    log_info "すべての監視設定を作成しています..."
    
    # SNS通知設定
    setup_sns_notifications
    
    # CPU使用率アラーム
    create_cpu_alarm
    
    # 予算アラート
    create_budget_alerts
    
    # CloudWatchダッシュボード
    create_cloudwatch_dashboard
    
    log_info "すべての監視設定完了 ✅"
}

# 設定確認
verify_monitoring() {
    log_info "設定された監視項目を確認しています..."
    
    echo ""
    echo "設定された監視項目:"
    echo "=================="
    
    # CloudWatchアラーム確認
    if aws cloudwatch describe-alarms --alarm-names "${PROJECT_NAME}-cpu-usage" --region "$AWS_REGION" &>/dev/null; then
        echo "✅ CPU使用率アラーム: ${PROJECT_NAME}-cpu-usage"
    else
        echo "❌ CPU使用率アラーム: 未設定"
    fi
    
    # 予算確認
    if aws budgets describe-budgets --account-id "$AWS_ACCOUNT_ID" --query "Budgets[?BudgetName=='${PROJECT_NAME}-monthly-budget'].BudgetName" --output text | grep -q "${PROJECT_NAME}-monthly-budget"; then
        echo "✅ 予算アラート: ${PROJECT_NAME}-monthly-budget"
    else
        echo "❌ 予算アラート: 未設定"
    fi
    
    # ダッシュボード確認
    if aws cloudwatch list-dashboards --region "$AWS_REGION" --query "DashboardEntries[?DashboardName=='${PROJECT_NAME}-monitoring'].DashboardName" --output text | grep -q "${PROJECT_NAME}-monitoring"; then
        echo "✅ CloudWatchダッシュボード: ${PROJECT_NAME}-monitoring"
    else
        echo "❌ CloudWatchダッシュボード: 未設定"
    fi
    
    # SNSトピック確認
    if aws sns list-topics --region "$AWS_REGION" --query "Topics[?contains(TopicArn, '${PROJECT_NAME}-alerts')].TopicArn" --output text | grep -q "${PROJECT_NAME}-alerts"; then
        echo "✅ SNSトピック: ${PROJECT_NAME}-alerts"
    else
        echo "❌ SNSトピック: 未設定"
    fi
}

# メイン処理
main() {
    echo "=========================================="
    echo "  コスト監視自動設定スクリプト"
    echo "=========================================="
    
    check_prerequisites
    get_aws_info
    setup_cost_monitoring_options
    verify_monitoring
    
    echo ""
    log_info "🎉 コスト監視設定が完了しました！"
    echo ""
    echo "次のステップ:"
    echo "1. AWS CloudWatchコンソールでダッシュボードを確認"
    echo "2. 予算アラートのメール確認"
    echo "3. SNSサブスクリプションの確認メールを確認"
    echo "4. 必要に応じてアラート設定を調整"
}

# スクリプト実行
main "$@" 