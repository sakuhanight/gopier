SHELL=/bin/bash
# Gopier Makefile for cross-platform
# 使用方法: make <target>

# 変数定義
BINARY_NAME=gopier
BUILD_DIR=build

# OS判定
ifeq ($(OS),Windows_NT)
    BINARY_NAME=gopier.exe
    RM=del /Q
    RMDIR=rmdir /S /Q
    MKDIR=mkdir
    CP=copy
    SHELL=cmd
else
    RM=rm -f
    RMDIR=rm -rf
    MKDIR=mkdir -p
    CP=cp
    SHELL=bash
endif

# デフォルトターゲット
.PHONY: all
all: clean build

# ビルド
.PHONY: build
build:
	@echo "ビルド中..."
	@VERSION=$${VERSION:-$$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}; \
	BUILD_TIME=$$(date '+%Y-%m-%d_%H-%M-%S'); \
	LDFLAGS="-X github.com/sakuhanight/gopier/cmd.Version=$$VERSION -X github.com/sakuhanight/gopier/cmd.BuildTime=$$BUILD_TIME"; \
	echo "Version: $$VERSION"; \
	echo "BuildTime: $$BUILD_TIME"; \
	set -x; \
	go build -ldflags "$$LDFLAGS" -o $(BINARY_NAME)
	@echo "ビルド完了: $(BINARY_NAME)"

# CI用通常ビルド
.PHONY: build-ci
build-ci:
	@echo "CI用通常ビルド中..."
	@VERSION=$${VERSION:-"ci-build"}; \
	BUILD_TIME=$$(date '+%Y-%m-%d_%H-%M-%S'); \
	LDFLAGS="-X github.com/sakuhanight/gopier/cmd.Version=$$VERSION -X github.com/sakuhanight/gopier/cmd.BuildTime=$$BUILD_TIME"; \
	echo "Version: $$VERSION"; \
	echo "BuildTime: $$BUILD_TIME"; \
	set -x; \
	go build -ldflags "$$LDFLAGS" -o $(BINARY_NAME)
	@echo "CI用通常ビルド完了: $(BINARY_NAME)"

# リリースビルド（最適化）
.PHONY: release
release:
	@echo "リリースビルド中..."
	@VERSION=$${VERSION:-$$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}; \
	BUILD_TIME=$$(date '+%Y-%m-%d_%H-%M-%S'); \
	LDFLAGS="-s -w -X github.com/sakuhanight/gopier/cmd.Version=$$VERSION -X github.com/sakuhanight/gopier/cmd.BuildTime=$$BUILD_TIME"; \
	echo "Version: $$VERSION"; \
	echo "BuildTime: $$BUILD_TIME"; \
	if [ "$$GOOS" = "windows" ]; then \
		go build -ldflags "$$LDFLAGS" -o gopier.exe; \
		echo "リリースビルド完了: gopier.exe"; \
	else \
		go build -ldflags "$$LDFLAGS" -o gopier; \
		echo "リリースビルド完了: gopier"; \
	fi

# クロスプラットフォームビルド
.PHONY: cross-build
cross-build:
	@echo "クロスプラットフォームビルド中..."
	@VERSION=$${VERSION:-$$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}; \
	BUILD_TIME=$$(date '+%Y-%m-%d_%H-%M-%S'); \
	LDFLAGS="-X github.com/sakuhanight/gopier/cmd.Version=$$VERSION -X github.com/sakuhanight/gopier/cmd.BuildTime=$$BUILD_TIME"; \
	echo "Version: $$VERSION"; \
	echo "BuildTime: $$BUILD_TIME"; \
	$(MKDIR) $(BUILD_DIR); \
	echo "Windows AMD64..."; \
	GOOS=windows GOARCH=amd64 go build -ldflags "$$LDFLAGS" -o $(BUILD_DIR)/gopier-windows-amd64.exe; \
	echo "Linux AMD64..."; \
	GOOS=linux GOARCH=amd64 go build -ldflags "$$LDFLAGS" -o $(BUILD_DIR)/gopier-linux-amd64; \
	echo "macOS AMD64..."; \
	GOOS=darwin GOARCH=amd64 go build -ldflags "$$LDFLAGS" -o $(BUILD_DIR)/gopier-darwin-amd64; \
	echo "macOS ARM64..."; \
	GOOS=darwin GOARCH=arm64 go build -ldflags "$$LDFLAGS" -o $(BUILD_DIR)/gopier-darwin-arm64
	@echo "クロスプラットフォームビルド完了"

# テスト実行
.PHONY: test
test:
	@echo "テスト実行中..."
	@go test -v ./... && echo "通常テスト成功" || (echo "通常テスト失敗"; exit 1)
	@echo "統合テスト実行中..."
	@go test -v ./tests/... && echo "統合テスト成功" || (echo "統合テスト失敗"; exit 1)

# 短時間テスト（管理者権限不要）
.PHONY: test-short
test-short:
	@echo "短時間テスト実行中（管理者権限不要）..."
	@go test -v -short ./internal/permissions/... && echo "短時間テスト成功" || (echo "短時間テスト失敗"; exit 1)

