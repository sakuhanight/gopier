#!/bin/bash

# CI ワークフロー整理スクリプト
# 古いワークフローファイルをバックアップして削除します

set -e

echo "=== CI ワークフロー整理スクリプト ==="
echo "このスクリプトは古いCIワークフローファイルを整理します。"
echo ""

# バックアップディレクトリの作成
BACKUP_DIR=".github/workflows/backup/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "バックアップディレクトリ: $BACKUP_DIR"
echo ""

# 古いワークフローファイルのリスト
OLD_WORKFLOWS=(
    "ci.yml"
    "ci-optimized.yml"
    "ci-unified.yml"
    "ci-simple.yml"
    "test.yml"
    "test-common.yml"
    "benchmark.yml"
    "windows-optimization.yml"
    "test-permissions.yml"
)

# 新しいワークフローファイルのリスト
NEW_WORKFLOWS=(
    "ci.yml"
    "test.yml"
    "benchmark.yml"
    "security.yml"
    "lint.yml"
)

echo "=== 新しいワークフローファイルの確認 ==="
for workflow in "${NEW_WORKFLOWS[@]}"; do
    if [ -f ".github/workflows/$workflow" ]; then
        echo "✓ $workflow が存在します"
    else
        echo "✗ $workflow が見つかりません"
        exit 1
    fi
done
echo ""

echo "=== 古いワークフローファイルのバックアップ ==="
for workflow in "${OLD_WORKFLOWS[@]}"; do
    if [ -f ".github/workflows/$workflow" ]; then
        echo "バックアップ中: $workflow"
        cp ".github/workflows/$workflow" "$BACKUP_DIR/"
    else
        echo "スキップ: $workflow (存在しません)"
    fi
done
echo ""

echo "=== 古いワークフローファイルの削除 ==="
read -p "古いワークフローファイルを削除しますか？ (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    for workflow in "${OLD_WORKFLOWS[@]}"; do
        if [ -f ".github/workflows/$workflow" ]; then
            echo "削除中: $workflow"
            rm ".github/workflows/$workflow"
        fi
    done
    echo "✓ 古いワークフローファイルを削除しました"
else
    echo "✗ 削除をキャンセルしました"
fi
echo ""

echo "=== 現在のワークフローファイル ==="
ls -la .github/workflows/
echo ""

echo "=== バックアップの場所 ==="
echo "バックアップディレクトリ: $BACKUP_DIR"
echo "バックアップされたファイル:"
ls -la "$BACKUP_DIR/"
echo ""

echo "=== 次のステップ ==="
echo "1. GitHub Actionsで新しいワークフローが正常に動作することを確認してください"
echo "2. 問題がない場合は、バックアップディレクトリを削除できます:"
echo "   rm -rf $BACKUP_DIR"
echo "3. 問題がある場合は、バックアップから復元できます:"
echo "   cp $BACKUP_DIR/* .github/workflows/"
echo ""

echo "=== 完了 ===" 