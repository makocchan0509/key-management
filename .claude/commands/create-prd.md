---
description: プロダクト要求定義書の作成
---

# プロダクト要求定義書の作成

このコマンドは、プロジェクトのプロジェクト要求定義書(PRD)を作成します。

## 実行方法

```bash
claude
> /create-prd
```

## 実行前の確認

`docs/ideas/` ディレクトリ内のファイルを確認します。
```bash
# 確認
ls docs/ideas/

# ファイルが存在する場合
✅ docs/ideas/initial-requirements.md が見つかりました
   この内容を元にPRDを作成します

# ファイルが存在しない場合
⚠️  docs/ideas/ にファイルがありません
   対話形式でPRDを作成します
```

## 手順

### ステップ0: インプットの読み込み

1. `docs/ideas/` 内のマークダウンファイルを全て読む
2. 内容を理解し、PRD作成の参考にする

### ステップ1: プロダクト要求定義書の作成

1. **prd-writingスキル**をロード
2. `docs/ideas/`の内容を元に`docs/product-requirements.md`を作成
3. 壁打ちで出たアイデアを具体化：
   - 詳細なユーザーストーリー
   - 受け入れ条件
   - 非機能要件
4. ユーザーに確認を求め、**承認されるまで待機**

## 完了条件

- 以下の永続ドキュメントが全て作成されていること
   -  docs/product-requirements.md

完了時のメッセージ:
```
「プロダクト要求定義書が完了しました!

作成したドキュメント:
✅ docs/product-requirements.md
```