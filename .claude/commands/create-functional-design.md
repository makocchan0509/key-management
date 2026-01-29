---
description: 機能設計書の作成
---

# 機能設計書の作成

このコマンドは、プロジェクトの機能設計書を作成します。

## 実行方法

```bash
claude
> /create-functional-design
```

## 実行前の確認

以下のファイルが存在するか確認します。
-  docs/product-requirements.md

```bash
# 確認
ls docs/

# ファイルが存在する場合
✅ ドキュメントが見つかりました
   この内容を元に機能設計書を作成します

# ファイルが存在しない場合
⚠️ ドキュメントが見つかりませんでした
   対話形式で機能設計書を作成します
```

## 手順

### ステップ1: 機能設計書の作成
1. **functional-designスキル**をロード
1. `docs/product-requirements.md`を読む
3. スキルのテンプレートとガイドに従って`docs/functional-design.md`を作成
4. ユーザーに確認を求め、**承認されるまで待機**

## 完了条件
- 以下の永続ドキュメントが全て作成されていること

完了時のメッセージ:
```
「機能設計書が完了しました!

作成したドキュメント:
✅ docs/functional-design.md
```