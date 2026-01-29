# 設計

## 変更範囲

### 1. ドキュメント更新

#### docs/architecture.md
- **変更箇所**: テクノロジースタック > 言語・ランタイム
- **変更内容**:
  - 変更前: `Go | 1.23.x`
  - 変更後: `Go | 1.25.x`

### 2. ビルド設定の更新

#### key-management-service/go.mod
- **変更箇所**: go ディレクティブ（3行目）
- **変更内容**:
  - 変更前: `go 1.22.12`
  - 変更後: `go 1.25.0`
- **影響**: モジュール互換性の基準バージョンが変更される

#### key-management-service/Dockerfile
- **変更箇所**: ビルドステージのベースイメージ（2行目）
- **変更内容**:
  - 変更前: `FROM golang:1.23-alpine AS builder`
  - 変更後: `FROM golang:1.25-alpine AS builder`
- **影響**: コンテナビルド時のGoコンパイラバージョンが変更される

### 3. ソースコードへの影響

#### Go 1.25の主な新機能・変更点（確認事項）
1. **新しい言語機能**: 現時点で利用している機能に影響なし
2. **標準ライブラリの変更**:
   - `log/slog`、`net/http`などの使用箇所は互換性あり
3. **非推奨化された機能**: 使用していない

#### 依存ライブラリの互換性
- `chi v5.1.0`: Go 1.21+ 対応（問題なし）
- `gorm v1.25.x`: Go 1.18+ 対応（問題なし）
- `cloud.google.com/go/*`: Go 1.21+ 対応（問題なし）
- `opentelemetry-go v1.29.x`: Go 1.21+ 対応（問題なし）
- `cobra v1.8.x`: Go 1.16+ 対応（問題なし）

## 実装アプローチ

### ステップ1: ドキュメント更新
1. `docs/architecture.md` を編集
2. Goバージョンの記載を1.25.xに変更

### ステップ2: ビルド設定更新
1. `key-management-service/go.mod` の go ディレクティブを更新
2. `key-management-service/Dockerfile` のベースイメージを更新

### ステップ3: ビルド検証
1. `go mod tidy` で依存関係を整理
2. `go build` でビルドエラーがないことを確認
3. Dockerイメージのビルド確認

### ステップ4: テスト実行
1. ユニットテストの実行: `go test ./...`
2. テストが引き続き正常に動作することを確認

## リスクと軽減策

### リスク1: 依存ライブラリの非互換性
- **可能性**: 低（すべてのライブラリはGo 1.21+対応）
- **軽減策**: `go mod tidy`で依存関係を再解決し、ビルドテストを実行

### リスク2: ビルドエラー
- **可能性**: 低（マイナーバージョンアップのため）
- **軽減策**: ローカル環境で事前にビルド検証

### リスク3: ランタイムの挙動変更
- **可能性**: 極めて低
- **軽減策**: テストの実行で既存機能の動作を保証

## 検証計画

1. **ビルド検証**
   ```bash
   cd key-management-service
   go mod tidy
   go build ./cmd/server
   go build ./cmd/keyctl
   ```

2. **テスト実行**
   ```bash
   go test ./...
   ```

3. **Dockerビルド検証**
   ```bash
   docker build -t key-management-service:test .
   ```

## ロールバック計画

変更が問題を引き起こした場合:
1. `go.mod`、`Dockerfile`、`docs/architecture.md` をGitで元のバージョンに戻す
2. `git restore` コマンドで対象ファイルを復元
