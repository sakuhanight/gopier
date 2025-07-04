#!/bin/bash

# AWS/EC2 全自動統合管理スクリプト
# 使用方法: ./scripts/aws-ec2-automation.sh [コマンド]
#
# コマンド:
#   auto-setup    - 完全自動設定（推奨）
#   start         - EC2ランナー起動
#   stop          - EC2ランナー停止
#   status        - ステータス確認
#   cleanup       - リソースクリーンアップ
#   monitor       - コスト監視設定
#   help          - ヘルプ表示

set -euo pipefail

# 色付き出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ログ関数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# 設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$PROJECT_ROOT/.aws-ec2-config.env"
PROJECT_NAME="gopier"

# デフォルト設定
DEFAULT_REGION="ap-northeast-1"
DEFAULT_INSTANCE_TYPE="c5.2xlarge"
DEFAULT_IAM_ROLE_NAME="gopier-ec2-role"
DEFAULT_SECURITY_GROUP_NAME="gopier-runner-sg"

# ヘルプ表示
show_help() {
    cat << EOF
AWS/EC2 全自動統合管理スクリプト

使用方法:
    $0 <command> [options]

コマンド:
    auto-setup    - 完全自動設定（推奨）
    start         - EC2ランナー起動
    stop          - EC2ランナー停止
    status        - ステータス確認
    cleanup       - リソースクリーンアップ
    monitor       - コスト監視設定
    help          - このヘルプを表示

オプション:
    --label       - ランナーラベル (デフォルト: gopier-runner-\$RANDOM)
    --type        - EC2インスタンスタイプ (デフォルト: c5.2xlarge)
    --region      - AWSリージョン (デフォルト: ap-northeast-1)
    --timeout     - タイムアウト時間（分）(デフォルト: 60)
    --help        - このヘルプを表示

自動設定機能:
    - AWS認証情報の自動検出
    - GitHub認証情報の自動検出
    - リポジトリ情報の自動検出
    - サブネット・セキュリティグループの自動選択・作成
    - IAMロール・ポリシーの自動作成
    - GitHub Secretsの自動設定
    - コスト監視の自動設定

例:
    $0 auto-setup                    # 完全自動設定
    $0 start --label my-runner       # ランナー起動
    $0 status                        # ステータス確認
    $0 cleanup                       # リソースクリーンアップ

注意:
    - AWS CLIとGitHub CLIが事前に設定されている必要があります
    - 初回実行時は auto-setup コマンドを使用してください
EOF
}

# 前提条件チェック
check_prerequisites() {
    log_step "前提条件をチェックしています..."
    
    # AWS CLI
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLIがインストールされていません"
        log_info "インストール方法:"
        log_info "  macOS: brew install awscli"
        log_info "  Linux: curl 'https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip' -o 'awscliv2.zip' && unzip awscliv2.zip && sudo ./aws/install"
        exit 1
    fi
    
    # GitHub CLI
    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLIがインストールされていません"
        log_info "インストール方法:"
        log_info "  macOS: brew install gh"
        log_info "  Linux: curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg"
        exit 1
    fi
    
    # jq
    if ! command -v jq &> /dev/null; then
        log_error "jqがインストールされていません"
        log_info "インストール方法:"
        log_info "  macOS: brew install jq"
        log_info "  Linux: sudo apt-get install jq"
        exit 1
    fi
    
    # AWS認証チェック
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS認証が設定されていません"
        log_info "以下のコマンドでAWS認証情報を設定してください:"
        log_info "  aws configure"
        log_info "  または"
        log_info "  aws sso login"
        exit 1
    fi
    
    # GitHub認証チェック
    if ! gh auth status &> /dev/null; then
        log_warning "GitHub CLIが認証されていません"
        log_info "GitHub CLIの認証を行います..."
        if ! gh auth login --web; then
            log_error "GitHub CLIの認証に失敗しました"
            exit 1
        fi
    fi
    
    log_success "前提条件チェック完了"
}

# 環境変数の自動設定
setup_environment() {
    log_step "環境変数を自動設定中..."
    
    # AWS認証情報をAWS CLIから取得
    if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]] || [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
        export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id)
        export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key)
        log_info "AWS認証情報をAWS CLIから取得しました"
    fi
    
    if [[ -z "${AWS_REGION:-}" ]]; then
        export AWS_REGION=$(aws configure get region || echo "$DEFAULT_REGION")
        log_info "AWS Region: $AWS_REGION"
    fi
    
    # GitHub認証情報の確認と設定
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        export GITHUB_TOKEN=$(gh auth token)
        if [[ -n "$GITHUB_TOKEN" ]]; then
            log_info "GitHub TokenをGitHub CLIから取得しました"
        else
            log_error "GitHub Tokenの取得に失敗しました"
            log_info "GitHub CLIの認証を確認してください: gh auth status"
            exit 1
        fi
    fi
    
    # GitHubリポジトリの自動検出
    if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
        # GitHub CLIから取得を試行
        local repo_info
        repo_info=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null || echo "")
        if [[ -n "$repo_info" ]]; then
            export GITHUB_REPOSITORY="$repo_info"
            log_info "GitHub Repository: $GITHUB_REPOSITORY"
        else
            # Gitリモートから取得を試行
            if command -v git &> /dev/null && git rev-parse --git-dir &> /dev/null; then
                local remote_url
                remote_url=$(git remote get-url origin 2>/dev/null || echo "")
                if [[ "$remote_url" =~ github\.com[:/]([^/]+/[^/]+?)(\.git)?$ ]]; then
                    export GITHUB_REPOSITORY="${BASH_REMATCH[1]}"
                    log_info "GitHub Repository: $GITHUB_REPOSITORY"
                fi
            fi
        fi
        
        if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
            log_error "GitHubリポジトリを自動検出できませんでした"
            log_info "環境変数 GITHUB_REPOSITORY を手動で設定してください"
            exit 1
        fi
    fi
    
    # IAMロール名のデフォルト値設定
    if [[ -z "${EC2_IAM_ROLE_NAME:-}" ]]; then
        export EC2_IAM_ROLE_NAME="$DEFAULT_IAM_ROLE_NAME"
        log_info "IAMロール名のデフォルト値を設定しました: $EC2_IAM_ROLE_NAME"
    fi
    
    log_success "環境変数設定完了"
}

