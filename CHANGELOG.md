# Changelog

本プロジェクトの全ての主な変更点を記載します。

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