# 管理者権限テスト
.PHONY: test-admin
test-admin:
	@echo "管理者権限テスト実行中..."
	@if [ "$$(go env GOOS)" = "windows" ]; then \
		echo "Windows環境で管理者権限テストを実行します..."; \
		go test -v -run "WithAdmin" ./internal/permissions/... && echo "管理者権限テスト成功" || (echo "管理者権限テスト失敗"; exit 1); \
	else \
		echo "管理者権限テストはWindowsでのみ実行可能です"; \
		exit 0; \
	fi

# 権限関連テスト（管理者権限が必要な場合がある）
.PHONY: test-permissions
test-permissions:
	@echo "権限関連テスト実行中..."
	@go test -v ./internal/permissions/... && echo "権限関連テスト成功" || (echo "権限関連テスト失敗"; exit 1)

# CI用テスト（並列実行）
.PHONY: test-ci
test-ci:
	@echo "CI用テスト実行中..."
	@go test -v -parallel=4 -timeout=10m ./cmd/... ./internal/... && echo "ユニットテスト成功" || (echo "ユニットテスト失敗"; exit 1)
	@go test -v -parallel=2 -timeout=10m ./tests/... && echo "統合テスト成功" || (echo "統合テスト失敗"; exit 1)

# 高速テスト（タイムアウト短縮）
.PHONY: test-fast
test-fast:
	@echo "高速テスト実行中..."
	@go test -v -timeout=5m -parallel=4 ./cmd/... ./internal/... && echo "高速テスト成功" || (echo "高速テスト失敗"; exit 1)

# テスト実行（タイムアウト付き）
.PHONY: test-timeout
test-timeout:
	@echo "テスト実行中（タイムアウト60秒）..."
	@if command -v gtimeout >/dev/null 2>&1; then \
		gtimeout 60s go test -v ./...; \
		echo "統合テスト実行中（タイムアウト60秒）..."; \
		gtimeout 60s go test -v ./tests/...; \
	elif command -v timeout >/dev/null 2>&1; then \
		timeout 60s go test -v ./...; \
		echo "統合テスト実行中（タイムアウト60秒）..."; \
		timeout 60s go test -v ./tests/...; \
	else \
		echo "タイムアウトコマンドが見つかりません。通常のテストを実行します。"; \
		go test -v ./...; \
		echo "統合テスト実行中..."; \
		go test -v ./tests/...; \
	fi

# テストカバレッジ
.PHONY: test-coverage
test-coverage:
	@echo "テストカバレッジ実行中..."
	@if COVERAGE=1 go test -v -coverprofile=coverage.out ./cmd/... ./internal/...; then \
		echo "テスト成功。カバレッジレポート生成中..."; \
		if [ -f coverage.out ]; then \
			go tool cover -html=coverage.out -o coverage.html 2>/dev/null || echo "HTMLレポート生成をスキップしました"; \
			echo "カバレッジレポート: coverage.html"; \
			go tool cover -func=coverage.out || echo "カバレッジ関数レポート生成をスキップしました"; \
		else \
			echo "カバレッジファイルが見つかりません"; \
		fi; \
	else \
		echo "テストが失敗しました。カバレッジレポートは生成されません。"; \
		exit 1; \
	fi

# 依存関係の整理
.PHONY: tidy
tidy:
	@echo "依存関係を整理中..."
	go mod tidy
	go mod verify

# クリーンアップ
.PHONY: clean
clean:
	@echo "クリーンアップ中..."
	$(RM) $(BINARY_NAME)
	$(RMDIR) $(BUILD_DIR)
	$(RM) coverage.out
	$(RM) coverage.html
	@echo "クリーンアップ完了"

# インストール
.PHONY: install
install: build
	@echo "インストール中..."
	$(CP) $(BINARY_NAME) $(GOPATH)/bin/
	@echo "インストール完了"

# アンインストール
.PHONY: uninstall
uninstall:
	@echo "アンインストール中..."
	$(RM) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "アンインストール完了"

# ヘルプ
.PHONY: help
help:
	@echo "利用可能なターゲット:"
	@echo "  build        - 通常ビルド"
	@echo "  release      - リリースビルド（最適化）"
	@echo "  cross-build  - クロスプラットフォームビルド"
	@echo "  test         - テスト実行"
	@echo "  test-short   - 短時間テスト（管理者権限不要）"
	@echo "  test-admin   - 管理者権限テスト（Windowsのみ）"
	@echo "  test-permissions - 権限関連テスト"
	@echo "  test-ci      - CI用テスト（並列実行）"
	@echo "  test-fast    - 高速テスト（タイムアウト短縮）"
	@echo "  test-coverage- テストカバレッジ"
	@echo "  tidy         - 依存関係の整理"
	@echo "  clean        - クリーンアップ"
	@echo "  install      - インストール"
	@echo "  uninstall    - アンインストール"
	@echo "  help         - このヘルプを表示" 