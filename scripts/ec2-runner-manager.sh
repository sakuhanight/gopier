#!/bin/bash

# EC2 Self-Hosted Runner Manager
# GitHub ActionsでEC2インスタンスを自動作成・管理するスクリプト

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
EC2 Self-Hosted Runner Manager

使用方法:
    $0 <command> [options]

コマンド:
    start       - EC2インスタンスを起動してGitHubランナーを設定
    stop        - EC2インスタンスを停止
    status      - EC2インスタンスのステータスを確認
    cleanup     - 古いインスタンスをクリーンアップ
    setup       - 初期設定を実行
    setup-iam   - IAM Instance Profileを設定
    verify-iam  - IAM設定を確認

オプション:
    --label     - ランナーラベル (デフォルト: ec2-runner-\$RANDOM)
    --type      - EC2インスタンスタイプ (デフォルト: t3.medium)
    --image     - AMI ID (デフォルト: Amazon Linux 2)
    --subnet    - サブネットID
    --sg        - セキュリティグループID
    --role      - IAMロール名
    --timeout   - タイムアウト時間（分）(デフォルト: 60)
    --role-name - IAMロール名 (setup-iam用、デフォルト: EC2RunnerRole)
    --profile-name - インスタンスプロファイル名 (setup-iam用、デフォルト: EC2RunnerRole)
    --help      - このヘルプを表示

環境変数 (自動設定対応):
    AWS_ACCESS_KEY_ID      - AWSアクセスキー (AWS CLIから自動取得)
    AWS_SECRET_ACCESS_KEY  - AWSシークレットキー (AWS CLIから自動取得)
    AWS_REGION            - AWSリージョン (AWS CLIから自動取得、デフォルト: us-east-1)
    GITHUB_TOKEN          - GitHub Personal Access Token (GitHub CLIから自動取得)
    GITHUB_REPOSITORY     - GitHubリポジトリ (複数方法で自動検出)

例:
    $0 start --label my-runner --type t3.large
    $0 stop --label my-runner
    $0 status --label my-runner
    $0 setup-iam --role-name MyRunnerRole --profile-name MyRunnerProfile
    $0 verify-iam

自動設定について:
    このスクリプトは必要な環境変数を自動的に設定します:
    - AWS認証情報: aws configure または aws sso login で設定済みの場合
    - GitHubトークン: GitHub CLI (gh) で認証済みの場合
    - GitHubリポジトリ: 以下の順序で自動検出
        1. GitHub CLI (gh repo view)
        2. Gitリモート (origin, upstream, github)
        3. 設定ファイル (.github/config)
        4. package.json (Node.jsプロジェクト)
        5. go.mod (Goプロジェクト)
EOF
}

