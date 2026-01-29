---
description: 開発ガイドラインの作成
---

# 開発ガイドラインの作成

このコマンドは、プロジェクトの開発ガイドラインを作成します。

## 実行方法

```bash
claude
> /create-dev-guideline
```

## 実行前の確認

以下のファイルが存在するか確認します。
-  docs/product-requirements.md
-  docs/functional-design.md
-  docs/architecture.md
-  docs/repository-structure.md 

```bash
# 確認
ls docs/

# ファイルが存在する場合
✅ ドキュメントが見つかりました
   この内容を元に開発ガイドラインを作成します

# ファイルが存在しない場合
⚠️ ドキュメントが見つかりませんでした
   対話形式で開発ガイドラインを作成します
```

## 手順

### ステップ1: 開発ガイドラインの作成
1. **development-guidelinesスキル**をロード
2. `docs/`に格納されているドキュメントを読む
3. スキルのテンプレートとガイドに従って`docs/development-guidelines.md`を作成
4. ユーザーに確認を求め、**承認されるまで待機**

## 完了条件
- 以下の永続ドキュメントが全て作成されていること

完了時のメッセージ:
```
「開発ガイドラインが完了しました!

作成したドキュメント:
✅ docs/development-guidelines.md
```