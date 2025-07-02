# Gopier Makefile for Windows
# 使用方法: make <target>

# 変数定義
BINARY_NAME=gopier.exe
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>nul || echo "dev")
BUILD_TIME=$(shell powershell -Command "Get-Date -Format 'yyyy-MM-dd HH:mm:ss'")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# デフォルトターゲット
.PHONY: all
all: clean build

# ビルド
.PHONY: build
build:
	@echo "ビルド中..."
	go build $(LDFLAGS) -o $(BINARY_NAME)
	@echo "ビルド完了: $(BINARY_NAME)"

# リリースビルド（最適化）
.PHONY: release
release:
	@echo "リリースビルド中..."
	go build $(LDFLAGS) -ldflags "-s -w" -o $(BINARY_NAME)
	@echo "リリースビルド完了: $(BINARY_NAME)"

# クロスプラットフォームビルド
.PHONY: cross-build
cross-build:
	@echo "クロスプラットフォームビルド中..."
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	
	@echo "Windows AMD64..."
	set GOOS=windows&& set GOARCH=amd64&& go build $(LDFLAGS) -o $(BUILD_DIR)/gopier-windows-amd64.exe
	
	@echo "Linux AMD64..."
	set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o $(BUILD_DIR)/gopier-linux-amd64
	
	@echo "macOS AMD64..."
	set GOOS=darwin&& set GOARCH=amd64&& go build $(LDFLAGS) -o $(BUILD_DIR)/gopier-darwin-amd64
	
	@echo "macOS ARM64..."
	set GOOS=darwin&& set GOARCH=arm64&& go build $(LDFLAGS) -o $(BUILD_DIR)/gopier-darwin-arm64
	
	@echo "クロスプラットフォームビルド完了"

# テスト実行
.PHONY: test
test:
	@echo "テスト実行中..."
	go test -v ./...

# テストカバレッジ
.PHONY: test-coverage
test-coverage:
	@echo "テストカバレッジ実行中..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポート: coverage.html"

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
	@if exist $(BINARY_NAME) del $(BINARY_NAME)
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)
	@if exist coverage.out del coverage.out
	@if exist coverage.html del coverage.html
	@echo "クリーンアップ完了"

# インストール
.PHONY: install
install: build
	@echo "インストール中..."
	copy $(BINARY_NAME) "%GOPATH%\bin\"
	@echo "インストール完了"

# アンインストール
.PHONY: uninstall
uninstall:
	@echo "アンインストール中..."
	@if exist "%GOPATH%\bin\$(BINARY_NAME)" del "%GOPATH%\bin\$(BINARY_NAME)"
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