# 環境変数の自動設定
setup_environment() {
    log "環境変数を自動設定中..."
    
    # AWS認証情報をAWS CLIから取得
    if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]] || [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]] || [[ -z "${AWS_REGION:-}" ]]; then
        log "AWS認証情報をAWS CLIから取得中..."
        
        # AWS CLIの設定を確認
        if ! command -v aws &> /dev/null; then
            error "AWS CLIがインストールされていません"
            exit 1
        fi
        
        # AWS認証情報のテスト
        if ! aws sts get-caller-identity &> /dev/null; then
            error "AWS認証情報が無効です。aws configure または aws sso login を実行してください"
            exit 1
        fi
        
        # AWS認証情報を環境変数に設定
        if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
            export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id)
            log "AWS_ACCESS_KEY_ID を設定しました"
        fi
        
        if [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
            export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key)
            log "AWS_SECRET_ACCESS_KEY を設定しました"
        fi
        
        if [[ -z "${AWS_REGION:-}" ]]; then
            export AWS_REGION=$(aws configure get region)
            if [[ -z "$AWS_REGION" ]]; then
                # デフォルトリージョンを設定
                export AWS_REGION="us-east-1"
                log "AWS_REGION をデフォルト値 (us-east-1) に設定しました"
            else
                log "AWS_REGION を設定しました: $AWS_REGION"
            fi
        fi
    fi
    
    # GitHub認証情報の確認と設定
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        # GitHub CLIからトークンを取得を試行
        if command -v gh &> /dev/null; then
            if gh auth status &> /dev/null; then
                export GITHUB_TOKEN=$(gh auth token)
                log "GITHUB_TOKEN をGitHub CLIから取得しました"
            else
                error "GITHUB_TOKEN が設定されていません。以下のいずれかの方法で設定してください:"
                error "1. 環境変数 GITHUB_TOKEN を設定"
                error "2. GitHub CLI で認証: gh auth login"
                error "3. 設定ファイル ~/.github/config にトークンを保存"
                exit 1
            fi
        else
            error "GITHUB_TOKEN が設定されていません。以下のいずれかの方法で設定してください:"
            error "1. 環境変数 GITHUB_TOKEN を設定"
            error "2. GitHub CLI をインストールして認証: gh auth login"
            error "3. 設定ファイル ~/.github/config にトークンを保存"
            exit 1
        fi
    fi
    
    if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
        log "GITHUB_REPOSITORY を自動検出中..."
        
        # 方法1: GitHub CLIから取得を試行
        if command -v gh &> /dev/null && gh auth status &> /dev/null; then
            local repo_info
            repo_info=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null || echo "")
            if [[ -n "$repo_info" ]]; then
                export GITHUB_REPOSITORY="$repo_info"
                log "GITHUB_REPOSITORY をGitHub CLIから取得しました: $GITHUB_REPOSITORY"
            fi
        fi
        
        # 方法2: Gitリモートから取得を試行
        if [[ -z "${GITHUB_REPOSITORY:-}" ]] && command -v git &> /dev/null && git rev-parse --git-dir &> /dev/null; then
            # 複数のリモート名を試行（origin, upstream, github等）
            local remote_names=("origin" "upstream" "github")
            local found_repo=""
            
            for remote_name in "${remote_names[@]}"; do
                local remote_url
                remote_url=$(git remote get-url "$remote_name" 2>/dev/null || echo "")
                if [[ -n "$remote_url" ]]; then
                    # git@github.com:owner/repo.git または https://github.com/owner/repo.git から owner/repo を抽出
                    if [[ "$remote_url" =~ github\.com[:/]([^/]+/[^/]+?)(\.git)?$ ]]; then
                        found_repo="${BASH_REMATCH[1]}"
                        log "GITHUB_REPOSITORY をGitリモート '$remote_name' から取得しました: $found_repo"
                        break
                    fi
                fi
            done
            
            if [[ -n "$found_repo" ]]; then
                export GITHUB_REPOSITORY="$found_repo"
            fi
        fi
        
        # 方法3: 設定ファイルから読み込みを試行
        if [[ -z "${GITHUB_REPOSITORY:-}" ]] && [[ -f "$PROJECT_ROOT/.github/config" ]]; then
            local config_repo
            config_repo=$(grep -E '^repository[[:space:]]*=' "$PROJECT_ROOT/.github/config" | head -1 | sed 's/^repository[[:space:]]*=[[:space:]]*//' 2>/dev/null || echo "")
            if [[ -n "$config_repo" ]]; then
                export GITHUB_REPOSITORY="$config_repo"
                log "GITHUB_REPOSITORY を設定ファイルから取得しました: $GITHUB_REPOSITORY"
            fi
        fi
        
        # 方法4: package.jsonから読み込みを試行（Node.jsプロジェクトの場合）
        if [[ -z "${GITHUB_REPOSITORY:-}" ]] && [[ -f "$PROJECT_ROOT/package.json" ]]; then
            local package_repo
            package_repo=$(jq -r '.repository.url // .repository // empty' "$PROJECT_ROOT/package.json" 2>/dev/null | sed 's|^https://github.com/||' | sed 's|^git@github.com:||' | sed 's|\.git$||' || echo "")
            if [[ -n "$package_repo" && "$package_repo" != "null" ]]; then
                export GITHUB_REPOSITORY="$package_repo"
                log "GITHUB_REPOSITORY をpackage.jsonから取得しました: $GITHUB_REPOSITORY"
            fi
        fi
        
        # 方法5: go.modから読み込みを試行（Goプロジェクトの場合）
        if [[ -z "${GITHUB_REPOSITORY:-}" ]] && [[ -f "$PROJECT_ROOT/go.mod" ]]; then
            local go_module
            go_module=$(head -1 "$PROJECT_ROOT/go.mod" | sed 's/^module //' 2>/dev/null || echo "")
            if [[ -n "$go_module" && "$go_module" =~ github\.com/([^/]+/[^/]+) ]]; then
                export GITHUB_REPOSITORY="${BASH_REMATCH[1]}"
                log "GITHUB_REPOSITORY をgo.modから取得しました: $GITHUB_REPOSITORY"
            fi
        fi
        
        if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
            error "GITHUB_REPOSITORY を自動検出できませんでした。以下のいずれかの方法で設定してください:"
            error "1. 環境変数 GITHUB_REPOSITORY を設定 (例: owner/repo)"
            error "2. GitHubリポジトリのディレクトリで実行"
            error "3. GitHub CLI で認証: gh auth login"
            error "4. 設定ファイル .github/config に repository=owner/repo を追加"
            error "5. package.json または go.mod にリポジトリ情報を追加"
            error ""
            error "現在のディレクトリ: $(pwd)"
            if command -v git &> /dev/null && git rev-parse --git-dir &> /dev/null; then
                error "Gitリモート一覧:"
                git remote -v 2>/dev/null | while read -r remote_name url; do
                    error "  $remote_name: $url"
                done
            fi
            exit 1
        fi
    fi
    
    # 設定の確認
    log "設定確認:"
    log "  AWS_REGION: $AWS_REGION"
    log "  GITHUB_REPOSITORY: $GITHUB_REPOSITORY"
    log "  GITHUB_TOKEN: ${GITHUB_TOKEN:0:8}..."
}

