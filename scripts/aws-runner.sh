#!/bin/bash

# AWSランナー統合管理スクリプト
# 使用方法: ./scripts/aws-runner.sh [コマンド]
#
# コマンド:
#   setup      - 完全自動設定（推奨）
#   info       - AWSリソース情報表示
#   config     - 設定ファイル管理
#   deploy     - AWSランナー設定
#   cleanup    - リソースクリーンアップ
#   help       - ヘルプ表示

set -e

# 色付き出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 設定ファイル
CONFIG_FILE="aws-runner-config.env"

# ヘルプ表示
show_help() {
    echo "AWSランナー統合管理スクリプト"
    echo
    echo "使用方法: $0 [コマンド]"
    echo
    echo "コマンド:"
    echo "  setup      - 完全自動設定（推奨）"
    echo "  info       - AWSリソース情報表示"
    echo "  config     - 設定ファイル管理"
    echo "  deploy     - AWSランナー設定"
    echo "  cleanup    - リソースクリーンアップ"
    echo "  help       - このヘルプを表示"
    echo
    echo "例:"
    echo "  $0 setup    # 完全自動設定"
    echo "  $0 info     # AWSリソース情報を表示"
    echo "  $0 deploy   # AWSランナーを設定"
}

# AWS CLIの確認
check_aws_cli() {
    log_info "AWS CLIの確認中..."
    
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLIがインストールされていません"
        log_info "インストール方法:"
        log_info "  macOS: brew install awscli"
        log_info "  Linux: curl 'https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip' -o 'awscliv2.zip' && unzip awscliv2.zip && sudo ./aws/install"
        exit 1
    fi
    
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS認証情報が正しく設定されていません"
        log_info "以下のコマンドでAWS認証情報を設定してください:"
        log_info "  aws configure"
        exit 1
    fi
    
    log_success "AWS CLI確認完了"
}

# AWS情報の自動取得
get_aws_info() {
    log_info "AWS情報を自動取得中..."
    
    # 現在のリージョンを取得
    local current_region=$(aws configure get region)
    if [ -z "$current_region" ]; then
        current_region="us-east-1"
    fi
    
    # アカウント情報を取得
    local account_id=$(aws sts get-caller-identity --query Account --output text)
    local user_arn=$(aws sts get-caller-identity --query Arn --output text)
    
    log_info "現在のリージョン: $current_region"
    log_info "AWSアカウントID: $account_id"
    log_info "ユーザーARN: $user_arn"
    
    # サブネット情報を取得
    log_info "サブネット情報を取得中..."
    local subnets=$(aws ec2 describe-subnets --query 'Subnets[*].[SubnetId,VpcId,AvailabilityZone,CidrBlock]' --output json 2>/dev/null || echo "[]")
    
    if [ "$subnets" = "[]" ]; then
        log_error "サブネットが見つかりません"
        exit 1
    fi
    
    # 最初のサブネットを選択
    local subnet_id=$(echo "$subnets" | jq -r '.[0][0]')
    local vpc_id=$(echo "$subnets" | jq -r '.[0][1]')
    local az=$(echo "$subnets" | jq -r '.[0][2]')
    
    log_info "選択されたサブネット: $subnet_id ($az)"
    
    # AMI情報を取得
    log_info "AMI情報を取得中..."
    local ami_id=$(aws ec2 describe-images \
        --owners amazon \
        --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" "Name=state,Values=available" \
        --query 'Images[0].ImageId' \
        --output text 2>/dev/null || echo "")
    
    if [ -z "$ami_id" ] || [ "$ami_id" = "None" ]; then
        log_warning "最新のAmazon Linux 2 AMIが見つかりません。デフォルト値を使用します。"
        ami_id="ami-0c02fb55956c7d316"
    fi
    
    log_info "選択されたAMI: $ami_id"
    
    # 環境変数に設定
    export AWS_REGION="$current_region"
    export EC2_SUBNET_ID="$subnet_id"
    export EC2_IMAGE_ID="$ami_id"
    export EC2_INSTANCE_TYPE="c5.4xlarge"
    
    log_success "AWS情報の取得が完了しました"
}

# ユーザー入力の取得
get_user_input() {
    log_info "ユーザー入力を受け付けます..."
    
    # AWS認証情報の確認
    echo
    log_info "AWS認証情報は既に設定されています。"
    log_info "現在の設定:"
    aws sts get-caller-identity --output table
    
    # GitHub Personal Access Tokenの入力
    echo
    log_info "GitHub Personal Access Tokenを入力してください:"
    log_info "GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)"
    log_info "必要な権限: repo, workflow"
    echo -n "GitHub Personal Access Token: "
    read -s gh_token
    echo
    
    if [ -z "$gh_token" ]; then
        log_error "GitHub Personal Access Tokenが入力されていません"
        exit 1
    fi
    
    # インスタンスタイプの選択
    echo
    log_info "EC2インスタンスタイプを選択してください:"
    echo "1. c5.xlarge (4 vCPU, 8 GB RAM) - 軽量テスト用"
    echo "2. c5.2xlarge (8 vCPU, 16 GB RAM) - 標準テスト用"
    echo "3. c5.4xlarge (16 vCPU, 32 GB RAM) - 高性能テスト用 [推奨]"
    echo "4. c5.9xlarge (36 vCPU, 72 GB RAM) - 大規模テスト用"
    echo -n "選択 (1-4) [3]: "
    read instance_choice
    
    case $instance_choice in
        1) export EC2_INSTANCE_TYPE="c5.xlarge" ;;
        2) export EC2_INSTANCE_TYPE="c5.2xlarge" ;;
        3|"") export EC2_INSTANCE_TYPE="c5.4xlarge" ;;
        4) export EC2_INSTANCE_TYPE="c5.9xlarge" ;;
        *) 
            log_warning "無効な選択です。デフォルト値を使用します。"
            export EC2_INSTANCE_TYPE="c5.4xlarge"
            ;;
    esac
    
    log_info "選択されたインスタンスタイプ: $EC2_INSTANCE_TYPE"
    
    # GitHub Personal Access Tokenを設定
    export GH_PERSONAL_ACCESS_TOKEN="$gh_token"
    
    log_success "ユーザー入力の取得が完了しました"
}

