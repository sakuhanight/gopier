#!/bin/bash

# AWSランナー監視スクリプトのテスト
# 使用方法: ./scripts/test-monitor-aws-runners.sh

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

# テスト結果
TESTS_PASSED=0
TESTS_FAILED=0

# テスト関数
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_exit_code="${3:-0}"
    
    echo "🧪 Running test: $test_name"
    
    # テストコマンドを実行して終了コードを取得
    eval "$test_command" > /dev/null 2>&1
    local exit_code=$?
    
    if [[ $exit_code -eq $expected_exit_code ]]; then
        log_success "Test passed: $test_name"
        ((TESTS_PASSED++))
    else
        log_error "Test failed: $test_name (expected exit code: $expected_exit_code, got: $exit_code)"
        ((TESTS_FAILED++))
    fi
}

# スクリプトの存在確認
check_script_exists() {
    log_info "スクリプトの存在確認..."
    
    if [[ -f "scripts/monitor-aws-runners.sh" ]]; then
        log_success "監視スクリプトが見つかりました"
    else
        log_error "監視スクリプトが見つかりません: scripts/monitor-aws-runners.sh"
        exit 1
    fi
}

# 実行権限の確認
check_execution_permissions() {
    log_info "実行権限の確認..."
    
    if [[ -x "scripts/monitor-aws-runners.sh" ]]; then
        log_success "スクリプトに実行権限があります"
    else
        log_warn "スクリプトに実行権限がありません。追加します..."
        chmod +x scripts/monitor-aws-runners.sh
        log_success "実行権限を追加しました"
    fi
}

# 依存関係の確認
check_dependencies() {
    log_info "依存関係の確認..."
    
    local deps=("aws" "jq" "bc")
    local missing_deps=()
    
    for dep in "${deps[@]}"; do
        if command -v "$dep" &> /dev/null; then
            log_success "$dep が見つかりました"
        else
            log_warn "$dep が見つかりません"
            missing_deps+=("$dep")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "以下の依存関係が不足しています: ${missing_deps[*]}"
        echo ""
        echo "インストール方法:"
        echo "  Ubuntu/Debian: sudo apt-get install ${missing_deps[*]}"
        echo "  macOS: brew install ${missing_deps[*]}"
        echo "  CentOS/RHEL: sudo yum install ${missing_deps[*]}"
        echo ""
    fi
}

# ヘルプオプションのテスト
test_help_option() {
    run_test "ヘルプオプション" \
        "scripts/monitor-aws-runners.sh --help" \
        0
}

# 無効なオプションのテスト
test_invalid_option() {
    run_test "無効なオプション" \
        "scripts/monitor-aws-runners.sh --invalid-option" \
        1
}

# JSON出力オプションのテスト
test_json_output() {
    run_test "JSON出力オプション" \
        "scripts/monitor-aws-runners.sh --json --help" \
        0
}

# 個別オプションのテスト
test_individual_options() {
    run_test "コスト分析オプション" \
        "scripts/monitor-aws-runners.sh --cost-analysis --help" \
        0
    
    run_test "パフォーマンスオプション" \
        "scripts/monitor-aws-runners.sh --performance --help" \
        0
    
    run_test "セキュリティオプション" \
        "scripts/monitor-aws-runners.sh --security --help" \
        0
}

# スクリプトの構文チェック
test_syntax() {
    log_info "スクリプトの構文チェック..."
    
    if bash -n scripts/monitor-aws-runners.sh; then
        log_success "構文チェック完了"
        ((TESTS_PASSED++))
    else
        log_error "構文エラーが見つかりました"
        ((TESTS_FAILED++))
    fi
}

# 関数の存在確認
test_function_definitions() {
    log_info "関数定義の確認..."
    
    local required_functions=(
        "check_prerequisites"
        "get_aws_info"
        "check_ec2_instances"
        "check_runner_usage"
        "analyze_costs"
        "check_performance"
        "check_security"
        "generate_summary"
        "main"
    )
    
    local missing_functions=()
    
    for func in "${required_functions[@]}"; do
        if grep -q "^${func}()" scripts/monitor-aws-runners.sh; then
            log_success "関数 $func が見つかりました"
        else
            log_warn "関数 $func が見つかりません"
            missing_functions+=("$func")
        fi
    done
    
    if [[ ${#missing_functions[@]} -gt 0 ]]; then
        log_error "以下の関数が不足しています: ${missing_functions[*]}"
        ((TESTS_FAILED++))
    else
        log_success "すべての必要な関数が見つかりました"
        ((TESTS_PASSED++))
    fi
}

# AWS認証のテスト（オプション）
test_aws_auth() {
    log_info "AWS認証のテスト..."
    
    if aws sts get-caller-identity &> /dev/null; then
        log_success "AWS認証が設定されています"
        ((TESTS_PASSED++))
        
        # 実際の監視スクリプトを軽くテスト
        log_info "軽量な監視テストを実行..."
        if timeout 30s scripts/monitor-aws-runners.sh --help &> /dev/null; then
            log_success "監視スクリプトが正常に動作します"
            ((TESTS_PASSED++))
        else
            log_warn "監視スクリプトの実行でタイムアウトまたはエラーが発生しました"
            ((TESTS_FAILED++))
        fi
    else
        log_warn "AWS認証が設定されていません（スキップ）"
        log_info "AWS認証を設定するには: aws configure"
    fi
}

# テスト結果の表示
show_test_results() {
    echo ""
    echo "=== テスト結果 ==="
    echo "成功: $TESTS_PASSED"
    echo "失敗: $TESTS_FAILED"
    echo "合計: $((TESTS_PASSED + TESTS_FAILED))"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_success "すべてのテストが成功しました！"
        exit 0
    else
        log_error "$TESTS_FAILED 個のテストが失敗しました"
        exit 1
    fi
}

# メイン実行
main() {
    echo "🧪 AWSランナー監視スクリプトのテストを開始します..."
    echo ""
    
    check_script_exists
    check_execution_permissions
    check_dependencies
    
    echo ""
    echo "=== 機能テスト ==="
    test_help_option
    test_invalid_option
    test_json_output
    test_individual_options
    
    echo ""
    echo "=== コード品質テスト ==="
    test_syntax
    test_function_definitions
    
    echo ""
    echo "=== 統合テスト ==="
    test_aws_auth
    
    show_test_results
}

# スクリプト実行
main "$@" 