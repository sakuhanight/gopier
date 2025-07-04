#!/bin/bash

# EC2 Runner Helper Script
# EC2ランナーの管理を補助するスクリプト

set -euo pipefail

# 設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# ログ関数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

# ヘルプ表示
show_help() {
    cat << EOF
EC2 Runner Helper Script

使用方法:
    $0 <command> [options]

コマンド:
    monitor      - ランナーの状態を監視
    cleanup      - 古いランナーをクリーンアップ
    list         - アクティブなランナーを一覧表示
    health-check - ランナーの健全性チェック
    cost-report  - コストレポートを生成
    setup-iam    - IAM Instance Profileを設定
    verify-iam   - IAM設定を確認

オプション:
    --repository - GitHubリポジトリ (owner/repo形式)
    --timeout    - タイムアウト時間（分）(デフォルト: 30)
    --dry-run    - 実際の変更を行わずにシミュレーション
    --role-name  - IAMロール名 (setup-iam用、デフォルト: EC2RunnerRole)
    --profile-name - インスタンスプロファイル名 (setup-iam用、デフォルト: EC2RunnerRole)
    --help       - このヘルプを表示

環境変数:
    AWS_ACCESS_KEY_ID      - AWSアクセスキー
    AWS_SECRET_ACCESS_KEY  - AWSシークレットキー
    AWS_REGION            - AWSリージョン
    GITHUB_TOKEN          - GitHub Personal Access Token

例:
    $0 monitor --repository owner/repo
    $0 cleanup --dry-run
    $0 list
    $0 setup-iam --role-name MyRunnerRole --profile-name MyRunnerProfile
    $0 verify-iam
EOF
}

# パラメータの解析
parse_args() {
    COMMAND=""
    REPOSITORY=""
    TIMEOUT_MINUTES=30
    DRY_RUN=false
    IAM_ROLE_NAME="EC2RunnerRole"
    IAM_PROFILE_NAME="EC2RunnerRole"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            monitor|cleanup|list|health-check|cost-report|setup-iam|verify-iam)
                COMMAND="$1"
                shift
                ;;
            --repository)
                REPOSITORY="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT_MINUTES="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --role-name)
                IAM_ROLE_NAME="$2"
                shift 2
                ;;
            --profile-name)
                IAM_PROFILE_NAME="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                error "不明なオプション: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 設定の検証