# 設定の検証
validate_config() {
    # 環境変数の自動設定を実行
    setup_environment
    
    # 最終的な設定確認
    local missing_vars=()
    
    if [[ -z "${AWS_ACCESS_KEY_ID:-}" ]]; then
        missing_vars+=("AWS_ACCESS_KEY_ID")
    fi
    if [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
        missing_vars+=("AWS_SECRET_ACCESS_KEY")
    fi
    if [[ -z "${AWS_REGION:-}" ]]; then
        missing_vars+=("AWS_REGION")
    fi
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        missing_vars+=("GITHUB_TOKEN")
    fi
    if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
        missing_vars+=("GITHUB_REPOSITORY")
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
    
    # AWS認証情報のテスト
    if ! aws sts get-caller-identity &> /dev/null; then
        error "AWS認証情報が無効です。以下のいずれかの方法で認証してください:"
        error "1. aws configure で認証情報を設定"
        error "2. aws sso login でSSO認証を実行"
        error "3. 環境変数で認証情報を設定"
        exit 1
    fi
    
    # 認証情報の詳細を表示
    local caller_identity
    caller_identity=$(aws sts get-caller-identity --output json)
    local account_id=$(echo "$caller_identity" | jq -r '.Account')
    local user_arn=$(echo "$caller_identity" | jq -r '.Arn')
    
    log "AWS認証情報確認:"
    log "  Account ID: $account_id"
    log "  User ARN: $user_arn"
}

# パラメータの解析
parse_args() {
    COMMAND=""
    LABEL=""
    INSTANCE_TYPE="t3.medium"
    IMAGE_ID=""
    SUBNET_ID=""
    SECURITY_GROUP_ID=""
    IAM_ROLE_NAME=""
    TIMEOUT_MINUTES=60
    IAM_ROLE_NAME_SETUP="EC2RunnerRole"
    IAM_PROFILE_NAME_SETUP="EC2RunnerRole"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            start|stop|status|cleanup|setup|setup-iam|verify-iam)
                COMMAND="$1"
                shift
                ;;
            --label)
                LABEL="$2"
                shift 2
                ;;
            --type)
                INSTANCE_TYPE="$2"
                shift 2
                ;;
            --image)
                IMAGE_ID="$2"
                shift 2
                ;;
            --subnet)
                SUBNET_ID="$2"
                shift 2
                ;;
            --sg)
                SECURITY_GROUP_ID="$2"
                shift 2
                ;;
            --role)
                IAM_ROLE_NAME="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT_MINUTES="$2"
                shift 2
                ;;
            --role-name)
                IAM_ROLE_NAME_SETUP="$2"
                shift 2
                ;;
            --profile-name)
                IAM_PROFILE_NAME_SETUP="$2"
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
    
    # デフォルト値の設定
    if [[ -z "$LABEL" ]]; then
        LABEL="ec2-runner-$(date +%s)"
    fi
    
    if [[ -z "$IMAGE_ID" ]]; then
        # Amazon Linux 2の最新AMIを取得
        IMAGE_ID=$(aws ec2 describe-images \
            --owners amazon \
            --filters "Name=name,Values=amzn2-ami-hvm-*-x86_64-gp2" \
            --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
            --output text)
    fi
}

