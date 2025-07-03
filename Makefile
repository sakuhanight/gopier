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
	go test -v ./...
	@echo "統合テスト実行中..."
	go test -v ./tests/...

# テストカバレッジ
.PHONY: test-coverage
test-coverage:
	@echo "テストカバレッジ実行中..."
	go test -v -coverprofile=coverage.out ./cmd/... ./internal/...
	@echo "カバレッジレポート生成中..."
	@if [ -f coverage.out ]; then \
		go tool cover -html=coverage.out -o coverage.html 2>/dev/null || echo "HTMLレポート生成をスキップしました"; \
		echo "カバレッジレポート: coverage.html"; \
		go tool cover -func=coverage.out || echo "カバレッジ関数レポート生成をスキップしました"; \
	else \
		echo "カバレッジファイルが見つかりません"; \
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
	@echo "  test-coverage- テストカバレッジ"
	@echo "  tidy         - 依存関係の整理"
	@echo "  clean        - クリーンアップ"
	@echo "  install      - インストール"
	@echo "  uninstall    - アンインストール"
	@echo "  help         - このヘルプを表示" 