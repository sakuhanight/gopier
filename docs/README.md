# ドキュメント

このディレクトリには、gopierプロジェクトの各種ドキュメントが含まれています。

## 📚 ドキュメント一覧

### 🚀 CI環境関連

#### [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) - **推奨**
現在のCI環境の包括的なドキュメントです。以下の内容を含みます：
- ワークフロー構成と使用方法
- パフォーマンス指標と最適化
- トラブルシューティング
- メンテナンスガイド

#### [CI_REFACTORING.md](CI_REFACTORING.md)
CIリファクタリングの詳細記録です。リファクタリングの背景、改善内容、効果を記録しています。

### 🔧 開発・運用関連

#### [AWS_RUNNER_SETUP.md](AWS_RUNNER_SETUP.md)
AWSセルフホステッドランナーのセットアップガイドです。

#### [CI_OPTIMIZATION.md](CI_OPTIMIZATION.md)
CI最適化の詳細とベストプラクティスです。

#### [SECURITY_PERMISSIONS.md](SECURITY_PERMISSIONS.md)
セキュリティ権限の設定と管理について説明しています。

#### [SIGNING.md](SIGNING.md)
リリース署名の手順と設定について説明しています。

## 📖 ドキュメントの使い方

### 開発者向け
1. **CI環境の理解**: [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) から始める
2. **ローカル開発**: Makefileのターゲットを活用
3. **トラブルシューティング**: 各ドキュメントのトラブルシューティングセクションを参照

### メンテナー向け
1. **CIシステムの管理**: [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) のメンテナンスセクション
2. **リファクタリング履歴**: [CI_REFACTORING.md](CI_REFACTORING.md) で変更履歴を確認
3. **今後の改善**: 各ドキュメントの改善計画を参照

### コントリビューター向け
1. **開発環境のセットアップ**: README.md の開発・テストセクション
2. **CI環境の理解**: [CI_ENVIRONMENT.md](CI_ENVIRONMENT.md) でCIの仕組みを理解
3. **コントリビューション**: 各ドキュメントのガイドラインに従う

## 🔄 ドキュメントの更新

### 更新のタイミング
- CI環境の変更時
- 新しい機能の追加時
- トラブルシューティング情報の追加時
- ベストプラクティスの更新時

### 更新手順
1. 該当するドキュメントを更新
2. 関連するドキュメントの参照も更新
3. このインデックスファイルも必要に応じて更新
4. プルリクエストでレビューを依頼

## 📞 サポート

ドキュメントに関する質問や改善提案がある場合は、以下をご利用ください：

- [GitHub Issues](https://github.com/sakuhanight/gopier/issues)
- [GitHub Discussions](https://github.com/sakuhanight/gopier/discussions)

---

**最終更新**: 2024年12月
**バージョン**: 1.0.0 