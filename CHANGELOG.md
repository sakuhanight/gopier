# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- 初回リリース準備
- GitHub Actionsによる自動ビルド・リリース
- マルチプラットフォーム対応（Linux/Windows/macOS Intel/macOS ARM64）
- BoltDBによる同期状態管理
- ハッシュ検証機能（MD5/SHA1/SHA256）
- 並列コピー・リトライ・ミラーモード
- 詳細なログ出力とエラーハンドリング
- フィルタリング機能（include/exclude patterns）
- 進捗表示と統計情報
- 設定ファイルサポート（YAML）
- ドライラン機能
- 検証専用モード

### Changed
- SQLiteからBoltDBにデータベースエンジンを変更（CGO依存を解消）

### Fixed
- ビルド版での応答なし問題（CGO依存のSQLiteドライバーが原因）
- 設定ファイル読み込み優先順位の修正

## [1.0.0] - 2025-01-02

### Added
- 初回リリース
- 基本的なファイル同期機能
- CLIインターフェース
- 設定ファイルサポート
- ログ機能
- テストスイート 