# 設定ファイルの生成
generate_config() {
    log_info "設定ファイルを生成中: $CONFIG_FILE"
    
    # AWS認証情報を取得
    local access_key_id=$(aws configure get aws_access_key_id)
    local secret_access_key=$(aws configure get aws_secret_access_key)
    
    cat > "$CONFIG_FILE" << EOF
# AWSランナー設定ファイル
# このファイルは自動生成されます

# AWS認証情報
AWS_ACCESS_KEY_ID=${access_key_id}
AWS_SECRET_ACCESS_KEY=${secret_access_key}
AWS_REGION=${AWS_REGION}

# EC2設定
EC2_INSTANCE_TYPE=${EC2_INSTANCE_TYPE}
EC2_IMAGE_ID=${EC2_IMAGE_ID}
EC2_SUBNET_ID=${EC2_SUBNET_ID}
EC2_SECURITY_GROUP_ID=
EC2_IAM_ROLE_NAME=

# GitHub設定
GH_PERSONAL_ACCESS_TOKEN=${GH_PERSONAL_ACCESS_TOKEN}

# その他の設定
CODECOV_TOKEN=
EOF
    
    log_success "設定ファイルを生成しました: $CONFIG_FILE"
}

# 設定ファイルの読み込み
load_config() {
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "設定ファイルが見つかりません: $CONFIG_FILE"
        log_info "setup コマンドを実行して設定ファイルを作成してください"
        return 1
    fi
    
    log_info "設定ファイルを読み込み中: $CONFIG_FILE"
    
    # 設定ファイルから環境変数を読み込み
    while IFS='=' read -r key value; do
        # コメント行と空行をスキップ
        if [[ $key =~ ^[[:space:]]*# ]] || [[ -z $key ]]; then
            continue
        fi
        
        # 前後の空白を削除
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)
        
        # プレースホルダーをチェック
        if [[ "$value" == "your_access_key_here" ]] || [[ "$value" == "your_secret_key_here" ]] || [[ "$value" == "your_github_token_here" ]]; then
            log_warning "プレースホルダーが設定されています: $key"
            continue
        fi
        
        # 環境変数をエクスポート
        export "$key=$value"
        log_info "環境変数を設定: $key"
    done < "$CONFIG_FILE"
    
    log_success "設定ファイルの読み込みが完了しました"
}

# AWSリソース情報の表示
show_aws_info() {
    log_info "AWSリソース情報を表示します"
    
    # アカウント情報
    log_info "AWSアカウント情報:"
    aws sts get-caller-identity --output table
    echo
    
    # VPC情報
    log_info "VPC情報:"
    vpc_list=$(aws ec2 describe-vpcs \
        --query 'Vpcs[*].[VpcId,State,CidrBlock,IsDefault]' \
        --output table)
    echo "$vpc_list"
    echo
    
    # サブネット情報
    log_info "サブネット情報:"
    subnet_list=$(aws ec2 describe-subnets \
        --query 'Subnets[*].[SubnetId,VpcId,AvailabilityZone,CidrBlock,State]' \
        --output table)
    echo "$subnet_list"
    echo
    
    # セキュリティグループ情報
    log_info "セキュリティグループ情報:"
    sg_list=$(aws ec2 describe-security-groups \
        --query 'SecurityGroups[*].[GroupId,GroupName,VpcId,Description]' \
        --output table)
    echo "$sg_list"
    echo
    
    # IAMロール情報
    log_info "IAMロール情報:"
    role_list=$(aws iam list-roles \
        --query 'Roles[?contains(RoleName, `GitHub`) || contains(RoleName, `Runner`) || contains(RoleName, `gopier`)].RoleName' \
        --output table)
    echo "$role_list"
    echo
    
    # AMI情報
    log_info "Amazon Linux 2 AMI情報:"
    ami_list=$(aws ec2 describe-images \
        --owners amazon \
        --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" "Name=state,Values=available" \
        --query 'Images[*].[ImageId,Name,Description,CreationDate]' \
        --output table)
    echo "$ami_list" | head -10
    echo
    
    # インスタンスタイプ情報
    log_info "推奨インスタンスタイプ情報:"
    local instance_types=("c5.xlarge" "c5.2xlarge" "c5.4xlarge" "c5.9xlarge")
    
    for instance_type in "${instance_types[@]}"; do
        echo "=== $instance_type ==="
        type_info=$(aws ec2 describe-instance-types \
            --instance-types "$instance_type" \
            --query 'InstanceTypes[0].[InstanceType,VCpuInfo.DefaultVCpus,MemoryInfo.SizeInMiB]' \
            --output table)
        echo "$type_info"
    done
    echo
    
    # コスト見積もり
    log_info "コスト見積もり（米国東部リージョン）:"
    echo "| インスタンスタイプ | 時間あたり | 月間（100時間） |"
    echo "|------------------|-----------|---------------|"
    echo "| c5.xlarge       | $0.17     | $17           |"
    echo "| c5.2xlarge      | $0.34     | $34           |"
    echo "| c5.4xlarge      | $0.68     | $68           |"
    echo "| c5.9xlarge      | $1.53     | $153          |"
    echo
    log_warning "実際のコストは使用時間とリージョンによって異なります"
    
    log_success "AWSリソース情報の表示が完了しました"
}

# セキュリティグループの作成
create_security_group() {
    log_info "セキュリティグループの作成中..."
    
    local sg_name="gopier-runner-sg"
    local vpc_id=$(aws ec2 describe-vpcs --query 'Vpcs[0].VpcId' --output text)
    
    # セキュリティグループが既に存在するかチェック
    local existing_sg=$(aws ec2 describe-security-groups \
        --filters "Name=group-name,Values=$sg_name" \
        --query 'SecurityGroups[0].GroupId' \
        --output text 2>/dev/null || echo "")
    
    if [ "$existing_sg" != "None" ] && [ -n "$existing_sg" ]; then
        log_info "セキュリティグループが既に存在します: $existing_sg"
        export EC2_SECURITY_GROUP_ID="$existing_sg"
        return 0
    fi
    
    # セキュリティグループを作成
    local sg_id=$(aws ec2 create-security-group \
        --group-name "$sg_name" \
        --description "Security group for GoPier CI runner" \
        --vpc-id "$vpc_id" \
        --query 'GroupId' \
        --output text)
    
    # SSHアクセスを許可
    aws ec2 authorize-security-group-ingress \
        --group-id "$sg_id" \
        --protocol tcp \
        --port 22 \
        --cidr 0.0.0.0/0
    
    # HTTPSアクセスを許可
    aws ec2 authorize-security-group-ingress \
        --group-id "$sg_id" \
        --protocol tcp \
        --port 443 \
        --cidr 0.0.0.0/0
    
    log_success "セキュリティグループを作成しました: $sg_id"
    export EC2_SECURITY_GROUP_ID="$sg_id"
}

# IAMロールの作成（権限チェック付き）
create_iam_role() {
    log_info "IAMロールの作成中..."
    
    local role_name="GitHubActionsRunnerRole"
    
    # ロールが既に存在するかチェック
    if aws iam get-role --role-name "$role_name" &> /dev/null; then
        log_info "IAMロールが既に存在します: $role_name"
        export EC2_IAM_ROLE_NAME="$role_name"
        return 0
    fi
    
    # IAM作成権限のチェック
    if ! aws iam create-role --role-name "test-role-check" --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}' &> /dev/null; then
        log_warning "IAMロール作成権限がありません。既存のロールを使用してください。"
        log_info "利用可能なIAMロール:"
        aws iam list-roles --query 'Roles[?contains(RoleName, `GitHub`) || contains(RoleName, `Runner`) || contains(RoleName, `gopier`)].RoleName' --output table
        echo
        log_info "既存のロール名を入力してください（空の場合はスキップ）:"
        read -r existing_role
        
        if [ -n "$existing_role" ]; then
            export EC2_IAM_ROLE_NAME="$existing_role"
            log_success "既存のIAMロールを使用します: $existing_role"
        else
            log_warning "IAMロールの設定をスキップします"
        fi
        return 0
    fi
    
    # テストロールを削除
    aws iam delete-role --role-name "test-role-check" &> /dev/null || true
    
    # 信頼ポリシーを作成
    cat > trust-policy.json << EOF
{
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
}
EOF
    
    # ロールを作成
    aws iam create-role \
        --role-name "$role_name" \
        --assume-role-policy-document file://trust-policy.json
    
    # ポリシーを作成
    cat > runner-policy.json << EOF
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
EOF
    
    # ポリシーを作成してロールにアタッチ
    aws iam create-policy \
        --policy-name "GitHubActionsRunnerPolicy" \
        --policy-document file://runner-policy.json
    
    aws iam attach-role-policy \
        --role-name "$role_name" \
        --policy-arn "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):policy/GitHubActionsRunnerPolicy"
    
    # インスタンスプロファイルを作成
    aws iam create-instance-profile --instance-profile-name "$role_name"
    aws iam add-role-to-instance-profile \
        --instance-profile-name "$role_name" \
        --role-name "$role_name"
    
    # 一時ファイルを削除
    rm -f trust-policy.json runner-policy.json
    
    log_success "IAMロールを作成しました: $role_name"
    export EC2_IAM_ROLE_NAME="$role_name"
}

# 設定の検証
validate_config() {
    log_info "設定の検証中..."
    
    # 設定ファイルの存在確認
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "設定ファイルが生成されていません"
        return 1
    fi
    
    # 必須項目の確認
    local required_vars=(
        "AWS_ACCESS_KEY_ID"
        "AWS_SECRET_ACCESS_KEY"
        "AWS_REGION"
        "EC2_INSTANCE_TYPE"
        "EC2_IMAGE_ID"
        "EC2_SUBNET_ID"
        "GH_PERSONAL_ACCESS_TOKEN"
    )
    
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        local value=$(grep "^${var}=" "$CONFIG_FILE" | cut -d'=' -f2)
        if [ -z "$value" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -gt 0 ]; then
        log_error "以下の項目が設定されていません:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        return 1
    fi
    
    log_success "設定の検証が完了しました"
}

# 設定ファイルの表示
show_config() {
    log_info "生成された設定ファイル:"
    echo "=========================================="
    cat "$CONFIG_FILE"
    echo "=========================================="
}

# 完全自動設定
setup() {
    log_info "AWSランナー設定ファイルの自動生成を開始します"
    
    check_aws_cli
    get_aws_info
    get_user_input
    generate_config
    validate_config
    show_config
    
    log_success "AWSランナー設定ファイルの作成が完了しました"
    echo
    log_info "次のステップ:"
    log_info "1. 設定ファイルを確認: cat $CONFIG_FILE"
    log_info "2. AWSランナーを設定: $0 deploy"
    echo
    log_info "注意事項:"
    log_info "- 設定ファイルには機密情報が含まれています"
    log_info "- 設定ファイルをGitにコミットしないでください"
    log_info "- .gitignoreに aws-runner-config.env を追加することを推奨します"
}

# AWSランナーのデプロイ
deploy() {
    log_info "AWSランナーの設定を開始します"
    
    check_aws_cli
    load_config
    
    # 必須環境変数の確認
    local required_vars=(
        "AWS_ACCESS_KEY_ID"
        "AWS_SECRET_ACCESS_KEY"
        "AWS_REGION"
        "EC2_INSTANCE_TYPE"
        "EC2_IMAGE_ID"
        "EC2_SUBNET_ID"
        "GH_PERSONAL_ACCESS_TOKEN"
    )
    
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -gt 0 ]; then
        log_error "以下の環境変数が設定されていません:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        log_info "setup コマンドを実行して設定ファイルを作成してください"
        return 1
    fi
    
    log_success "設定確認完了"
    
    create_security_group
    create_iam_role
    
    # 設定ファイルを更新
    if [ -n "$EC2_SECURITY_GROUP_ID" ]; then
        sed -i.bak "s/EC2_SECURITY_GROUP_ID=/EC2_SECURITY_GROUP_ID=$EC2_SECURITY_GROUP_ID/" "$CONFIG_FILE"
    fi
    
    if [ -n "$EC2_IAM_ROLE_NAME" ]; then
        sed -i.bak "s/EC2_IAM_ROLE_NAME=/EC2_IAM_ROLE_NAME=$EC2_IAM_ROLE_NAME/" "$CONFIG_FILE"
    fi
    
    rm -f "${CONFIG_FILE}.bak"
    
    log_success "AWSランナー設定が完了しました"
    log_info "次のステップ:"
    log_info "1. GitHub Secretsに設定値を追加してください"
    log_info "2. CIワークフローを実行してテストしてください"
}