# EC2インスタンスの起動
start_runner() {
    log "EC2ランナーを起動中: $LABEL"
    
    # ユーザーデータスクリプトの作成
    local user_data_script
    user_data_script=$(create_user_data_script)
    
    # EC2インスタンスの作成
    local instance_id
    instance_id=$(aws ec2 run-instances \
        --image-id "$IMAGE_ID" \
        --instance-type "$INSTANCE_TYPE" \
        --key-name "${EC2_KEY_NAME:-}" \
        --subnet-id "$SUBNET_ID" \
        --security-group-ids "$SECURITY_GROUP_ID" \
        --iam-instance-profile Name="$IAM_ROLE_NAME" \
        --user-data "$user_data_script" \
        --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=GitHub-Runner-$LABEL},{Key=GitHubRepository,Value=$GITHUB_REPOSITORY},{Key=RunnerLabel,Value=$LABEL}]" \
        --query 'Instances[0].InstanceId' \
        --output text)
    
    log "インスタンス作成完了: $instance_id"
    
    # インスタンスの起動完了を待機
    log "インスタンスの起動を待機中..."
    aws ec2 wait instance-running --instance-ids "$instance_id"
    
    # パブリックIPアドレスの取得
    local public_ip
    public_ip=$(aws ec2 describe-instances \
        --instance-ids "$instance_id" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text)
    
    log "インスタンス起動完了: $instance_id (IP: $public_ip)"
    
    # ランナーの登録完了を待機
    wait_for_runner_registration "$instance_id" "$public_ip"
    
    # 出力
    echo "label=$LABEL" >> $GITHUB_OUTPUT
    echo "ec2-instance-id=$instance_id" >> $GITHUB_OUTPUT
    echo "public-ip=$public_ip" >> $GITHUB_OUTPUT
    
    log "EC2ランナーの起動が完了しました"
}