# AWS情報の自動取得
get_aws_info() {
    log_step "AWS情報を自動取得中..."
    
    # アカウント情報
    local account_id=$(aws sts get-caller-identity --query Account --output text)
    local user_arn=$(aws sts get-caller-identity --query Arn --output text)
    
    log_info "AWS Account ID: $account_id"
    log_info "User ARN: $user_arn"
    
    # 最新のAmazon Linux 2 AMIを取得
    log_info "最新のAmazon Linux 2 AMIを取得中..."
    local ami_id=$(aws ec2 describe-images \
        --owners amazon \
        --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" "Name=state,Values=available" \
        --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
        --output text)
    
    if [[ -z "$ami_id" ]] || [[ "$ami_id" == "None" ]]; then
        log_error "AMI IDの取得に失敗しました"
        exit 1
    fi
    
    export EC2_IMAGE_ID="$ami_id"
    log_info "EC2 Image ID: $EC2_IMAGE_ID"
    
    log_success "AWS情報取得完了"
}

# サブネットの自動選択・作成
setup_subnet() {
    log_step "サブネットを自動設定中..."
    
    # 既存のサブネットを確認
    local subnets=$(aws ec2 describe-subnets --query 'Subnets[*].[SubnetId,VpcId,AvailabilityZone,DefaultForAz]' --output json)
    local subnet_count=$(echo "$subnets" | jq length)
    
    if [[ "$subnet_count" -eq 0 ]]; then
        log_error "サブネットが見つかりません"
        exit 1
    fi
    
    # デフォルトサブネットを優先して選択
    local selected_subnet=""
    local selected_vpc=""
    local selected_az=""
    
    # デフォルトサブネットを探す
    for i in $(seq 0 $((subnet_count - 1))); do
        local is_default=$(echo "$subnets" | jq -r ".[$i][3]")
        if [[ "$is_default" == "true" ]]; then
            selected_subnet=$(echo "$subnets" | jq -r ".[$i][0]")
            selected_vpc=$(echo "$subnets" | jq -r ".[$i][1]")
            selected_az=$(echo "$subnets" | jq -r ".[$i][2]")
            log_info "デフォルトサブネットを選択: $selected_subnet ($selected_az)"
            break
        fi
    done
    
    # デフォルトサブネットが見つからない場合は最初のサブネットを使用
    if [[ -z "$selected_subnet" ]]; then
        selected_subnet=$(echo "$subnets" | jq -r '.[0][0]')
        selected_vpc=$(echo "$subnets" | jq -r '.[0][1]')
        selected_az=$(echo "$subnets" | jq -r '.[0][2]')
        log_info "最初のサブネットを選択: $selected_subnet ($selected_az)"
    fi
    
    export EC2_SUBNET_ID="$selected_subnet"
    export EC2_VPC_ID="$selected_vpc"
    export EC2_AVAILABILITY_ZONE="$selected_az"
    
    log_success "サブネット設定完了: $selected_subnet"
}

