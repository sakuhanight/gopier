#!/bin/bash

# CI ワークフロー整理スクリプト
# 古いワークフローファイルをバックアップして整理します

set -e

echo "=== CI ワークフロー整理スクリプト ==="

# バックアップディレクトリの作成
BACKUP_DIR=".github/workflows/backup/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "バックアップディレクトリ: $BACKUP_DIR"

# 古いワークフローファイルのリスト
OLD_WORKFLOWS=(
    "ci.yml"
    "ci-optimized.yml"
    "test.yml"
    "windows-optimization.yml"
    "benchmark.yml"
    "test-permissions.yml"
)

# 新しいワークフローファイルのリスト
NEW_WORKFLOWS=(
    "ci-unified.yml"
    "ci-simple.yml"
    "test-common.yml"
)

echo "=== 古いワークフローファイルのバックアップ ==="

# 古いワークフローファイルをバックアップ
for workflow in "${OLD_WORKFLOWS[@]}"; do
    if [ -f ".github/workflows/$workflow" ]; then
        echo "バックアップ中: $workflow"
        cp ".github/workflows/$workflow" "$BACKUP_DIR/"
    else
        echo "ファイルが見つかりません: $workflow"
    fi
done

echo "=== 新しいワークフローファイルの確認 ==="

# 新しいワークフローファイルの存在確認
for workflow in "${NEW_WORKFLOWS[@]}"; do
    if [ -f ".github/workflows/$workflow" ]; then
        echo "✓ 存在: $workflow"
    else
        echo "✗ 見つかりません: $workflow"
    fi
done

echo ""
echo "=== 整理オプション ==="
echo "1. 古いワークフローファイルを削除"
echo "2. 古いワークフローファイルを無効化（.disabled 拡張子を追加）"
echo "3. 何もしない（バックアップのみ）"
echo ""

read -p "選択してください (1-3): " choice

case $choice in
    1)
        echo "=== 古いワークフローファイルを削除中 ==="
        for workflow in "${OLD_WORKFLOWS[@]}"; do
            if [ -f ".github/workflows/$workflow" ]; then
                echo "削除中: $workflow"
                rm ".github/workflows/$workflow"
            fi
        done
        echo "削除完了"
        ;;
    2)
        echo "=== 古いワークフローファイルを無効化中 ==="
        for workflow in "${OLD_WORKFLOWS[@]}"; do
            if [ -f ".github/workflows/$workflow" ]; then
                echo "無効化中: $workflow"
                mv ".github/workflows/$workflow" ".github/workflows/$workflow.disabled"
            fi
        done
        echo "無効化完了"
        ;;
    3)
        echo "バックアップのみ実行しました"
        ;;
    *)
        echo "無効な選択です"
        exit 1
        ;;
esac

echo ""
echo "=== 推奨設定 ==="
echo "GitHub Actionsで以下のワークフローを有効にしてください："
echo ""
echo "推奨（高速）: ci-simple.yml"
echo "包括的: ci-unified.yml"
echo ""
echo "バックアップは以下に保存されました: $BACKUP_DIR"
echo ""
echo "=== 完了 ===" 