validate_config() {
    local missing_vars=()
    
    # AWS認証情報の確認
    if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
        missing_vars+=("AWS_ACCESS_KEY_ID")
    fi
    if [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
        missing_vars+=("AWS_SECRET_ACCESS_KEY")
    fi
    if [[ -z "${AWS_REGION:-}" ]]; then
        missing_vars+=("AWS_REGION")
    fi
    
    # GitHub認証情報の確認
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        missing_vars+=("GITHUB_TOKEN")
    fi
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        error "以下の環境変数が設定されていません: ${missing_vars[*]}"
        exit 1
    fi
}

# AWS CLIの確認
check_aws_cli() {
    if ! command -v aws &> /dev/null; then
        error "AWS CLIがインストールされていません"
        exit 1
    fi
}

# ランナーの監視
monitor_runners() {
    log "ランナーの状態を監視中..."
    
    local repository="${REPOSITORY:-$GITHUB_REPOSITORY}"
    if [[ -z "$repository" ]]; then
        error "リポジトリが指定されていません"
        exit 1
    fi
    
    # GitHubランナーの一覧を取得
    local runners
    runners=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$repository/actions/runners" | \
        jq -r '.runners[] | "\(.id),\(.name),\(.status),\(.busy),\(.created_at)"' 2>/dev/null || echo "")
    
    if [[ -z "$runners" ]]; then
        log "ランナーが見つかりません"
        return 0
    fi
    
    echo "=== GitHub Runners Status ==="
    echo "ID,Name,Status,Busy,Created"
    echo "$runners" | while IFS=',' read -r id name status busy created; do
        if [[ -n "$id" && "$id" != "null" ]]; then
            echo "$id,$name,$status,$busy,$created"
        fi
    done
    
    # EC2インスタンスの状態を確認
    echo ""
    echo "=== EC2 Instances Status ==="
    local instances
    instances=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=$repository" "Name=instance-state-name,Values=running,stopping,stopped" \
        --query 'Reservations[].Instances[].[InstanceId,State.Name,Tags[?Key==`RunnerLabel`].Value|[0],LaunchTime]' \
        --output text)
    
    if [[ -z "$instances" ]]; then
        log "EC2インスタンスが見つかりません"
    else
        echo "Instance ID,State,Runner Label,Launch Time"
        echo "$instances" | while read -r instance_id state label launch_time; do
            if [[ -n "$instance_id" && "$instance_id" != "None" ]]; then
                echo "$instance_id,$state,$label,$launch_time"
            fi
        done
    fi
}

# 古いランナーのクリーンアップ
cleanup_old_runners() {
    log "古いランナーをクリーンアップ中..."
    
    local repository="${REPOSITORY:-$GITHUB_REPOSITORY}"
    if [[ -z "$repository" ]]; then
        error "リポジトリが指定されていません"
        exit 1
    fi
    
    # 24時間以上経過したインスタンスを検索
    local old_instances
    old_instances=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=$repository" "Name=instance-state-name,Values=running" \
        --query 'Reservations[].Instances[?LaunchTime<`'$(date -d '24 hours ago' -u +%Y-%m-%dT%H:%M:%S)'`].[InstanceId,Tags[?Key==`RunnerLabel`].Value|[0]]' \
        --output text)
    
    if [[ -z "$old_instances" ]]; then
        log "クリーンアップ対象のインスタンスはありません"
        return 0
    fi
    
    echo "クリーンアップ対象のインスタンス:"
    echo "$old_instances" | while read -r instance_id label; do
        if [[ -n "$instance_id" && "$instance_id" != "None" ]]; then
            echo "  - $instance_id ($label)"
        fi
    done
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DRY RUN: 実際の削除は実行されません"
        return 0
    fi
    
    echo "$old_instances" | while read -r instance_id label; do
        if [[ -n "$instance_id" && "$instance_id" != "None" ]]; then
            log "古いインスタンスを停止中: $instance_id ($label)"
            
            # GitHubランナーの削除
            local runner_id
            runner_id=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
                "https://api.github.com/repos/$repository/actions/runners" | \
                jq -r '.runners[] | select(.name == "'$label'") | .id' 2>/dev/null || echo "")
            
            if [[ -n "$runner_id" && "$runner_id" != "null" ]]; then
                curl -X DELETE -H "Authorization: token $GITHUB_TOKEN" \
                    "https://api.github.com/repos/$repository/actions/runners/$runner_id"
                log "GitHubランナーを削除しました: $runner_id"
            fi
            
            # EC2インスタンスの停止
            aws ec2 terminate-instances --instance-ids "$instance_id"
            log "インスタンス停止要求を送信しました: $instance_id"
        fi
    done
}

# ランナーの一覧表示
list_runners() {
    log "アクティブなランナーを一覧表示中..."
    
    local repository="${REPOSITORY:-$GITHUB_REPOSITORY}"
    if [[ -z "$repository" ]]; then
        error "リポジトリが指定されていません"
        exit 1
    fi
    
    # GitHubランナーの一覧
    echo "=== GitHub Runners ==="
    local runners
    runners=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$repository/actions/runners" | \
        jq -r '.runners[] | "\(.name) (\(.status)) - Busy: \(.busy)"' 2>/dev/null || echo "")
    
    if [[ -z "$runners" ]]; then
        echo "GitHubランナーが見つかりません"
    else
        echo "$runners"
    fi
    
    # EC2インスタンスの一覧
    echo ""
    echo "=== EC2 Instances ==="
    local instances
    instances=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=$repository" \
        --query 'Reservations[].Instances[].[InstanceId,State.Name,Tags[?Key==`RunnerLabel`].Value|[0],InstanceType,LaunchTime]' \
        --output text)
    
    if [[ -z "$instances" ]]; then
        echo "EC2インスタンスが見つかりません"
    else
        echo "Instance ID,State,Runner Label,Type,Launch Time"
        echo "$instances" | while read -r instance_id state label type launch_time; do
            if [[ -n "$instance_id" && "$instance_id" != "None" ]]; then
                echo "$instance_id,$state,$label,$type,$launch_time"
            fi
        done
    fi
}