# セキュリティグループの自動作成
setup_security_group() {
    log_step "セキュリティグループを自動設定中..."
    
    # 既存のセキュリティグループを確認
    local existing_sg=$(aws ec2 describe-security-groups \
        --filters "Name=group-name,Values=$DEFAULT_SECURITY_GROUP_NAME" \
        --query 'SecurityGroups[0].GroupId' \
        --output text 2>/dev/null || echo "")
    
    if [[ -n "$existing_sg" ]] && [[ "$existing_sg" != "None" ]]; then
        log_info "既存のセキュリティグループを使用: $existing_sg"
        export EC2_SECURITY_GROUP_ID="$existing_sg"
        return 0
    fi
    
    # 新しいセキュリティグループを作成
    log_info "新しいセキュリティグループを作成中..."
    local sg_id=$(aws ec2 create-security-group \
        --group-name "$DEFAULT_SECURITY_GROUP_NAME" \
        --description "Security group for $PROJECT_NAME EC2 runners" \
        --vpc-id "$EC2_VPC_ID" \
        --query 'GroupId' \
        --output text)
    
    # SSHアクセスを許可（自分のIPから）
    local my_ip=$(curl -s ifconfig.me)
    aws ec2 authorize-security-group-ingress \
        --group-id "$sg_id" \
        --protocol tcp \
        --port 22 \
        --cidr "$my_ip/32" \
        --description "SSH access from my IP" 2>/dev/null || true
    
    # HTTPSアクセスを許可
    aws ec2 authorize-security-group-ingress \
        --group-id "$sg_id" \
        --protocol tcp \
        --port 443 \
        --cidr 0.0.0.0/0 \
        --description "HTTPS access" 2>/dev/null || true
    
    # HTTPアクセスを許可
    aws ec2 authorize-security-group-ingress \
        --group-id "$sg_id" \
        --protocol tcp \
        --port 80 \
        --cidr 0.0.0.0/0 \
        --description "HTTP access" 2>/dev/null || true
    
    # アウトバウンドトラフィックを許可
    aws ec2 authorize-security-group-egress \
        --group-id "$sg_id" \
        --protocol -1 \
        --port -1 \
        --cidr 0.0.0.0/0 \
        --description "All outbound traffic" 2>/dev/null || true
    
    export EC2_SECURITY_GROUP_ID="$sg_id"
    log_success "セキュリティグループ作成完了: $sg_id"
}

# IAMロールの自動作成
setup_iam_role() {
    log_step "IAMロールを自動設定中..."
    
    # 既存のロールを確認
    if aws iam get-role --role-name "$DEFAULT_IAM_ROLE_NAME" &>/dev/null; then
        log_info "既存のIAMロールを使用: $DEFAULT_IAM_ROLE_NAME"
        
        # Instance Profileの存在確認
        if ! aws iam get-instance-profile --instance-profile-name "$DEFAULT_IAM_ROLE_NAME" &>/dev/null; then
            log_warning "IAM Instance Profileが存在しません。作成します..."
            aws iam create-instance-profile --instance-profile-name "$DEFAULT_IAM_ROLE_NAME"
            aws iam add-role-to-instance-profile \
                --instance-profile-name "$DEFAULT_IAM_ROLE_NAME" \
                --role-name "$DEFAULT_IAM_ROLE_NAME"
            log_success "IAM Instance Profileを作成しました: $DEFAULT_IAM_ROLE_NAME"
        else
            log_info "IAM Instance Profileが存在します: $DEFAULT_IAM_ROLE_NAME"
        fi
        
        export EC2_IAM_ROLE_NAME="$DEFAULT_IAM_ROLE_NAME"
        return 0
    fi
    
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
    aws iam create-role \
        --role-name "$DEFAULT_IAM_ROLE_NAME" \
        --assume-role-policy-document file:///tmp/ec2-trust-policy.json
    
    # EC2用ポリシーを作成
    cat > /tmp/ec2-policy.json << EOF
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
EOF
    
    # ポリシーを作成してアタッチ
    aws iam create-policy \
        --policy-name "${DEFAULT_IAM_ROLE_NAME}-policy" \
        --policy-document file:///tmp/ec2-policy.json
    
    local account_id=$(aws sts get-caller-identity --query Account --output text)
    aws iam attach-role-policy \
        --role-name "$DEFAULT_IAM_ROLE_NAME" \
        --policy-arn "arn:aws:iam::${account_id}:policy/${DEFAULT_IAM_ROLE_NAME}-policy"
    
    # インスタンスプロファイルを作成
    aws iam create-instance-profile --instance-profile-name "$DEFAULT_IAM_ROLE_NAME"
    aws iam add-role-to-instance-profile \
        --instance-profile-name "$DEFAULT_IAM_ROLE_NAME" \
        --role-name "$DEFAULT_IAM_ROLE_NAME"
    
    # 一時ファイルを削除
    rm -f /tmp/ec2-trust-policy.json /tmp/ec2-policy.json
    
    export EC2_IAM_ROLE_NAME="$DEFAULT_IAM_ROLE_NAME"
    log_success "IAMロール作成完了: $DEFAULT_IAM_ROLE_NAME"
}

# GitHub Secretsの自動設定
setup_github_secrets() {
    log_step "GitHub Secretsを自動設定中..."
    
    # 現在のリポジトリを取得
    local repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
    log_info "対象リポジトリ: $repo"
    
    # 空文字チェックとデフォルト値設定
    local aws_region="${AWS_REGION:-$DEFAULT_REGION}"
    local instance_type="${EC2_INSTANCE_TYPE:-$DEFAULT_INSTANCE_TYPE}"
    local iam_role_name="${EC2_IAM_ROLE_NAME:-$DEFAULT_IAM_ROLE_NAME}"
    
    # Secrets設定（空文字を防ぐ）
    local secrets=(
        "AWS_ACCESS_KEY_ID:$AWS_ACCESS_KEY_ID"
        "AWS_SECRET_ACCESS_KEY:$AWS_SECRET_ACCESS_KEY"
        "AWS_REGION:$aws_region"
        "EC2_SECURITY_GROUP_ID:$EC2_SECURITY_GROUP_ID"
        "EC2_SUBNET_ID:$EC2_SUBNET_ID"
        "EC2_IMAGE_ID:$EC2_IMAGE_ID"
        "EC2_INSTANCE_TYPE:$instance_type"
        "EC2_IAM_ROLE_NAME:$iam_role_name"
        "GITHUB_TOKEN:$GITHUB_TOKEN"
    )
    
    for secret in "${secrets[@]}"; do
        local key="${secret%%:*}"
        local value="${secret#*:}"
        
        if [[ -n "$value" ]]; then
            log_info "$key を設定中..."
            echo "$value" | gh secret set "$key" --repo "$repo" 2>/dev/null || {
                log_warning "$key の設定に失敗しました（既に存在する可能性があります）"
            }
        else
            log_warning "$key の値が空のため、設定をスキップします"
        fi
    done
    
    log_success "GitHub Secrets設定完了"
}