# リソースクリーンアップ
cleanup() {
    log_info "リソースクリーンアップを開始します"
    
    check_aws_cli
    load_config
    
    # セキュリティグループの削除
    if [ -n "$EC2_SECURITY_GROUP_ID" ]; then
        log_info "セキュリティグループを削除中: $EC2_SECURITY_GROUP_ID"
        aws ec2 delete-security-group --group-id "$EC2_SECURITY_GROUP_ID" 2>/dev/null || log_warning "セキュリティグループの削除に失敗しました"
    fi
    
    # IAMロールの削除
    if [ -n "$EC2_IAM_ROLE_NAME" ]; then
        log_info "IAMロールを削除中: $EC2_IAM_ROLE_NAME"
        # インスタンスプロファイルからロールを削除
        aws iam remove-role-from-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME" --role-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || true
        # インスタンスプロファイルを削除
        aws iam delete-instance-profile --instance-profile-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || true
        # ポリシーをデタッチ
        aws iam detach-role-policy --role-name "$EC2_IAM_ROLE_NAME" --policy-arn "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):policy/GitHubActionsRunnerPolicy" 2>/dev/null || true
        # ポリシーを削除
        aws iam delete-policy --policy-arn "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):policy/GitHubActionsRunnerPolicy" 2>/dev/null || true
        # ロールを削除
        aws iam delete-role --role-name "$EC2_IAM_ROLE_NAME" 2>/dev/null || log_warning "IAMロールの削除に失敗しました"
    fi
    
    # 設定ファイルの削除
    if [ -f "$CONFIG_FILE" ]; then
        log_info "設定ファイルを削除中: $CONFIG_FILE"
        rm -f "$CONFIG_FILE"
    fi
    
    log_success "リソースクリーンアップが完了しました"
}

# メイン処理
main() {
    case "${1:-help}" in
        setup)
            setup
            ;;
        info)
            check_aws_cli
            show_aws_info
            ;;
        config)
            if [ -f "$CONFIG_FILE" ]; then
                show_config
            else
                log_error "設定ファイルが見つかりません: $CONFIG_FILE"
                log_info "setup コマンドを実行して設定ファイルを作成してください"
            fi
            ;;
        deploy)
            deploy
            ;;
        cleanup)
            cleanup
            ;;
        help|*)
            show_help
            ;;
    esac
}

# スクリプトの実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 