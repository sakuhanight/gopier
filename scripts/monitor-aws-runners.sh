#!/bin/bash

# AWSランナー監視スクリプト
# 使用方法: ./scripts/monitor-aws-runners.sh [オプション]
# オプション:
#   --cost-analysis     : コスト分析を実行
#   --performance       : パフォーマンスメトリクスを表示
#   --security          : セキュリティ状態を確認
#   --all               : すべての監視を実行
#   --json              : JSON形式で出力

set -e

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

log_success() {
    echo -e "\033[36m[SUCCESS]\033[0m $1"
}

# 設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="gopier"
DEFAULT_REGION="ap-northeast-1"
OUTPUT_FORMAT="text"

# オプション解析
COST_ANALYSIS=false
PERFORMANCE=false
SECURITY=false
ALL_MONITORING=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --cost-analysis)
            COST_ANALYSIS=true
            shift
            ;;
        --performance)
            PERFORMANCE=true
            shift
            ;;
        --security)
            SECURITY=true
            shift
            ;;
        --all)
            ALL_MONITORING=true
            shift
            ;;
        --json)
            OUTPUT_FORMAT="json"
            shift
            ;;
        --help|-h)
            echo "使用方法: $0 [オプション]"
            echo "オプション:"
            echo "  --cost-analysis     : コスト分析を実行"
            echo "  --performance       : パフォーマンスメトリクスを表示"
            echo "  --security          : セキュリティ状態を確認"
            echo "  --all               : すべての監視を実行"
            echo "  --json              : JSON形式で出力"
            echo "  --help, -h          : このヘルプを表示"
            exit 0
            ;;
        *)
            log_error "不明なオプション: $1"
            exit 1
            ;;
    esac
done

# デフォルトで全ての監視を実行
if [[ "$COST_ANALYSIS" == false && "$PERFORMANCE" == false && "$SECURITY" == false && "$ALL_MONITORING" == false ]]; then
    ALL_MONITORING=true
fi

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
    
    log_success "前提条件チェック完了"
}

# AWS情報を取得
get_aws_info() {
    log_info "AWS情報を取得しています..."
    
    # アカウントID
    AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    
    # リージョン
    AWS_REGION=$(aws configure get region || echo "$DEFAULT_REGION")
    
    # ユーザー情報
    AWS_USER=$(aws sts get-caller-identity --query Arn --output text)
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        echo "{\"account_id\": \"$AWS_ACCOUNT_ID\", \"region\": \"$AWS_REGION\", \"user\": \"$AWS_USER\"}"
    else
        echo "=== AWS情報 ==="
        echo "アカウントID: $AWS_ACCOUNT_ID"
        echo "リージョン: $AWS_REGION"
        echo "ユーザー: $AWS_USER"
        echo ""
    fi
}

# EC2インスタンスの状態を確認
check_ec2_instances() {
    log_info "EC2インスタンスの状態を確認しています..."
    
    local instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*$PROJECT_NAME*" \
        --query 'Reservations[*].Instances[*]' \
        --output json)
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        echo "$instances"
    else
        echo "=== EC2インスタンス状態 ==="
        
        local instance_count=$(echo "$instances" | jq -r '.[] | length')
        if [[ "$instance_count" -eq 0 ]]; then
            log_warn "実行中のEC2インスタンスが見つかりません"
            return
        fi
        
        echo "$instances" | jq -r '.[] | .[] | "インスタンスID: \(.InstanceId)" + "\n状態: \(.State.Name)" + "\nタイプ: \(.InstanceType)" + "\n起動時刻: \(.LaunchTime)" + "\nタグ: \(.Tags // [] | map("\(.Key)=\(.Value)") | join(", "))" + "\n---"'
    fi
}

# ランナーの使用状況を確認
check_runner_usage() {
    log_info "GitHubランナーの使用状況を確認しています..."
    
    # 最近のワークフロー実行を確認
    local workflow_runs=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=*$PROJECT_NAME*" \
        --query 'Reservations[*].Instances[*].[InstanceId,State.Name,LaunchTime,Tags[?Key==`TestType`].Value|[0]]' \
        --output json)
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        echo "$workflow_runs"
    else
        echo "=== ランナー使用状況 ==="
        
        local runner_count=$(echo "$workflow_runs" | jq -r '.[] | length')
        if [[ "$runner_count" -eq 0 ]]; then
            log_warn "GitHubランナーインスタンスが見つかりません"
            return
        fi
        
        echo "$workflow_runs" | jq -r '.[] | .[] | "インスタンスID: \(.[0])" + "\n状態: \(.[1])" + "\n起動時刻: \(.[2])" + "\nテストタイプ: \(.[3] // "不明")" + "\n---"'
    fi
}