# 設定ファイルの保存
save_config() {
    log_step "設定を保存中..."
    
    # 空文字チェックとデフォルト値設定
    local aws_region="${AWS_REGION:-$DEFAULT_REGION}"
    local instance_type="${EC2_INSTANCE_TYPE:-$DEFAULT_INSTANCE_TYPE}"
    local iam_role_name="${EC2_IAM_ROLE_NAME:-$DEFAULT_IAM_ROLE_NAME}"
    local project_name="${PROJECT_NAME:-gopier}"
    
    # 必須項目の検証
    if [[ -z "$aws_region" ]]; then
        log_error "AWS_REGIONが設定されていません"
        exit 1
    fi
    
    if [[ -z "$instance_type" ]]; then
        log_error "EC2_INSTANCE_TYPEが設定されていません"
        exit 1
    fi
    
    if [[ -z "$iam_role_name" ]]; then
        log_error "EC2_IAM_ROLE_NAMEが設定されていません"
        exit 1
    fi
    
    if [[ -z "$GITHUB_REPOSITORY" ]]; then
        log_error "GITHUB_REPOSITORYが設定されていません"
        exit 1
    fi
    
    # 設定ファイルに保存
    cat > "$CONFIG_FILE" << EOF
# AWS/EC2自動設定ファイル
# 生成日時: $(date)

# AWS設定
AWS_REGION=$aws_region
EC2_INSTANCE_TYPE=$instance_type

# EC2設定
EC2_IMAGE_ID=$EC2_IMAGE_ID
EC2_SUBNET_ID=$EC2_SUBNET_ID
EC2_VPC_ID=$EC2_VPC_ID
EC2_AVAILABILITY_ZONE=$EC2_AVAILABILITY_ZONE
EC2_SECURITY_GROUP_ID=$EC2_SECURITY_GROUP_ID
EC2_IAM_ROLE_NAME=$iam_role_name

# GitHub設定
GITHUB_REPOSITORY=$GITHUB_REPOSITORY
GITHUB_TOKEN=$GITHUB_TOKEN

# プロジェクト設定
PROJECT_NAME=$project_name
EOF
    
    log_success "設定ファイル保存完了: $CONFIG_FILE"
    log_info "保存された設定:"
    log_info "  AWS_REGION: $aws_region"
    log_info "  EC2_INSTANCE_TYPE: $instance_type"
    log_info "  EC2_IAM_ROLE_NAME: $iam_role_name"
    log_info "  GITHUB_REPOSITORY: $GITHUB_REPOSITORY"
}