# ランナーの健全性チェック
health_check() {
    log "ランナーの健全性チェックを実行中..."
    
    local repository="${REPOSITORY:-$GITHUB_REPOSITORY}"
    if [[ -z "$repository" ]]; then
        error "リポジトリが指定されていません"
        exit 1
    fi
    
    local issues_found=false
    
    # GitHubランナーの状態チェック
    local runners
    runners=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$repository/actions/runners" | \
        jq -r '.runners[] | "\(.name),\(.status)"' 2>/dev/null || echo "")
    
    echo "=== GitHub Runners Health Check ==="
    echo "$runners" | while IFS=',' read -r name status; do
        if [[ -n "$name" && "$name" != "null" ]]; then
            if [[ "$status" != "online" ]]; then
                echo "⚠️  $name: $status (offline)"
                issues_found=true
            else
                echo "✅ $name: $status"
            fi
        fi
    done
    
    # EC2インスタンスの状態チェック
    echo ""
    echo "=== EC2 Instances Health Check ==="
    local instances
    instances=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=$repository" \
        --query 'Reservations[].Instances[].[InstanceId,State.Name,Tags[?Key==`RunnerLabel`].Value|[0]]' \
        --output text)
    
    echo "$instances" | while read -r instance_id state label; do
        if [[ -n "$instance_id" && "$instance_id" != "None" ]]; then
            if [[ "$state" != "running" ]]; then
                echo "⚠️  $instance_id ($label): $state"
                issues_found=true
            else
                echo "✅ $instance_id ($label): $state"
            fi
        fi
    done
    
    if [[ "$issues_found" == "true" ]]; then
        echo ""
        echo "⚠️  健全性チェックで問題が見つかりました"
        return 1
    else
        echo ""
        echo "✅ すべてのランナーが正常です"
    fi
}

# IAMロールの作成
create_iam_role() {
    local role_name="$1"
    log "IAMロールを作成中: $role_name"
    
    # 信頼ポリシー
    local trust_policy='{
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
    
    # ロールの作成
    if aws iam get-role --role-name "$role_name" &> /dev/null; then
        log "IAMロール $role_name は既に存在します"
    else
        aws iam create-role \
            --role-name "$role_name" \
            --assume-role-policy-document "$trust_policy"
        log "IAMロール $role_name を作成しました"
    fi
}

# IAMポリシーの作成とアタッチ
create_and_attach_policies() {
    local role_name="$1"
    log "IAMポリシーを作成・アタッチ中..."
    
    # カスタムポリシーの作成
    local custom_policy='{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "ec2:DescribeInstances",
                    "ec2:DescribeSecurityGroups",
                    "ec2:DescribeSubnets",
                    "ec2:DescribeImages",
                    "ec2:DescribeTags",
                    "ec2:DescribeInstanceStatus"
                ],
                "Resource": "*"
            },
            {
                "Effect": "Allow",
                "Action": [
                    "logs:CreateLogGroup",
                    "logs:CreateLogStream",
                    "logs:PutLogEvents"
                ],
                "Resource": "arn:aws:logs:*:*:*"
            }
        ]
    }'
    
    local policy_name="${role_name}Policy"
    
    # ポリシーの作成
    if aws iam get-policy --policy-arn "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):policy/$policy_name" &> /dev/null; then
        log "IAMポリシー $policy_name は既に存在します"
    else
        aws iam create-policy \
            --policy-name "$policy_name" \
            --policy-document "$custom_policy"
        log "IAMポリシー $policy_name を作成しました"
    fi
    
    # ポリシーのアタッチ
    aws iam attach-role-policy \
        --role-name "$role_name" \
        --policy-arn "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):policy/$policy_name"
    log "IAMポリシー $policy_name をロール $role_name にアタッチしました"
    
    # 管理ポリシーのアタッチ
    aws iam attach-role-policy \
        --role-name "$role_name" \
        --policy-arn "arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess"
    log "AmazonEC2ReadOnlyAccessポリシーをロール $role_name にアタッチしました"
}

# インスタンスプロファイルの作成
create_instance_profile() {
    local role_name="$1"
    local profile_name="$2"
    log "インスタンスプロファイルを作成中: $profile_name"
    
    # インスタンスプロファイルの作成
    if aws iam get-instance-profile --instance-profile-name "$profile_name" &> /dev/null; then
        log "インスタンスプロファイル $profile_name は既に存在します"
    else
        aws iam create-instance-profile \
            --instance-profile-name "$profile_name"
        log "インスタンスプロファイル $profile_name を作成しました"
    fi
    
    # ロールをインスタンスプロファイルに追加
    if aws iam get-instance-profile --instance-profile-name "$profile_name" | jq -e ".InstanceProfile.Roles[] | select(.RoleName == \"$role_name\")" &> /dev/null; then
        log "ロール $role_name は既にインスタンスプロファイル $profile_name に追加されています"
    else
        aws iam add-role-to-instance-profile \
            --instance-profile-name "$profile_name" \
            --role-name "$role_name"
        log "ロール $role_name をインスタンスプロファイル $profile_name に追加しました"
    fi
}

# IAM設定の確認
verify_iam_setup() {
    local profile_name="$1"
    log "IAM設定を確認中..."
    
    # インスタンスプロファイルの確認
    local profile_arn
    profile_arn=$(aws iam get-instance-profile --instance-profile-name "$profile_name" --query 'InstanceProfile.Arn' --output text 2>/dev/null || echo "")
    
    if [[ -n "$profile_arn" && "$profile_arn" != "None" ]]; then
        log "✅ インスタンスプロファイル $profile_name が正常に設定されました"
        log "   ARN: $profile_arn"
        log "   GitHub Secretsで使用する値: $profile_name"
        return 0
    else
        error "❌ インスタンスプロファイル $profile_name が見つかりません"
        return 1
    fi
}

