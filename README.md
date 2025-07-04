# Gopier

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey.svg)](https://github.com/sakuhanight/gopier)
[![Go Report Card](https://goreportcard.com/badge/github.com/sakuhanight/gopier)](https://goreportcard.com/report/github.com/sakuhanight/gopier)
[![codecov](https://codecov.io/gh/sakuhanight/gopier/branch/main/graph/badge.svg)](https://codecov.io/gh/sakuhanight/gopier)

高性能・多機能なファイル同期CLIツール

---

## 概要
GopierはGo言語で実装された、高速・堅牢なファイル同期ツールです。大容量・大量ファイルのコピーやミラーリング、ハッシュ検証、詳細なログ出力、リトライ、フィルタリング、進捗表示など、実運用に耐える多彩な機能を備えています。

- **クロスプラットフォーム**（macOS/Linux/Windows）
- **BoltDBによる同期状態管理**
- **並列コピー・リトライ・ミラーモード**
- **詳細なエラーハンドリングとログ**
- **CLI/設定ファイル両対応**

---

## インストール

### Goからビルド
```sh
git clone https://github.com/sakuhanight/gopier.git
cd gopier
go build -o gopier
```

### バイナリ配布
[GitHub Releases](https://github.com/sakuhanight/gopier/releases)から最新版をダウンロードして展開するだけで利用可能です。

各OS向けのバイナリが用意されています：
- `gopier-darwin-amd64.tar.gz` (macOS Intel)
- `gopier-darwin-arm64.tar.gz` (macOS Apple Silicon)
- `gopier-linux-amd64.tar.gz` (Linux)
- `gopier-windows-amd64.exe.zip` (Windows)

配布物には設定ファイル例（`config.example.yaml`）も含まれています。

### クロスコンパイル
Goのみで完結しているため、`GOOS`や`GOARCH`を指定してクロスビルド可能です。

---

## 設定ファイル

- 設定ファイル名: `.gopier.yaml`
- 優先順位:
  1. `--config`で明示指定したパス
  2. 実行ファイルと同じディレクトリの`.gopier.yaml`
  3. ホームディレクトリの`.gopier.yaml`

### サンプル
```yaml
source: ./src
destination: ./dst
log_file: gopier.log
workers: 8
buffer_size: 8
retry_count: 3
retry_wait: 5
include_pattern: "*.txt,*.jpg"
exclude_pattern: "*.tmp,*.bak"
recursive: true
mirror: false
dry_run: false
verbose: false
skip_newer: false
no_progress: false
preserve_mod_time: true
overwrite_existing: true
sync_mode: normal
sync_db_path: sync_state.db
include_failed: true
max_fail_count: 5
verify_only: false
verify_changed: false
verify_all: false
final_report: ""
hash_algorithm: sha256
verify_hash: true
```

### 主な項目
- `source`/`destination`: コピー元・先ディレクトリ
- `workers`: 並列ワーカー数
- `buffer_size`: バッファサイズ（MB）
- `retry_count`/`retry_wait`: リトライ回数・待機秒
- `mirror`: ミラーモード（宛先にないファイル削除）
- `dry_run`: ドライラン（実際にはコピーしない）
- `verbose`: 詳細ログ
- `sync_mode`: `normal`/`initial`/`incremental`
- `sync_db_path`: 同期状態DBファイル
- `verify_only`/`verify_changed`/`verify_all`: ハッシュ検証

---

## 使い方

### 基本コマンド
```sh
./gopier -s ./src -d ./dst
```

### 主なオプション
- `-s, --source`: コピー元ディレクトリ
- `-d, --destination`: コピー先ディレクトリ
- `-w, --workers`: 並列ワーカー数
- `-m, --mirror`: ミラーモード
- `-n, --dry-run`: ドライラン
- `-v, --verbose`: 詳細ログ
- `--verify-only`: コピーせず検証のみ
- `--verify-changed`: 同期したファイルのみ検証
- `--verify-all`: すべてのファイルを検証
- `--create-config`: デフォルト設定ファイル作成
- `--show-config`: 現在の設定値を表示

### 例
- ミラーモードで同期:
  ```sh
  ./gopier -s ./src -d ./dst --mirror
  ```
- ドライラン:
  ```sh
  ./gopier -s ./src -d ./dst --dry-run
  ```
- 詳細ログ:
  ```sh
  ./gopier -s ./src -d ./dst --verbose
  ```

---

## 同期モードとデータベース

- **同期状態はBoltDB（sync_state.db）で管理**
- `normal`（通常）/`initial`（初期同期）/`incremental`（追加同期）
- 失敗ファイルの再同期や検証履歴もDBで一元管理
- DBファイルは`--db`でパス指定可能

### データベース閲覧・管理

データベースの内容を閲覧・管理するための`db`サブコマンドが利用可能です：

```sh
# ファイル一覧の表示
./gopier db list --db sync_state.db

# 統計情報の表示
./gopier db stats --db sync_state.db

# CSV形式でエクスポート
./gopier db export --db sync_state.db --output export.csv --format csv

# JSON形式でエクスポート
./gopier db export --db sync_state.db --output export.json --format json

# 特定ステータスのファイルのみ表示
./gopier db list --db sync_state.db --status success

# サイズ順でソート（逆順）
./gopier db list --db sync_state.db --sort-by size --reverse

# 表示件数を制限
./gopier db list --db sync_state.db --limit 10

# データベースのリセット（初期同期モード用）
./gopier db reset --db sync_state.db
```

#### 利用可能なサブコマンド
- `list`: データベース内のファイル一覧を表示
- `stats`: 同期統計情報を表示
- `export`: データベースの内容をファイルにエクスポート（CSV/JSON）
- `clean`: 古いレコードを削除
- `reset`: データベースをリセット（初期同期モード用）

#### フィルタリング・ソート機能
- `--status`: 特定のステータスのファイルのみ表示
- `--sort-by`: ソート項目（path, size, mod_time, status, last_sync_time）
- `--reverse`: 逆順でソート
- `--limit`: 表示件数の制限

---

## エラーハンドリング・ログ

- すべてのエラーはloggerで一元管理
- 致命的エラーとリカバリ可能エラーを区別
- リトライ状況も詳細に記録
- `--verbose`で詳細なエラー・リトライ情報を出力
- デフォルトはファイルごとの成功・失敗・スキップのみ簡易表示

---

## パフォーマンス・並列処理

- ワーカー数（`--workers`）やバッファサイズ（`--buffer`）を調整可能
- 大量ファイル・大容量データも高速に同期
- 進捗表示（`--no-progress`で非表示化も可）

---

## トラブルシューティング

- 設定ファイルが読み込まれない場合は`--config`で明示指定
- DBファイルのパーミッションやディスク容量を確認
- ログファイルや`--verbose`出力で詳細な原因を特定
- クロスプラットフォームで動作しない場合はGoバージョンや依存パッケージを確認

---

## 開発・テスト

### テストの実行
```sh
go test ./...
```

### ベンチマークの実行
Goの標準ベンチマーク機能でパフォーマンステストが可能です。

例：copier/verifierパッケージのベンチマーク
```sh
go test -bench=. ./internal/copier
go test -bench=. ./internal/verifier
```

- `-bench=.` で全てのベンチマークが実行されます。
- 必要に応じて `-bench=関数名` で個別に実行できます。

### コントリビュート
- Issue/Pull Request歓迎
- テスト・ドキュメントの追加も大歓迎

---

## ライセンス

MIT License

---

## 作者

@sakuhanight