# 設定ファイルの読み込み
load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        log_info "設定ファイルを読み込み中: $CONFIG_FILE"
        source "$CONFIG_FILE"
    else
        log_info "設定ファイルが見つかりません: $CONFIG_FILE"
        log_info "環境変数を使用します"
        
        # 必要な環境変数の確認
        local required_vars=(
            "AWS_REGION"
            "EC2_INSTANCE_TYPE"
            "EC2_IMAGE_ID"
            "EC2_SUBNET_ID"
            "EC2_SECURITY_GROUP_ID"
            "EC2_IAM_ROLE_NAME"
            "GITHUB_REPOSITORY"
            "GITHUB_TOKEN"
        )
        
        local missing_vars=()
        for var in "${required_vars[@]}"; do
            if [[ -z "${!var}" ]]; then
                missing_vars+=("$var")
            fi
        done
        
        if [[ ${#missing_vars[@]} -gt 0 ]]; then
            log_error "以下の環境変数が設定されていません:"
            printf '  - %s\n' "${missing_vars[@]}"
            log_info "GitHub Actions Secretsまたは環境変数を設定してください"
            exit 1
        fi
        
        log_success "環境変数による設定読み込み完了"
    fi
}

# 完全自動設定
auto_setup() {
    log_step "完全自動設定を開始します..."
    
    check_prerequisites
    setup_environment
    get_aws_info
    setup_subnet
    setup_security_group
    setup_iam_role
    setup_github_secrets
    
    # IAMロール名の自動設定
    if [[ -z "$EC2_IAM_ROLE_NAME" ]]; then
        export EC2_IAM_ROLE_NAME="$DEFAULT_IAM_ROLE_NAME"
        log_info "IAMロール名を自動設定しました: $EC2_IAM_ROLE_NAME"
    fi
    
    # 設定ファイルに保存
    save_config
    
    log_success "完全自動設定が完了しました！"
    echo
    log_info "次のステップ:"
    log_info "1. GitHub ActionsワークフローでEC2ランナーを使用できます"
    log_info "2. ランナー起動: $0 start"
    log_info "3. ステータス確認: $0 status"
    echo
    log_info "設定情報:"
    log_info "  インスタンスタイプ: $EC2_INSTANCE_TYPE"
    log_info "  リージョン: $AWS_REGION"
    log_info "  サブネット: $EC2_SUBNET_ID"
    log_info "  セキュリティグループ: $EC2_SECURITY_GROUP_ID"
    log_info "  IAMロール: $EC2_IAM_ROLE_NAME"
}

# EC2ランナー起動
start_runner() {
    log_step "EC2ランナーを起動中..."
    
    load_config
    
    # IAMロール名のデフォルト値チェック
    if [[ -z "${EC2_IAM_ROLE_NAME:-}" ]]; then
        export EC2_IAM_ROLE_NAME="$DEFAULT_IAM_ROLE_NAME"
        log_info "IAMロール名のデフォルト値を設定しました: $EC2_IAM_ROLE_NAME"
    fi
    
    # GitHubトークンのデフォルト値チェック
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        export GITHUB_TOKEN=$(gh auth token)
        if [[ -n "$GITHUB_TOKEN" ]]; then
            log_info "GitHub TokenをGitHub CLIから取得しました"
        else
            log_error "GitHub Tokenが設定されていません"
            log_info "GitHub CLIの認証を確認してください: gh auth status"
            exit 1
        fi
    fi
    
    # AMI IDの検証
    if [[ -n "$EC2_IMAGE_ID" ]]; then
        log_info "AMI IDを検証中: $EC2_IMAGE_ID"
        if ! aws ec2 describe-images --image-ids "$EC2_IMAGE_ID" --query 'Images[0].ImageId' --output text &>/dev/null; then
            log_warning "指定されたAMI IDが無効です: $EC2_IMAGE_ID"
            log_info "最新のAmazon Linux 2 AMIを自動取得します..."
            
            local new_ami_id=$(aws ec2 describe-images \
                --owners amazon \
                --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" "Name=state,Values=available" \
                --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
                --output text)
            
            if [[ -n "$new_ami_id" ]] && [[ "$new_ami_id" != "None" ]]; then
                export EC2_IMAGE_ID="$new_ami_id"
                log_success "新しいAMI IDを設定しました: $EC2_IMAGE_ID"
            else
                log_error "AMI IDの自動取得に失敗しました"
                exit 1
            fi
        else
            log_success "AMI IDが有効です: $EC2_IMAGE_ID"
        fi
    else
        log_error "EC2_IMAGE_IDが設定されていません"
        exit 1
    fi
    
    # ラベル生成
    local label="${LABEL:-gopier-runner-$(date +%s)}"
    
    # IAMロールとInstance Profileの検証
    log_info "IAMロールとInstance Profileを検証中: $EC2_IAM_ROLE_NAME"
    
    # 1. IAMロールの存在確認
    if ! aws iam get-role --role-name "$EC2_IAM_ROLE_NAME" &>/dev/null; then
        log_warning "IAMロールが存在しません: $EC2_IAM_ROLE_NAME"
        log_info "IAMロールとInstance Profileを作成します..."
        
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
        
        # IAMロールを作成
        aws iam create-role \
            --role-name "$EC2_IAM_ROLE_NAME" \
            --assume-role-policy-document file:///tmp/ec2-trust-policy.json
        
        # EC2用ポリシーを作成
        cat > /tmp/ec2-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeTags",
                "ec2:DescribeInstances",
                "ec2:DescribeRegions",
                "ec2:DescribeAvailabilityZones",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeVpcs",
                "ec2:DescribeImages",
                "ec2:DescribeInstanceStatus",
                "ec2:DescribeInstanceAttribute",
                "ec2:DescribeInstanceCreditSpecifications",
                "ec2:DescribeInstanceTypes",
                "ec2:DescribeKeyPairs",
                "ec2:DescribeNetworkInterfaces",
                "ec2:DescribeNetworkAcls",
                "ec2:DescribeRouteTables",
                "ec2:DescribeInternetGateways",
                "ec2:DescribeNatGateways",
                "ec2:DescribeVpcEndpoints",
                "ec2:DescribeVpcPeeringConnections",
                "ec2:DescribeTransitGateways",
                "ec2:DescribeTransitGatewayVpcAttachments",
                "ec2:DescribeTransitGatewayRouteTables",
                "ec2:DescribeTransitGatewayAttachments",
                "ec2:DescribeTransitGatewayMulticastDomains",
                "ec2:DescribeTransitGatewayPeeringAttachments",
                "ec2:DescribeTransitGatewayConnects",
                "ec2:DescribeTransitGatewayConnectPeers",
                "ec2:DescribeTransitGatewayPolicyTables",
                "ec2:DescribeTransitGatewayRouteTableAnnouncements",
                "ec2:DescribeTransitGatewayVpcAttachments",
                "ec2:DescribeTransitGatewayRouteTables",
                "ec2:DescribeTransitGatewayAttachments",
                "ec2:DescribeTransitGatewayMulticastDomains",
                "ec2:DescribeTransitGatewayPeeringAttachments",
                "ec2:DescribeTransitGatewayConnects",
                "ec2:DescribeTransitGatewayConnectPeers",
                "ec2:DescribeTransitGatewayPolicyTables",
                "ec2:DescribeTransitGatewayRouteTableAnnouncements"
            ],
            "Resource": "*"
        }
    ]
}
EOF
        
        # ポリシーを作成
        aws iam create-policy \
            --policy-name "${EC2_IAM_ROLE_NAME}-policy" \
            --policy-document file:///tmp/ec2-policy.json
        
        # ロールにポリシーをアタッチ
        aws iam attach-role-policy \
            --role-name "$EC2_IAM_ROLE_NAME" \
            --policy-arn "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):policy/${EC2_IAM_ROLE_NAME}-policy"
        
        log_success "IAMロールを作成しました: $EC2_IAM_ROLE_NAME"
    else
        log_success "IAMロールが存在します: $EC2_IAM_ROLE_NAME"
    fi
    
    # 2. Instance Profileの存在確認と作成
    if ! aws iam get-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME" &>/dev/null; then
        log_warning "IAM Instance Profileが存在しません: $EC2_IAM_ROLE_NAME"
        log_info "IAM Instance Profileを作成します..."
        
        # Instance Profileを作成
        aws iam create-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || {
            log_warning "Instance Profileの作成に失敗しました（既に存在する可能性があります）"
        }
        
        # ロールをInstance Profileに追加
        aws iam add-role-to-instance-profile \
            --instance-profile-name "$EC2_IAM_ROLE_NAME" \
            --role-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || {
            log_warning "ロールの追加に失敗しました（既に追加されている可能性があります）"
        }
        
        # 作成完了を待機
        sleep 10
        log_success "IAM Instance Profileを作成しました: $EC2_IAM_ROLE_NAME"
    else
        log_success "IAM Instance Profileが存在します: $EC2_IAM_ROLE_NAME"
    fi
    
    # 3. 最終検証
    log_info "IAMロールとInstance Profileの最終検証中..."
    if aws iam get-role --role-name "$EC2_IAM_ROLE_NAME" &>/dev/null; then
        log_success "✓ IAMロール検証完了: $EC2_IAM_ROLE_NAME"
    else
        log_error "✗ IAMロール検証失敗: $EC2_IAM_ROLE_NAME"
        exit 1
    fi
    
    if aws iam get-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME" &>/dev/null; then
        log_success "✓ IAM Instance Profile検証完了: $EC2_IAM_ROLE_NAME"
    else
        log_error "✗ IAM Instance Profile検証失敗: $EC2_IAM_ROLE_NAME"
        exit 1
    fi
    
    # ユーザーデータスクリプトの作成
    local user_data_script
    user_data_script=$(create_user_data_script "$label")
    
    # EC2インスタンスの作成
    local instance_id
    instance_id=$(aws ec2 run-instances \
        --image-id "$EC2_IMAGE_ID" \
        --instance-type "$EC2_INSTANCE_TYPE" \
        --subnet-id "$EC2_SUBNET_ID" \
        --security-group-ids "$EC2_SECURITY_GROUP_ID" \
        --iam-instance-profile Name="$EC2_IAM_ROLE_NAME" \
        --user-data "$user_data_script" \
        --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=GitHub-Runner-$label},{Key=GitHubRepository,Value=$GITHUB_REPOSITORY},{Key=RunnerLabel,Value=$label}]" \
        --query 'Instances[0].InstanceId' \
        --output text)
    
    log_info "インスタンス作成完了: $instance_id"
    
    # インスタンスの起動完了を待機
    log_info "インスタンスの起動を待機中..."
    aws ec2 wait instance-running --instance-ids "$instance_id"
    
    # パブリックIPアドレスの取得
    local public_ip
    public_ip=$(aws ec2 describe-instances \
        --instance-ids "$instance_id" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text)
    
    log_success "EC2ランナー起動完了: $instance_id (IP: $public_ip)"
    log_info "ラベル: $label"
}

