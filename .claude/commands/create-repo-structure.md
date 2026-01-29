---
description: リポジトリ構造定義書の作成
---

# リポジトリ構造定義書の作成

このコマンドは、プロジェクトのリポジトリ構造定義書を作成します。

## 実行方法

```bash
claude
> /create-repo-structure
```

## 実行前の確認

以下のファイルが存在するか確認します。
-  docs/product-requirements.md
-  docs/functional-design.md
-  docs/architecture.md

```bash
# 確認
ls docs/

# ファイルが存在する場合
✅ ドキュメントが見つかりました
   この内容を元にリポジトリ構造定義書を作成します

# ファイルが存在しない場合
⚠️ ドキュメントが見つかりませんでした
   対話形式でリポジトリ構造定義書を作成します
```

## 手順

### ステップ1: リポジトリ構造定義書の作成
1. **repository-structureスキル**をロード
2. `docs/`に格納されているドキュメントを読む
3. スキルのテンプレートとガイドに従って`docs/repository-structure.md`を作成
4. ユーザーに確認を求め、**承認されるまで待機**

## 完了条件
- 以下の永続ドキュメントが全て作成されていること

完了時のメッセージ:
```
「リポジトリ構造定義書が完了しました!

作成したドキュメント:
✅ docs/repository-structure.md
```