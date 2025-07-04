#!/bin/bash

# EC2 Runner Management System Test Script
# EC2ランナー管理システムのテストスクリプト

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

# テスト結果
TESTS_PASSED=0
TESTS_FAILED=0

# テスト関数
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    log "Running test: $test_name"
    
    if eval "$test_command"; then
        log "✅ PASS: $test_name"
        ((TESTS_PASSED++))
    else
        error "❌ FAIL: $test_name"
        ((TESTS_FAILED++))
    fi
}

# ヘルプ表示
show_help() {
    cat << EOF
EC2 Runner Management System Test Script

使用方法:
    $0 [options]

オプション:
    --help       - このヘルプを表示
    --dry-run    - 実際のAWS操作を行わずにテスト
    --verbose    - 詳細な出力

テスト内容:
    1. スクリプトの構文チェック
    2. 依存関係の確認
    3. 設定の検証
    4. AWS CLIの確認
    5. GitHub APIの確認
    6. スクリプトのヘルプ表示テスト

環境変数:
    AWS_ACCESS_KEY_ID      - AWSアクセスキー
    AWS_SECRET_ACCESS_KEY  - AWSシークレットキー
    AWS_REGION            - AWSリージョン
    GITHUB_TOKEN          - GitHub Personal Access Token
    GITHUB_REPOSITORY     - GitHubリポジトリ (owner/repo形式)

例:
    $0 --dry-run
    $0 --verbose
EOF
}

# パラメータの解析
parse_args() {
    DRY_RUN=false
    VERBOSE=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_help
                exit 0
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            *)
                error "不明なオプション: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# テスト1: スクリプトの構文チェック
test_syntax_check() {
    log "Testing script syntax..."
    
    # メインスクリプトの構文チェック
    if bash -n "$SCRIPT_DIR/ec2-runner-manager.sh"; then
        log "✅ Main script syntax is valid"
    else
        error "❌ Main script syntax error"
        return 1
    fi
    
    # ヘルパースクリプトの構文チェック
    if bash -n "$SCRIPT_DIR/ec2-runner-helper.sh"; then
        log "✅ Helper script syntax is valid"
    else
        error "❌ Helper script syntax error"
        return 1
    fi
}

# テスト2: 依存関係の確認
test_dependencies() {
    log "Testing dependencies..."
    
    local missing_deps=()
    
    # 必要なコマンドの確認
    for cmd in aws curl jq; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error "❌ Missing dependencies: ${missing_deps[*]}"
        return 1
    else
        log "✅ All dependencies are available"
    fi
}

# テスト3: 設定の検証
test_configuration() {
    log "Testing configuration..."
    
    local missing_vars=()
    
    # 必要な環境変数の確認
    for var in AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_REGION; do
        if [[ -z "${!var:-}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        error "❌ Missing environment variables: ${missing_vars[*]}"
        return 1
    else
        log "✅ AWS configuration is valid"
    fi
    
    # GitHub設定の確認
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        error "❌ Missing GITHUB_TOKEN"
        return 1
    fi
    
    if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
        error "❌ Missing GITHUB_REPOSITORY"
        return 1
    fi
    
    log "✅ GitHub configuration is valid"
}

# テスト4: AWS CLIの確認
test_aws_cli() {
    log "Testing AWS CLI..."
    
    if ! aws sts get-caller-identity &> /dev/null; then
        error "❌ AWS CLI authentication failed"
        return 1
    fi
    
    local account_id
    account_id=$(aws sts get-caller-identity --query 'Account' --output text)
    log "✅ AWS CLI is working (Account: $account_id)"
}

# テスト5: GitHub APIの確認
test_github_api() {
    log "Testing GitHub API..."
    
    local response
    response=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY" | \
        jq -r '.name' 2>/dev/null || echo "")
    
    if [[ -z "$response" || "$response" == "null" ]]; then
        error "❌ GitHub API authentication failed"
        return 1
    fi
    
    log "✅ GitHub API is working (Repository: $response)"
}

# テスト6: スクリプトのヘルプ表示テスト
test_script_help() {
    log "Testing script help output..."
    
    # メインスクリプトのヘルプ
    if ! "$SCRIPT_DIR/ec2-runner-manager.sh" --help &> /dev/null; then
        error "❌ Main script help failed"
        return 1
    fi
    
    # ヘルパースクリプトのヘルプ
    if ! "$SCRIPT_DIR/ec2-runner-helper.sh" --help &> /dev/null; then
        error "❌ Helper script help failed"
        return 1
    fi
    
    log "✅ Script help output is working"
}

# テスト7: パラメータ解析テスト
test_parameter_parsing() {
    log "Testing parameter parsing..."
    
    # 無効なコマンドのテスト
    if "$SCRIPT_DIR/ec2-runner-manager.sh" invalid-command &> /dev/null; then
        error "❌ Invalid command should fail"
        return 1
    fi
    
    # 無効なオプションのテスト
    if "$SCRIPT_DIR/ec2-runner-manager.sh" --invalid-option &> /dev/null; then
        error "❌ Invalid option should fail"
        return 1
    fi
    
    log "✅ Parameter parsing is working correctly"
}

# テスト8: ドライランテスト
test_dry_run() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log "Testing dry run mode..."
        
        # ドライランでのクリーンアップテスト
        if ! "$SCRIPT_DIR/ec2-runner-helper.sh" cleanup --dry-run &> /dev/null; then
            error "❌ Dry run cleanup failed"
            return 1
        fi
        
        log "✅ Dry run mode is working"
    fi
}

# メイン処理
main() {
    log "Starting EC2 Runner Management System Tests"
    
    # 引数の解析
    parse_args "$@"
    
    # テストの実行
    run_test "Syntax Check" "test_syntax_check"
    run_test "Dependencies" "test_dependencies"
    run_test "Configuration" "test_configuration"
    run_test "AWS CLI" "test_aws_cli"
    run_test "GitHub API" "test_github_api"
    run_test "Script Help" "test_script_help"
    run_test "Parameter Parsing" "test_parameter_parsing"
    run_test "Dry Run Mode" "test_dry_run"
    
    # 結果の表示
    echo ""
    echo "=== Test Results ==="
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo "Total: $((TESTS_PASSED + TESTS_FAILED))"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo ""
        echo "✅ All tests passed! The EC2 Runner Management System is ready to use."
        exit 0
    else
        echo ""
        echo "❌ Some tests failed. Please check the configuration and dependencies."
        exit 1
    fi
}

# スクリプトの実行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 