# ユーザーデータスクリプトの作成
create_user_data_script() {
    cat << EOF
#!/bin/bash
set -euo pipefail

# ログ設定
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

echo "=== EC2 GitHub Runner Setup Started ==="

# 環境変数の設定
GITHUB_REPOSITORY="$GITHUB_REPOSITORY"
RUNNER_TOKEN="$RUNNER_TOKEN"
RUNNER_LABEL="$LABEL"

# システムの更新
yum update -y

# 必要なパッケージのインストール
yum install -y git wget unzip curl jq

# Goのインストール（最新版）
GO_VERSION="1.22.0"
wget -q https://go.dev/dl/go\${GO_VERSION}.linux-amd64.tar.gz
tar -C /usr/local -xzf go\${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=\$PATH:/usr/local/go/bin' >> /home/ec2-user/.bashrc
echo 'export PATH=\$PATH:/usr/local/go/bin' >> /root/.bashrc

# GitHubランナーのダウンロードと設定
cd /home/ec2-user
RUNNER_VERSION="2.311.0"
wget -q https://github.com/actions/runner/releases/download/v\${RUNNER_VERSION}/actions-runner-linux-x64-\${RUNNER_VERSION}.tar.gz
tar -xzf actions-runner-linux-x64-\${RUNNER_VERSION}.tar.gz
rm actions-runner-linux-x64-\${RUNNER_VERSION}.tar.gz

# ランナーの設定
./config.sh \\
    --url https://github.com/\${GITHUB_REPOSITORY} \\
    --token \${RUNNER_TOKEN} \\
    --name \${RUNNER_LABEL} \\
    --labels \${RUNNER_LABEL},linux,aws \\
    --unattended \\
    --replace

# ランナーサービスのインストールと起動
./svc.sh install
./svc.sh start

# メモリ最適化設定
echo 'export GOGC=100' >> /home/ec2-user/.bashrc
echo 'export GOMEMLIMIT=4GiB' >> /home/ec2-user/.bashrc
echo 'export GOMAXPROCS=8' >> /home/ec2-user/.bashrc

echo "=== EC2 GitHub Runner Setup Completed ==="
EOF
}

# ランナー登録の待機
wait_for_runner_registration() {
    local instance_id="$1"
    local public_ip="$2"
    local max_attempts=30
    local attempt=1
    
    log "ランナーの登録完了を待機中..."
    
    while [[ $attempt -le $max_attempts ]]; do
        # GitHub APIでランナーのステータスを確認
        local runners
        runners=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
            "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runners" | \
            jq -r '.runners[] | select(.name == "'$LABEL'") | .status' 2>/dev/null || echo "")
        
        if [[ "$runners" == "online" ]]; then
            log "ランナーの登録が完了しました"
            return 0
        fi
        
        log "ランナーの登録待機中... (試行 $attempt/$max_attempts)"
        sleep 30
        ((attempt++))
    done
    
    error "ランナーの登録がタイムアウトしました"
    return 1
}

# EC2インスタンスの停止
stop_runner() {
    log "EC2ランナーを停止中: $LABEL"
    
    # ランナーラベルからインスタンスIDを取得
    local instance_id
    instance_id=$(aws ec2 describe-instances \
        --filters "Name=tag:RunnerLabel,Values=$LABEL" "Name=instance-state-name,Values=running,stopping,stopped" \
        --query 'Reservations[0].Instances[0].InstanceId' \
        --output text)
    
    if [[ "$instance_id" == "None" || -z "$instance_id" ]]; then
        error "ラベル $LABEL のインスタンスが見つかりません"
        return 1
    fi
    
    # GitHubランナーの削除
    log "GitHubランナーを削除中..."
    local runner_id
    runner_id=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runners" | \
        jq -r '.runners[] | select(.name == "'$LABEL'") | .id' 2>/dev/null || echo "")
    
    if [[ -n "$runner_id" && "$runner_id" != "null" ]]; then
        curl -X DELETE -H "Authorization: token $GITHUB_TOKEN" \
            "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runners/$runner_id"
        log "GitHubランナーを削除しました: $runner_id"
    fi
    
    # EC2インスタンスの停止
    aws ec2 terminate-instances --instance-ids "$instance_id"
    log "インスタンス停止要求を送信しました: $instance_id"
    
    # インスタンスの停止完了を待機
    aws ec2 wait instance-terminated --instance-ids "$instance_id"
    log "インスタンスの停止が完了しました: $instance_id"
}

# EC2インスタンスのステータス確認
check_status() {
    log "EC2ランナーのステータスを確認中: $LABEL"
    
    # EC2インスタンスのステータス
    local instance_info
    instance_info=$(aws ec2 describe-instances \
        --filters "Name=tag:RunnerLabel,Values=$LABEL" \
        --query 'Reservations[0].Instances[0]' \
        --output json 2>/dev/null || echo "{}")
    
    if [[ "$instance_info" == "{}" ]]; then
        echo "インスタンスが見つかりません: $LABEL"
        return 1
    fi
    
    local instance_id=$(echo "$instance_info" | jq -r '.InstanceId')
    local state=$(echo "$instance_info" | jq -r '.State.Name')
    local public_ip=$(echo "$instance_info" | jq -r '.PublicIpAddress // "N/A"')
    
    echo "=== EC2 Instance Status ==="
    echo "Instance ID: $instance_id"
    echo "State: $state"
    echo "Public IP: $public_ip"
    
    # GitHubランナーのステータス
    local runner_info
    runner_info=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runners" | \
        jq -r '.runners[] | select(.name == "'$LABEL'")' 2>/dev/null || echo "{}")
    
    if [[ "$runner_info" == "{}" ]]; then
        echo "GitHubランナーが見つかりません: $LABEL"
    else
        local runner_status=$(echo "$runner_info" | jq -r '.status')
        local busy=$(echo "$runner_info" | jq -r '.busy')
        echo "=== GitHub Runner Status ==="
        echo "Status: $runner_status"
        echo "Busy: $busy"
    fi
}