# コスト分析
analyze_costs() {
    if [[ "$COST_ANALYSIS" == false && "$ALL_MONITORING" == false ]]; then
        return
    fi
    
    log_info "コスト分析を実行しています..."
    
    # 今月のコスト
    local current_month=$(date +%Y-%m)
    local start_date="${current_month}-01"
    local end_date=$(date +%Y-%m-%d)
    
    local cost_data=$(aws ce get-cost-and-usage \
        --time-period Start="$start_date",End="$end_date" \
        --granularity MONTHLY \
        --metrics BlendedCost \
        --group-by Type=DIMENSION,Key=SERVICE \
        --output json 2>/dev/null || echo '{"ResultsByTime": []}')
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        echo "$cost_data"
    else
        echo "=== コスト分析 (${start_date} 〜 ${end_date}) ==="
        
        local total_cost=$(echo "$cost_data" | jq -r '.ResultsByTime[0].Total.BlendedCost.Amount // "0"')
        local currency=$(echo "$cost_data" | jq -r '.ResultsByTime[0].Total.BlendedCost.Unit // "USD"')
        
        echo "総コスト: $total_cost $currency"
        echo ""
        echo "サービス別コスト:"
        echo "$cost_data" | jq -r '.ResultsByTime[0].Groups[]? | "\(.Keys[0]): \(.Metrics.BlendedCost.Amount) \(.Metrics.BlendedCost.Unit)"'
    fi
}

# パフォーマンスメトリクス
check_performance() {
    if [[ "$PERFORMANCE" == false && "$ALL_MONITORING" == false ]]; then
        return
    fi
    
    log_info "パフォーマンスメトリクスを取得しています..."
    
    # 実行中のインスタンスのメトリクスを取得
    local instance_ids=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*$PROJECT_NAME*" "Name=instance-state-name,Values=running" \
        --query 'Reservations[*].Instances[*].InstanceId' \
        --output text)
    
    if [[ -z "$instance_ids" ]]; then
        log_warn "実行中のインスタンスがありません"
        return
    fi
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        local metrics_data="{}"
        for instance_id in $instance_ids; do
            local cpu_metric=$(aws cloudwatch get-metric-statistics \
                --namespace AWS/EC2 \
                --metric-name CPUUtilization \
                --dimensions Name=InstanceId,Value="$instance_id" \
                --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
                --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
                --period 300 \
                --statistics Average \
                --output json 2>/dev/null || echo '{"Datapoints": []}')
            
            local memory_metric=$(aws cloudwatch get-metric-statistics \
                --namespace System/Linux \
                --metric-name MemoryUtilization \
                --dimensions Name=InstanceId,Value="$instance_id" \
                --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
                --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
                --period 300 \
                --statistics Average \
                --output json 2>/dev/null || echo '{"Datapoints": []}')
            
            metrics_data=$(echo "$metrics_data" | jq --arg id "$instance_id" --argjson cpu "$cpu_metric" --argjson mem "$memory_metric" '. + {($id): {"cpu": $cpu, "memory": $mem}}')
        done
        echo "$metrics_data"
    else
        echo "=== パフォーマンスメトリクス (過去1時間) ==="
        
        for instance_id in $instance_ids; do
            echo "インスタンスID: $instance_id"
            
            # CPU使用率
            local cpu_avg=$(aws cloudwatch get-metric-statistics \
                --namespace AWS/EC2 \
                --metric-name CPUUtilization \
                --dimensions Name=InstanceId,Value="$instance_id" \
                --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
                --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
                --period 300 \
                --statistics Average \
                --query 'Datapoints[0].Average' \
                --output text 2>/dev/null || echo "N/A")
            
            echo "  CPU使用率: ${cpu_avg}%"
            
            # ネットワーク使用量
            local network_in=$(aws cloudwatch get-metric-statistics \
                --namespace AWS/EC2 \
                --metric-name NetworkIn \
                --dimensions Name=InstanceId,Value="$instance_id" \
                --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
                --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
                --period 300 \
                --statistics Sum \
                --query 'Datapoints[0].Sum' \
                --output text 2>/dev/null || echo "N/A")
            
            local network_out=$(aws cloudwatch get-metric-statistics \
                --namespace AWS/EC2 \
                --metric-name NetworkOut \
                --dimensions Name=InstanceId,Value="$instance_id" \
                --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
                --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
                --period 300 \
                --statistics Sum \
                --query 'Datapoints[0].Sum' \
                --output text 2>/dev/null || echo "N/A")
            
            echo "  ネットワーク受信: ${network_in} bytes"
            echo "  ネットワーク送信: ${network_out} bytes"
            echo "---"
        done
    fi
}