# ユーザーデータスクリプトの作成
create_user_data_script() {
    local label="$1"
    
    # GitHubリポジトリとトークンを取得
    local repo="${GITHUB_REPOSITORY:-}"
    local token="${GITHUB_TOKEN:-}"
    
    if [[ -z "$repo" ]]; then
        log_error "GITHUB_REPOSITORYが設定されていません"
        exit 1
    fi
    
    if [[ -z "$token" ]]; then
        log_error "GITHUB_TOKENが設定されていません"
        exit 1
    fi
    
    # ランナートークンを取得（事前に取得）
    local runner_token
    runner_token=$(gh api repos/$repo/actions/runners/registration-token --jq .token 2>/dev/null || echo "")
    
    if [[ -z "$runner_token" ]]; then
        log_warning "ランナートークンの取得に失敗しました。GITHUB_TOKENを使用します"
        runner_token="$token"
    fi
    
    cat << EOF
#!/bin/bash
set -e

# ログ設定
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1
echo "Starting user data script at \$(date)"

# システムアップデート
echo "Updating system packages..."
yum update -y

# 必要なパッケージのインストール
echo "Installing required packages..."
yum install -y git curl wget unzip jq

# Goのインストール（最新版のみ）
echo "Installing Go..."
cd /tmp
wget -q https://go.dev/dl/go1.22.linux-amd64.tar.gz
if [ -f go1.22.linux-amd64.tar.gz ]; then
    tar -C /usr/local -xzf go1.22.linux-amd64.tar.gz
    rm -f go1.22.linux-amd64.tar.gz
    echo "Go installed successfully"
else
    echo "Failed to download Go"
    exit 1
fi

# 環境変数の設定
echo 'export PATH=\$PATH:/usr/local/go/bin' >> /home/ec2-user/.bashrc
echo 'export GOGC=100' >> /home/ec2-user/.bashrc
echo 'export GOMAXPROCS=8' >> /home/ec2-user/.bashrc
source /home/ec2-user/.bashrc

# Dockerのインストール
echo "Installing Docker..."
yum install -y docker
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user

# Dockerの起動確認
if systemctl is-active --quiet docker; then
    echo "Docker started successfully"
else
    echo "Failed to start Docker"
    systemctl status docker
    exit 1
fi

# Windows用のツールチェーン（クロスコンパイル用）
echo "Installing Windows toolchain..."
yum install -y mingw64-gcc || {
    echo "Failed to install Windows toolchain, continuing..."
}

# GitHub Actionsランナーのダウンロード
echo "Downloading GitHub Actions runner..."
mkdir -p /opt/actions-runner
cd /opt/actions-runner

# 最新バージョンのランナーを取得
RUNNER_VERSION=\$(curl -s https://api.github.com/repos/actions/runner/releases/latest | jq -r .tag_name)
echo "Using runner version: \$RUNNER_VERSION"

if [ -n "\$RUNNER_VERSION" ] && [ "\$RUNNER_VERSION" != "null" ]; then
    curl -o actions-runner-linux-x64.tar.gz -L https://github.com/actions/runner/releases/download/\$RUNNER_VERSION/actions-runner-linux-x64-\${RUNNER_VERSION#v}.tar.gz
    if [ -f actions-runner-linux-x64.tar.gz ]; then
        tar xzf ./actions-runner-linux-x64.tar.gz
        rm -f actions-runner-linux-x64.tar.gz
        echo "Runner downloaded successfully"
    else
        echo "Failed to download runner"
        exit 1
    fi
else
    echo "Failed to get runner version"
    exit 1
fi

# ランナーの設定
echo "Configuring runner..."
if ./config.sh --url https://github.com/$repo --token $runner_token --labels $label --unattended --replace; then
    echo "Runner configured successfully"
else
    echo "Failed to configure runner"
    echo "Checking config logs..."
    cat _diag/*.log || true
    exit 1
fi

# ランナーをサービスとしてインストール
echo "Installing runner as a service..."
if ./svc.sh install ec2-user; then
    echo "Runner service installed successfully"
else
    echo "Failed to install runner service"
    exit 1
fi

# ランナーサービスの起動
echo "Starting runner service..."
if ./svc.sh start; then
    echo "Runner service started successfully"
else
    echo "Failed to start runner service"
    echo "Checking service status..."
    systemctl status actions.runner.* || true
    exit 1
fi

# 起動確認
echo "Waiting for runner to start..."
sleep 30

# ランナーの状態確認
if systemctl is-active --quiet actions.runner.*; then
    echo "Runner service is running"
else
    echo "Runner service is not running"
    echo "Checking service status..."
    systemctl status actions.runner.* || true
    echo "Checking runner logs..."
    tail -20 /var/log/actions-runner.log || true
    exit 1
fi

# ランナーの登録確認
echo "Checking runner registration..."
sleep 10
if [ -f .runner ]; then
    echo "Runner configuration file exists"
    cat .runner
else
    echo "Runner configuration file not found"
    echo "Checking runner directory contents..."
    ls -la /opt/actions-runner/ || true
    exit 1
fi

echo "User data script completed at \$(date)"
echo "Runner setup completed successfully"
EOF
}

# EC2ランナー停止
stop_runner() {
    log_step "EC2ランナーを停止中..."
    
    load_config
    
    # 実行中のインスタンスを検索
    local instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*GitHub-Runner*" "Name=instance-state-name,Values=running" \
        --query 'Reservations[*].Instances[*].InstanceId' \
        --output text)
    
    if [[ -z "$instances" ]]; then
        log_warning "実行中のEC2ランナーが見つかりません"
        return 0
    fi
    
    for instance_id in $instances; do
        log_info "インスタンスを停止中: $instance_id"
        aws ec2 terminate-instances --instance-ids "$instance_id"
    done
    
    log_success "EC2ランナー停止完了"
}

# ステータス確認
check_status() {
    log_step "ステータスを確認中..."
    
    load_config
    
    # EC2インスタンスの状態
    local instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*GitHub-Runner*" \
        --query 'Reservations[*].Instances[*].[InstanceId,State.Name,InstanceType,LaunchTime,Tags[?Key==`RunnerLabel`].Value|[0]]' \
        --output table)
    
    if [[ -z "$instances" ]] || [[ "$instances" == "None" ]]; then
        log_info "EC2ランナーインスタンスが見つかりません"
    else
        echo "$instances"
    fi
    
    # GitHubランナーの状態
    log_info "GitHubランナーの状態を確認中..."
    gh api repos/$GITHUB_REPOSITORY/actions/runners --jq '.runners[] | "\(.name): \(.status) (\(.busy))"' 2>/dev/null || {
        log_warning "GitHubランナー情報の取得に失敗しました"
    }
}

# リソースクリーンアップ
cleanup_resources() {
    log_step "リソースクリーンアップを開始..."
    
    load_config
    
    # 実行中のインスタンスを停止
    local instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Name,Values=*GitHub-Runner*" "Name=instance-state-name,Values=running" \
        --query 'Reservations[*].Instances[*].InstanceId' \
        --output text)
    
    if [[ -n "$instances" ]]; then
        log_info "実行中のインスタンスを停止中..."
        aws ec2 terminate-instances --instance-ids $instances
        aws ec2 wait instance-terminated --instance-ids $instances
    fi
    
    # セキュリティグループの削除
    if [[ -n "$EC2_SECURITY_GROUP_ID" ]]; then
        log_info "セキュリティグループを削除中: $EC2_SECURITY_GROUP_ID"
        aws ec2 delete-security-group --group-id "$EC2_SECURITY_GROUP_ID" 2>/dev/null || {
            log_warning "セキュリティグループの削除に失敗しました"
        }
    fi
    
    # IAMロールの削除
    if [[ -n "$EC2_IAM_ROLE_NAME" ]]; then
        log_info "IAMロールを削除中: $EC2_IAM_ROLE_NAME"
        
        # インスタンスプロファイルからロールを削除
        aws iam remove-role-from-instance-profile \
            --instance-profile-name "$EC2_IAM_ROLE_NAME" \
            --role-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || true
        
        # インスタンスプロファイルを削除
        aws iam delete-instance-profile \
            --instance-profile-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || true
        
        # ポリシーをデタッチ
        local account_id=$(aws sts get-caller-identity --query Account --output text)
        aws iam detach-role-policy \
            --role-name "$EC2_IAM_ROLE_NAME" \
            --policy-arn "arn:aws:iam::${account_id}:policy/${EC2_IAM_ROLE_NAME}-policy" 2>/dev/null || true
        
        # ポリシーを削除
        aws iam delete-policy \
            --policy-arn "arn:aws:iam::${account_id}:policy/${EC2_IAM_ROLE_NAME}-policy" 2>/dev/null || true
        
        # ロールを削除
        aws iam delete-role --role-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || {
            log_warning "IAMロールの削除に失敗しました"
        }
    fi
    
    # 設定ファイルの削除
    if [[ -f "$CONFIG_FILE" ]]; then
        log_info "設定ファイルを削除中: $CONFIG_FILE"
        rm -f "$CONFIG_FILE"
    fi
    
    log_success "リソースクリーンアップ完了"
}

# コスト監視設定
setup_monitoring() {
    log_step "コスト監視を設定中..."
    
    load_config
    
    # CloudWatchダッシュボードの作成
    local dashboard_name="${PROJECT_NAME}-cost-dashboard"
    
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
                "title": "$PROJECT_NAME - CPU Utilization",
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
                "title": "$PROJECT_NAME - Network In",
                "period": 300
            }
        }
    ]
}
EOF
    
    aws cloudwatch put-dashboard \
        --dashboard-name "$dashboard_name" \
        --dashboard-body file:///tmp/dashboard.json 2>/dev/null || {
        log_warning "CloudWatchダッシュボードの作成に失敗しました"
    }
    
    rm -f /tmp/dashboard.json
    
    log_success "コスト監視設定完了"
    log_info "CloudWatchダッシュボード: $dashboard_name"
}

# パラメータの解析
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --label)
                LABEL="$2"
                shift 2
                ;;
            --type)
                EC2_INSTANCE_TYPE="$2"
                shift 2
                ;;
            --region)
                AWS_REGION="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                COMMAND="$1"
                shift
                ;;
        esac
    done
}

# メイン処理
main() {
    # デフォルト値の設定
    EC2_INSTANCE_TYPE="${EC2_INSTANCE_TYPE:-$DEFAULT_INSTANCE_TYPE}"
    AWS_REGION="${AWS_REGION:-$DEFAULT_REGION}"
    TIMEOUT="${TIMEOUT:-60}"
    
    case "${COMMAND:-}" in
        auto-setup)
            auto_setup
            ;;
        start)
            start_runner
            ;;
        stop)
            stop_runner
            ;;
        status)
            check_status
            ;;
        cleanup)
            cleanup_resources
            ;;
        monitor)
            setup_monitoring
            ;;
        help|"")
            show_help
            ;;
        *)
            log_error "不明なコマンド: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

# スクリプト実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi 