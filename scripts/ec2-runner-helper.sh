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

オプション:
    --repository - GitHubリポジトリ (owner/repo形式)
    --timeout    - タイムアウト時間（分）(デフォルト: 30)
    --dry-run    - 実際の変更を行わずにシミュレーション
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
EOF
}

# パラメータの解析
parse_args() {
    COMMAND=""
    REPOSITORY=""
    TIMEOUT_MINUTES=30
    DRY_RUN=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            monitor|cleanup|list|health-check|cost-report)
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