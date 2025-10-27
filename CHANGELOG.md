# Changelog

本プロジェクトの全ての主な変更点を記載します。

## [Unreleased]
### Added
- internal/copierパッケージの包括的なテストケース追加
  - コンテキストキャンセル処理のテスト
  - 検証モード（ModeVerify, ModeCopyAndVerify）のテスト
  - リトライロジックのテスト
  - 上書き無効時のスキップ処理テスト
  - 並行アクセス処理のテスト
  - 複雑なフィルタリング機能のテスト
  - ハッシュ検証エラーのテスト
  - 非再帰モードのテスト
  - 進捗チャンネル処理のテスト
  - ファイルシステムエラーハンドリングのテスト
### Changed
- internal/copierのテストカバレッジを49.7%から53.2%に向上
- テストの安定性向上（タイミング依存のテストケースを柔軟に判定）
- エラーハンドリングとエッジケースのテスト網羅性を大幅改善

## [v0.9.4] - 2025-07-02
### Fixed
- Makefileのビルドコマンドでバージョン情報埋め込み時のエラーを修正
- ビルド時のLDFLAGS変数展開問題を解決し、--versionオプションで正しくバージョン表示
- GitHub Actionsのrelease.ymlでタグからバージョンを自動取得しビルド時間を埋め込む機能を改善
### Changed
- CI/CDワークフローをMakefileを使用するように修正
- GitHub Actionsのci.ymlとrelease.ymlでmakeコマンドを使用
- Makefileで環境変数VERSIONをサポートし、CI/CDでのバージョン制御を改善
- ビルドプロセスの一貫性と保守性を向上

## [v0.9.3] - 2025-07-02
### Fixed
- ベンチマーク実行時のチャンネル操作エラーを修正（copier/verifierパッケージ）
- 進捗報告のチャンネル送信を安全にし、二重closeやsend on closed channelを防止

## [v0.9.2] - 2025-07-02
### Changed
- Windowsリリースアセット作成をPowerShell構文に修正し、OSごとに分割
- cmd, main: テストカバレッジ向上のためテスト追加・修正
- Windowsビルド時のGOARCH指定をPowerShell流に修正

## [v0.9.1] - 2025-07-02
### Added
- データベース閲覧機能（list, stats, export, clean, resetサブコマンド）
- .gitignoreにテストファイルと設定ファイルを追加
- copier/verifierパッケージにベンチマーク関数を追加
### Changed
- Windowsビルド時のbash if文エラーをOSごとに分割して修正
- .gitignoreの更新

## [v0.9.0] - 2025-07-02
### Changed
- キャッシュを一時的に無効化し、File existsエラーを完全回避
- キャッシュrestore-keysを具体化し競合・壊れキャッシュ回避
- tarアーカイブ作成前に既存ファイルを削除し、File existsエラーを回避
- Goモジュールキャッシュ設定を追加（actions/cache@v4）
- Windows環境でのビルドエラーを修正
- Dependabotの導入

## [v0.1.0] - 2025-07-02
### Added
- 初回リリース
- 基本的なファイル同期機能
- CLIインターフェース
- 設定ファイルサポート
- ログ機能
- テストスイート 