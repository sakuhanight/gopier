# ドキュメント

このディレクトリには、Gopierプロジェクトの各種ドキュメントが含まれています。

## 📚 ドキュメント一覧

### 🚀 開発環境関連

#### [DEVELOPMENT_SETUP.md](DEVELOPMENT_SETUP.md) - **推奨**
開発環境セットアップの包括的なガイドです。以下の内容を含みます：
- クイックスタートガイド
- PowerShellスクリプトの構成と使用方法
- 設定ファイルのカスタマイズ
- トラブルシューティング
- パフォーマンス最適化
- CI/CD統合

### 🔧 CI環境関連

#### [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) - **推奨**
現在のCI環境の包括的なドキュメントです。以下の内容を含みます：
- ワークフロー構成と使用方法
- PowerShellスクリプト統合
- パフォーマンス指標と最適化
- トラブルシューティング
- メンテナンスガイド

#### [CI_OPTIMIZATION.md](CI_OPTIMIZATION.md)
CI最適化の詳細とベストプラクティスです。以下の内容を含みます：
- 最適化の背景と解決策
- PowerShellスクリプト統合
- メモリ最適化設定
- パフォーマンス改善効果
- 今後の改善計画

### 🔒 セキュリティ・運用関連

#### [SECURITY_PERMISSIONS.md](SECURITY_PERMISSIONS.md)
セキュリティ権限の設定と管理について説明しています。

#### [SIGNING.md](SIGNING.md)
リリース署名の手順と設定について説明しています。

## 📖 ドキュメントの使い方

### 開発者向け
1. **開発環境のセットアップ**: [DEVELOPMENT_SETUP.md](DEVELOPMENT_SETUP.md) から始める
2. **CI環境の理解**: [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) でCIの仕組みを理解
3. **ローカル開発**: PowerShellスクリプトを活用した効率的な開発
4. **トラブルシューティング**: 各ドキュメントのトラブルシューティングセクションを参照

### メンテナー向け
1. **CIシステムの管理**: [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) のメンテナンスセクション
2. **最適化の実施**: [CI_OPTIMIZATION.md](CI_OPTIMIZATION.md) で最適化手法を確認
3. **今後の改善**: 各ドキュメントの改善計画を参照

### コントリビューター向け
1. **開発環境のセットアップ**: [DEVELOPMENT_SETUP.md](DEVELOPMENT_SETUP.md) のクイックスタート
2. **CI環境の理解**: [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) でCIの仕組みを理解
3. **コントリビューション**: 各ドキュメントのガイドラインに従う

## 🔄 ドキュメントの更新

### 更新のタイミング
- 開発環境の変更時
- CI環境の変更時
- 新しい機能の追加時
- トラブルシューティング情報の追加時
- ベストプラクティスの更新時

### 更新手順
1. 該当するドキュメントを更新
2. 関連するドキュメントの参照も更新
3. このインデックスファイルも必要に応じて更新
4. プルリクエストでレビューを依頼

## 🛠️ 主要な改善点

### PowerShellスクリプトのリファクタリング
- **共通モジュール**: `scripts/common/` に共通機能を分離
- **設定の一元化**: JSONファイルによる設定管理
- **ログ機能の統一**: 色付きログとファイル出力
- **エラーハンドリング**: 統一されたエラー処理

### CI環境の最適化
- **実行時間**: 30-50%短縮
- **リソース使用量**: Windows環境で40%削減
- **保守性**: PowerShellスクリプト統合により大幅改善
- **設定管理**: 設定ファイルによる動的調整

### 開発ワークフローの改善
- **効率的なテスト**: 短時間テストと包括的テストの使い分け
- **クロスプラットフォーム**: 複数プラットフォームでのビルド
- **管理者権限テスト**: Windows権限テストの自動化
- **カバレッジ測定**: 自動化されたカバレッジレポート

## 📞 サポート

ドキュメントに関する質問や改善提案がある場合は、以下をご利用ください：

- [GitHub Issues](https://github.com/sakuhanight/gopier/issues)
- [GitHub Discussions](https://github.com/sakuhanight/gopier/discussions)

## 🔗 関連リンク

- [プロジェクトREADME](../README.md)
- [PowerShellスクリプトドキュメント](../scripts/README.md)
- [GitHub Actionsワークフロー](../.github/workflows/)

---

**最終更新**: 2025/07/06