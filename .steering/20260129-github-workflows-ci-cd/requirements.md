# 要求内容

## 概要

GitHub Actions workflowを作成し、テストコードの実行、コンテナイメージのビルド、Google Cloud Artifact Registryへのイメージプッシュを自動化する。

## 背景

key-management-serviceのCI/CDパイプラインを構築することで、以下を実現する:
- コードの品質を継続的に検証する
- ビルドとデプロイの自動化により開発効率を向上させる
- Artifact Registryにイメージを保存し、Cloud Runへのデプロイ準備を整える

## 実装対象の機能

### 1. CI（継続的インテグレーション）ワークフロー
- プルリクエストおよびプッシュ時に自動的にテストを実行
- コードフォーマットチェック（gofmt）
- 静的解析（go vet）
- リント（golangci-lint）
- ユニットテストの実行

### 2. CD（継続的デプロイ）ワークフロー
- Dockerイメージのビルド
- Google Cloud Artifact Registryへのイメージプッシュ
- mainブランチへのマージ時またはタグ作成時に実行

### 3. Google Cloudインフラストラクチャ（Terraform）
- Artifact Registryリポジトリの作成
- Workload Identity Poolの作成
- Workload Identity Provider（GitHub用）の設定
- サービスアカウントの作成と権限付与

## 受け入れ条件

### CIワークフロー
- [ ] プルリクエスト作成時にCIが自動実行される
- [ ] プッシュ時（main, feature/*, fix/*ブランチ）にCIが自動実行される
- [ ] gofmt/goimportsによるフォーマットチェックが実行される
- [ ] go vetによる静的解析が実行される
- [ ] golangci-lintによるリントチェックが実行される
- [ ] go testによるユニットテストが実行される
- [ ] ビルド確認（go build）が実行される

### CDワークフロー
- [ ] mainブランチへのプッシュ時にCDが自動実行される
- [ ] タグ（v*形式）作成時にCDが自動実行される
- [ ] Dockerイメージが正常にビルドされる
- [ ] Google CloudへのWorkload Identity認証が設定される
- [ ] Artifact Registryにイメージがプッシュされる
- [ ] タグ名に基づいたイメージタグが付与される

### Terraformインフラストラクチャ
- [ ] Artifact Registryリポジトリが作成される
- [ ] Workload Identity Poolが作成される
- [ ] GitHub用Workload Identity Providerが設定される
- [ ] CI/CD用サービスアカウントが作成される
- [ ] サービスアカウントにArtifact Registry書き込み権限が付与される
- [ ] Workload IdentityとサービスアカウントのIAMバインディングが設定される
- [ ] `terraform plan` が正常に実行できる
- [ ] `terraform apply` でリソースが作成できる

## 成功指標

- CI/CDパイプラインが正常に動作し、手動介入なしで自動実行される
- テスト失敗時にマージがブロックされる
- mainブランチのコードが常にビルド可能な状態に保たれる

## スコープ外

以下はこのフェーズでは実装しません:

- テストコードの新規実装（既存のテストコードを使用）
- Cloud Runへの自動デプロイ
- Cloud SQL、Cloud KMSとの統合テスト
- 本番環境用のシークレット管理

## 参照ドキュメント

- `docs/product-requirements.md` - プロダクト要求定義書
- `docs/functional-design.md` - 機能設計書
- `docs/architecture.md` - アーキテクチャ設計書
- `docs/development-guidelines.md` - 開発ガイドライン（CI/CDパイプラインのサンプル記載）
