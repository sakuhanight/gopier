# GPG署名の設定

このドキュメントでは、リリース成果物にGPG署名を追加するための設定手順を説明します。

## 前提条件

- GPGがインストールされていること
- GitHubリポジトリの管理者権限があること

## 手順

### 1. GPGキーの生成

```bash
# GPGキーを生成
gpg --full-generate-key

# キーIDを確認
gpg --list-secret-keys --keyid-format LONG
```

### 2. 秘密鍵のエクスポート

```bash
# 秘密鍵をエクスポート（ASCII形式）
gpg --export-secret-keys --armor YOUR_KEY_ID > private_key.asc

# 公開鍵をエクスポート
gpg --export --armor YOUR_KEY_ID > public_key.asc
```

### 3. GitHub Secretsの設定

GitHubリポジトリのSettings > Secrets and variables > Actionsで以下のシークレットを設定：

- `GPG_PRIVATE_KEY`: 秘密鍵の内容（private_key.ascの内容）
- `GPG_PASSPHRASE`: GPGキーのパスフレーズ
- `GPG_KEY_ID`: GPGキーID（例：ABC123DEF456）

### 4. 公開鍵の共有

公開鍵（public_key.asc）をリポジトリにコミットするか、リリースページに添付して、ユーザーが署名を検証できるようにします。

## 署名の検証

ユーザーは以下のコマンドで署名を検証できます：

```bash
# 公開鍵をインポート
gpg --import public_key.asc

# 署名を検証
gpg --verify file.tar.gz.asc file.tar.gz
```

## 注意事項

- 秘密鍵は絶対にリポジトリにコミットしないでください
- パスフレーズは安全に管理してください
- キーを紛失した場合は、新しいキーを生成して再設定してください 

**最終更新**: 2025/07/06