# 古いインスタンスのクリーンアップ
cleanup_old_instances() {
    log "古いEC2インスタンスをクリーンアップ中..."
    
    # 24時間以上経過したインスタンスを検索
    local old_instances
    old_instances=$(aws ec2 describe-instances \
        --filters "Name=tag:GitHubRepository,Values=$GITHUB_REPOSITORY" "Name=instance-state-name,Values=running" \
        --query 'Reservations[].Instances[?LaunchTime<`'$(date -d '24 hours ago' -u +%Y-%m-%dT%H:%M:%S)'`].[InstanceId,Tags[?Key==`RunnerLabel`].Value|[0]]' \
        --output text)
    
    if [[ -z "$old_instances" ]]; then
        log "クリーンアップ対象のインスタンスはありません"
        return 0
    fi
    
    echo "$old_instances" | while read -r instance_id label; do
        if [[ -n "$instance_id" && "$instance_id" != "None" ]]; then
            log "古いインスタンスを停止中: $instance_id ($label)"
            stop_runner --label "$label"
        fi
    done
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
    create_iam_role "$IAM_ROLE_NAME_SETUP"
    create_and_attach_policies "$IAM_ROLE_NAME_SETUP"
    create_instance_profile "$IAM_ROLE_NAME_SETUP" "$IAM_PROFILE_NAME_SETUP"
    verify_iam_setup "$IAM_PROFILE_NAME_SETUP"
    
    log "IAM Instance Profileの設定が完了しました"
    log ""
    log "GitHub Secretsで以下の値を設定してください:"
    log "  EC2_IAM_ROLE_NAME: $IAM_PROFILE_NAME_SETUP"
    log ""
    log "注意: インスタンスプロファイルの作成後、反映まで数分かかる場合があります"
}

# IAM設定の確認
verify_iam() {
    log "IAM設定を確認中..."
    
    # AWS CLIの確認
    check_aws_cli
    
    # デフォルトのプロファイル名を使用
    local profile_name="${IAM_ROLE_NAME:-EC2RunnerRole}"
    
    if verify_iam_setup "$profile_name"; then
        log "✅ IAM設定は正常です"
    else
        error "❌ IAM設定に問題があります"
        log "以下のコマンドでIAM設定を実行してください:"
        log "  $0 setup-iam --role-name $profile_name --profile-name $profile_name"
        exit 1
    fi
}

# 初期設定
setup() {
    log "EC2ランナー管理の初期設定を実行中..."
    
    # AWS CLIの確認
    check_aws_cli
    
    # 必要なIAMロールの確認
    if [[ -n "$IAM_ROLE_NAME" ]]; then
        if ! aws iam get-role --role-name "$IAM_ROLE_NAME" &> /dev/null; then
            error "IAMロール $IAM_ROLE_NAME が存在しません"
            exit 1
        fi
    fi
    
    # セキュリティグループの確認
    if [[ -n "$SECURITY_GROUP_ID" ]]; then
        if ! aws ec2 describe-security-groups --group-ids "$SECURITY_GROUP_ID" &> /dev/null; then
            error "セキュリティグループ $SECURITY_GROUP_ID が存在しません"
            exit 1
        fi
    fi
    
    # サブネットの確認
    if [[ -n "$SUBNET_ID" ]]; then
        if ! aws ec2 describe-subnets --subnet-ids "$SUBNET_ID" &> /dev/null; then
            error "サブネット $SUBNET_ID が存在しません"
            exit 1
        fi
    fi
    
    log "初期設定が完了しました"
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
            cleanup_old_instances
            ;;
        setup)
            setup
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