# セキュリティ状態の確認
check_security() {
    if [[ "$SECURITY" == false && "$ALL_MONITORING" == false ]]; then
        return
    fi
    
    log_info "セキュリティ状態を確認しています..."
    
    # セキュリティグループの確認
    local security_groups=$(aws ec2 describe-security-groups \
        --filters "Name=group-name,Values=*$PROJECT_NAME*" \
        --query 'SecurityGroups[*].[GroupId,GroupName,Description,IpPermissions]' \
        --output json)
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        echo "$security_groups"
    else
        echo "=== セキュリティ状態 ==="
        
        local sg_count=$(echo "$security_groups" | jq -r 'length')
        if [[ "$sg_count" -eq 0 ]]; then
            log_warn "プロジェクト関連のセキュリティグループが見つかりません"
            return
        fi
        
        echo "$security_groups" | jq -r '.[] | "セキュリティグループID: \(.[0])" + "\n名前: \(.[1])" + "\n説明: \(.[2])" + "\n---"'
        
        # IAMロールの確認
        echo "=== IAMロール確認 ==="
        local iam_roles=$(aws iam list-roles --query 'Roles[?contains(RoleName, `gopier`) || contains(RoleName, `runner`)].RoleName' --output text 2>/dev/null || echo "")
        
        if [[ -n "$iam_roles" ]]; then
            echo "関連するIAMロール:"
            for role in $iam_roles; do
                echo "  - $role"
            done
        else
            log_warn "関連するIAMロールが見つかりません"
        fi
    fi
}

# リソース使用量の要約
generate_summary() {
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        return
    fi
    
    log_info "リソース使用量の要約を生成しています..."
    
    echo ""
    echo "=== 監視サマリー ==="
    echo "監視時刻: $(date)"
    
    # 実行中インスタンス数
    local running_instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*$PROJECT_NAME*" "Name=instance-state-name,Values=running" \
        --query 'Reservations[*].Instances[*].InstanceId' \
        --output text | wc -w)
    
    echo "実行中インスタンス数: $running_instances"
    
    # 停止中インスタンス数
    local stopped_instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*$PROJECT_NAME*" "Name=instance-state-name,Values=stopped" \
        --query 'Reservations[*].Instances[*].InstanceId' \
        --output text | wc -w)
    
    echo "停止中インスタンス数: $stopped_instances"
    
    # 今月の推定コスト
    local current_month=$(date +%Y-%m)
    local start_date="${current_month}-01"
    local end_date=$(date +%Y-%m-%d)
    
    local monthly_cost=$(aws ce get-cost-and-usage \
        --time-period Start="$start_date",End="$end_date" \
        --granularity MONTHLY \
        --metrics BlendedCost \
        --query 'ResultsByTime[0].Total.BlendedCost.Amount' \
        --output text 2>/dev/null || echo "0")
    
    echo "今月の推定コスト: $${monthly_cost}"
    
    # 推奨事項
    echo ""
    echo "=== 推奨事項 ==="
    
    if [[ "$running_instances" -gt 0 ]]; then
        log_warn "実行中のインスタンスがあります。使用後は停止してください。"
    fi
    
    if [[ "$stopped_instances" -gt 5 ]]; then
        log_warn "停止中のインスタンスが多数あります。不要なインスタンスを削除してください。"
    fi
    
    local cost_float=$(echo "$monthly_cost" | sed 's/,//g')
    if (( $(echo "$cost_float > 100" | bc -l) )); then
        log_warn "今月のコストが高くなっています。インスタンスタイプの見直しを検討してください。"
    fi
    
    log_success "監視完了"
}

# メイン実行
main() {
    echo "🔍 AWSランナー監視スクリプトを開始します..."
    echo ""
    
    check_prerequisites
    get_aws_info
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        # JSON形式での出力
        local json_output="{}"
        
        # EC2インスタンス情報
        local ec2_data=$(check_ec2_instances)
        json_output=$(echo "$json_output" | jq --argjson ec2 "$ec2_data" '. + {"ec2_instances": $ec2}')
        
        # ランナー使用状況
        local runner_data=$(check_runner_usage)
        json_output=$(echo "$json_output" | jq --argjson runner "$runner_data" '. + {"runner_usage": $runner}')
        
        # コスト分析
        if [[ "$COST_ANALYSIS" == true || "$ALL_MONITORING" == true ]]; then
            local cost_data=$(analyze_costs)
            json_output=$(echo "$json_output" | jq --argjson cost "$cost_data" '. + {"cost_analysis": $cost}')
        fi
        
        # パフォーマンスメトリクス
        if [[ "$PERFORMANCE" == true || "$ALL_MONITORING" == true ]]; then
            local perf_data=$(check_performance)
            json_output=$(echo "$json_output" | jq --argjson perf "$perf_data" '. + {"performance": $perf}')
        fi
        
        # セキュリティ状態
        if [[ "$SECURITY" == true || "$ALL_MONITORING" == true ]]; then
            local sec_data=$(check_security)
            json_output=$(echo "$json_output" | jq --argjson sec "$sec_data" '. + {"security": $sec}')
        fi
        
        echo "$json_output"
    else
        # テキスト形式での出力
        check_ec2_instances
        check_runner_usage
        
        if [[ "$COST_ANALYSIS" == true || "$ALL_MONITORING" == true ]]; then
            analyze_costs
        fi
        
        if [[ "$PERFORMANCE" == true || "$ALL_MONITORING" == true ]]; then
            check_performance
        fi
        
        if [[ "$SECURITY" == true || "$ALL_MONITORING" == true ]]; then
            check_security
        fi
        
        generate_summary
    fi
}

# スクリプト実行
main "$@" 