# IAM設定
setup_iam() {
    log "IAM Instance Profileを設定中..."
    
    # AWS CLIの確認
    check_aws_cli
    
    # アカウント情報の表示
    local account_id
    account_id=$(aws sts get-caller-identity --query 'Account' --output text)
    log "AWS Account ID: $account_id"
    log "Region: $AWS_REGION"
    
    # 設定の実行
    create_iam_role "$IAM_ROLE_NAME"
    create_and_attach_policies "$IAM_ROLE_NAME"
    create_instance_profile "$IAM_ROLE_NAME" "$IAM_PROFILE_NAME"
    verify_iam_setup "$IAM_PROFILE_NAME"
    
    log "IAM Instance Profileの設定が完了しました"
    log ""
    log "GitHub Secretsで以下の値を設定してください:"
    log "  EC2_IAM_ROLE_NAME: $IAM_PROFILE_NAME"
    log ""
    log "注意: インスタンスプロファイルの作成後、反映まで数分かかる場合があります"
}

# IAM設定の確認
verify_iam() {
    log "IAM設定を確認中..."
    
    # AWS CLIの確認
    check_aws_cli
    
    # デフォルトのプロファイル名を使用
    local profile_name="${IAM_PROFILE_NAME:-EC2RunnerRole}"
    
    if verify_iam_setup "$profile_name"; then
        log "✅ IAM設定は正常です"
    else
        error "❌ IAM設定に問題があります"
        log "以下のコマンドでIAM設定を実行してください:"
        log "  $0 setup-iam --role-name $profile_name --profile-name $profile_name"
        exit 1
    fi
}

# コストレポートの生成
cost_report() {
    log "コストレポートを生成中..."
    
    local repository="${REPOSITORY:-$GITHUB_REPOSITORY}"
    if [[ -z "$repository" ]]; then
        error "リポジトリが指定されていません"
        exit 1
    fi
    
    # 過去30日間のコストを取得
    local start_date=$(date -d '30 days ago' +%Y-%m-%d)
    local end_date=$(date +%Y-%m-%d)
    
    echo "=== Cost Report ($start_date to $end_date) ==="
    
    # EC2インスタンスのコスト
    local ec2_cost
    ec2_cost=$(aws ce get-cost-and-usage \
        --time-period Start="$start_date",End="$end_date" \
        --granularity MONTHLY \
        --metrics BlendedCost \
        --group-by Type=DIMENSION,Key=SERVICE \
        --filter '{"Dimensions": {"Key": "SERVICE", "Values": ["Amazon Elastic Compute Cloud - Compute"]}}' \
        --query 'ResultsByTime[0].Groups[0].Metrics.BlendedCost.Amount' \
        --output text 2>/dev/null || echo "0")
    
    echo "EC2 Cost: \$$ec2_cost"
    
    # インスタンスタイプ別の使用時間
    echo ""
    echo "=== Instance Usage ==="
    local instance_usage
    instance_usage=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=$repository" \
        --query 'Reservations[].Instances[].[InstanceType,Tags[?Key==`RunnerLabel`].Value|[0],LaunchTime,State.Name]' \
        --output text)
    
    if [[ -n "$instance_usage" ]]; then
        echo "Type,Label,Launch Time,State"
        echo "$instance_usage" | while read -r type label launch_time state; do
            if [[ -n "$type" && "$type" != "None" ]]; then
                echo "$type,$label,$launch_time,$state"
            fi
        done
    else
        echo "インスタンスの使用履歴が見つかりません"
    fi
}

# メイン処理
main() {
    # 引数の解析
    parse_args "$@"
    
    # コマンドの確認
    if [[ -z "$COMMAND" ]]; then
        error "コマンドを指定してください"
        show_help
        exit 1
    fi
    
    # 設定の検証
    validate_config
    
    # AWS CLIの確認
    check_aws_cli
    
    # コマンドの実行
    case "$COMMAND" in
        monitor)
            monitor_runners
            ;;
        cleanup)
            cleanup_old_runners
            ;;
        list)
            list_runners
            ;;
        health-check)
            health_check
            ;;
        cost-report)
            cost_report
            ;;
        setup-iam)
            setup_iam
            ;;
        verify-iam)
            verify_iam
            ;;
        *)
            error "不明なコマンド: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

# スクリプトの実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 