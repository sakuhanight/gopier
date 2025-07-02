# リリース手順

## バージョニング

Gopierは[セマンティックバージョニング](https://semver.org/lang/ja/)に従います：

- **MAJOR.MINOR.PATCH** 形式（例：`v1.0.0`）
- **MAJOR**: 互換性のない変更
- **MINOR**: 後方互換性のある新機能追加
- **PATCH**: 後方互換性のあるバグ修正

## リリース手順

### 1. 変更内容の確認
```bash
git log --oneline $(git describe --tags --abbrev=0)..HEAD
```

### 2. バージョンタグの作成
```bash
# パッチバージョン（バグ修正）
git tag v1.0.1

# マイナーバージョン（新機能）
git tag v1.1.0

# メジャーバージョン（破壊的変更）
git tag v2.0.0
```

### 3. タグのプッシュ
```bash
git push origin v1.0.1
```

### 4. 自動リリース
GitHub Actionsが自動的に以下を実行します：

1. **ビルド**: 3つのOS（Linux/Windows/macOS）でバイナリをビルド
2. **アーティファクト作成**: 各OS向けの配布パッケージを作成
3. **リリース作成**: GitHub Releasesに自動でリリースを作成
4. **配布物**: バイナリ、設定ファイル例、README、LICENSEを含む

## 配布物の内容

各OS向けの配布パッケージには以下が含まれます：

- `gopier` (または `gopier.exe`)
- `config.example.yaml` (設定ファイル例)
- `README.md`
- `LICENSE`

## 対応アーキテクチャ

- **Linux**: AMD64
- **Windows**: AMD64
- **macOS**: AMD64 (Intel) / ARM64 (Apple Silicon)

## リリースノート

GitHub Actionsが自動的にリリースノートを生成します：

- コミット履歴から変更内容を自動抽出
- タグ間の変更をまとめて表示
- プルリクエストの情報も含む

## 手動リリース（緊急時）

GitHub Actionsが失敗した場合の手動リリース手順：

```bash
# ローカルでビルド
go build -ldflags="-s -w" -o gopier .

# 配布パッケージ作成
mkdir release
cp gopier release/
cp config.example.yaml release/
cp README.md release/
cp LICENSE release/
tar -czf gopier-$(git describe --tags).tar.gz -C release .
```

## 注意事項

- タグは必ず `v` プレフィックスを付ける（例：`v1.0.0`）
- リリース前にテストが通ることを確認
- 破壊的変更がある場合はREADMEを更新
- 依存関係の更新時は `go.mod` と `go.sum` を確認 