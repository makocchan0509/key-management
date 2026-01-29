# 設計書

## アーキテクチャ概要

GitHub Actionsを使用したCI/CDパイプラインを構築する。CIとCDを分離したワークフローとして実装し、それぞれの責務を明確にする。

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub Actions                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────┐    ┌──────────────────────────┐   │
│  │     CI Workflow      │    │      CD Workflow         │   │
│  │  (.github/workflows/ │    │  (.github/workflows/     │   │
│  │      ci.yml)         │    │      cd.yml)             │   │
│  ├──────────────────────┤    ├──────────────────────────┤   │
│  │ Trigger:             │    │ Trigger:                 │   │
│  │ - push (main,        │    │ - push (main)            │   │
│  │   feature/*, fix/*)  │    │ - tag (v*)               │   │
│  │ - pull_request       │    │                          │   │
│  ├──────────────────────┤    ├──────────────────────────┤   │
│  │ Jobs:                │    │ Jobs:                    │   │
│  │ 1. Format check      │    │ 1. Build Docker image    │   │
│  │ 2. Vet               │    │ 2. Push to Artifact      │   │
│  │ 3. Lint              │    │    Registry              │   │
│  │ 4. Test              │    │                          │   │
│  │ 5. Build             │    │                          │   │
│  └──────────────────────┘    └──────────────────────────┘   │
│                                       │                      │
└───────────────────────────────────────┼──────────────────────┘
                                        │
                                        ▼
                    ┌─────────────────────────────────────┐
                    │   Google Cloud Artifact Registry    │
                    │   (asia-northeast1-docker.pkg.dev)  │
                    └─────────────────────────────────────┘
```

## コンポーネント設計

### 1. CIワークフロー（ci.yml）

**責務**:
- コード品質の検証（フォーマット、lint、静的解析）
- テストの実行
- ビルドの確認

**実装の要点**:
- `key-management-service/`ディレクトリ内で実行
- Go 1.22.x（go.modに合わせる）を使用
- キャッシュを活用してビルド時間を短縮
- テストはCGO_ENABLED=0で実行（go.modの設定に合わせる）

**トリガー条件**:
```yaml
on:
  push:
    branches: [main, 'feature/**', 'fix/**']
    paths:
      - 'key-management-service/**'
  pull_request:
    branches: [main]
    paths:
      - 'key-management-service/**'
```

### 2. CDワークフロー（cd.yml）

**責務**:
- Dockerイメージのビルド
- Artifact Registryへのプッシュ

**実装の要点**:
- Workload Identity連携によるキーレス認証
- マルチプラットフォームビルド対応（linux/amd64）
- イメージタグ戦略:
  - mainブランチ: `latest` + コミットSHA
  - タグプッシュ: タグ名（例: `v1.0.0`）

**トリガー条件**:
```yaml
on:
  push:
    branches: [main]
    paths:
      - 'key-management-service/**'
  push:
    tags: ['v*']
```

## データフロー

### CI実行フロー
```
1. コードがプッシュまたはPR作成される
2. GitHub Actionsがトリガーされる
3. Go環境をセットアップ
4. 依存関係をキャッシュから復元/ダウンロード
5. gofmt/goimportsでフォーマットチェック
6. go vetで静的解析
7. golangci-lintでlint
8. go testでテスト実行
9. go buildでビルド確認
10. 結果をGitHubに報告
```

### CD実行フロー
```
1. mainブランチへのプッシュまたはタグ作成
2. GitHub Actionsがトリガーされる
3. Google Cloudへの認証（Workload Identity）
4. Docker Buildxでイメージをビルド
5. Artifact Registryにプッシュ
6. ビルド完了を報告
```

## エラーハンドリング戦略

### CIワークフロー
- 各ステップで失敗した場合は即座にワークフローを停止
- エラー内容をGitHub Actions UIで表示
- PRの場合はマージをブロック

### CDワークフロー
- 認証失敗時はエラーを報告して終了
- ビルド失敗時はプッシュをスキップ
- 既存のイメージを上書きしない（タグが異なる場合）

## テスト戦略

### ユニットテスト
- 既存の `*_test.go` ファイルを使用
- `go test -race -coverprofile=coverage.out ./...`

### 統合テスト
- このフェーズではスコープ外
- 将来的にCloud環境でのテストを追加予定

## 依存ライブラリ

GitHub Actions で使用するアクション:

```yaml
actions:
  - actions/checkout@v4          # コードのチェックアウト
  - actions/setup-go@v5          # Go環境のセットアップ
  - actions/cache@v4             # 依存関係のキャッシュ
  - golangci/golangci-lint-action@v6  # golangci-lint実行
  - google-github-actions/auth@v2     # GCP認証
  - docker/setup-buildx-action@v3     # Docker Buildx
  - docker/login-action@v3            # Docker ログイン
  - docker/build-push-action@v6       # イメージビルド・プッシュ
  - docker/metadata-action@v5         # イメージメタデータ
```

## ディレクトリ構造

```
key-management/
├── .github/
│   └── workflows/
│       ├── ci.yml              # CIワークフロー（新規作成）
│       └── cd.yml              # CDワークフロー（新規作成）
└── key-management-service/
    ├── Dockerfile              # 既存
    ├── Makefile                # 既存
    ├── go.mod                  # 既存
    └── ...
```

## 実装の順序

1. `.github/workflows/` ディレクトリを作成
2. `ci.yml` を作成（CIワークフロー）
3. `cd.yml` を作成（CDワークフロー）

## セキュリティ考慮事項

- **Workload Identity連携**: サービスアカウントキーをシークレットとして保存しない
- **最小権限の原則**: Artifact Registry への書き込み権限のみを付与
- **シークレット管理**:
  - `GCP_PROJECT_ID`: GitHub Secretsで管理
  - `GCP_WORKLOAD_IDENTITY_PROVIDER`: GitHub Secretsで管理
  - `GCP_SERVICE_ACCOUNT`: GitHub Secretsで管理
  - `ARTIFACT_REGISTRY_REPO`: GitHub Secretsで管理

## パフォーマンス考慮事項

- **Goモジュールキャッシュ**: `actions/cache` を使用して依存関係をキャッシュ
- **Docker レイヤーキャッシュ**: GitHub Actions のキャッシュを活用
- **並列実行**: CIの各チェックは可能な限り並列で実行

## 将来の拡張性

- Cloud Runへの自動デプロイ追加
- ステージング環境へのデプロイ
- E2Eテストの追加
- セキュリティスキャン（Trivy等）の追加
- 依存関係の自動更新（Dependabot）

## Terraformインフラストラクチャ設計

### 概要

GitHub ActionsからGoogle Cloudリソースへアクセスするために必要なインフラをTerraformで構築する。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Google Cloud                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                    Workload Identity Federation                        │ │
│  │  ┌─────────────────────┐    ┌─────────────────────────────────────┐   │ │
│  │  │  Workload Identity  │    │    Workload Identity Provider       │   │ │
│  │  │       Pool          │───▶│    (GitHub OIDC)                    │   │ │
│  │  │  "github-actions"   │    │    issuer: token.actions.           │   │ │
│  │  │                     │    │            githubusercontent.com     │   │ │
│  │  └─────────────────────┘    └─────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                      │                                       │
│                                      │ IAM Binding                           │
│                                      │ (workloadIdentityUser)                │
│                                      ▼                                       │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                       Service Account                                  │ │
│  │  "github-actions-cicd@{project}.iam.gserviceaccount.com"              │ │
│  │                                                                        │ │
│  │  Roles:                                                                │ │
│  │  - roles/artifactregistry.writer                                       │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                      │                                       │
│                                      │ Push Images                           │
│                                      ▼                                       │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                      Artifact Registry                                 │ │
│  │  Location: asia-northeast1                                             │ │
│  │  Repository: key-management-service                                    │ │
│  │  Format: DOCKER                                                        │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Terraformリソース設計

#### 1. Artifact Registry Repository

```hcl
resource "google_artifact_registry_repository" "key_management_service" {
  location      = "asia-northeast1"
  repository_id = "key-management-service"
  description   = "Docker repository for key-management-service"
  format        = "DOCKER"

  cleanup_policies {
    id     = "keep-minimum-versions"
    action = "KEEP"
    most_recent_versions {
      keep_count = 10
    }
  }
}
```

**設計意図**:
- リージョン: asia-northeast1（東京）でCloud Runと同一リージョン
- クリーンアップポリシー: 最新10バージョンを保持し、ストレージコストを最適化
- フォーマット: Docker（OCI互換）

#### 2. Workload Identity Pool

```hcl
resource "google_iam_workload_identity_pool" "github_actions" {
  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"
  description               = "Workload Identity Pool for GitHub Actions"
}
```

**設計意図**:
- GitHub Actionsからの認証を一元管理
- 複数のリポジトリで共有可能な設計

#### 3. Workload Identity Provider

```hcl
resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_actions.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"
  description                        = "GitHub OIDC provider"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.repository" = "assertion.repository"
    "attribute.ref"        = "assertion.ref"
  }

  attribute_condition = "assertion.repository == '${var.github_repository}'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}
```

**設計意図**:
- OIDC認証によりサービスアカウントキー不要
- attribute_conditionで特定リポジトリからのアクセスのみ許可
- 属性マッピングでGitHub Actions のコンテキスト情報を保持

#### 4. Service Account

```hcl
resource "google_service_account" "github_actions_cicd" {
  account_id   = "github-actions-cicd"
  display_name = "GitHub Actions CI/CD"
  description  = "Service account for GitHub Actions CI/CD pipeline"
}
```

#### 5. IAM Bindings

```hcl
# Artifact Registry への書き込み権限
resource "google_artifact_registry_repository_iam_member" "github_actions_writer" {
  location   = google_artifact_registry_repository.key_management_service.location
  repository = google_artifact_registry_repository.key_management_service.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.github_actions_cicd.email}"
}

# Workload Identity からサービスアカウントへのなりすまし許可
resource "google_service_account_iam_member" "workload_identity_user" {
  service_account_id = google_service_account.github_actions_cicd.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_actions.name}/attribute.repository/${var.github_repository}"
}
```

**設計意図**:
- 最小権限の原則: Artifact Registryへの書き込みのみ許可
- リポジトリレベルでのIAMバインディング（プロジェクト全体ではなく）

### Terraform変数

```hcl
variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
}

variable "region" {
  description = "Google Cloud region"
  type        = string
  default     = "asia-northeast1"
}

variable "github_repository" {
  description = "GitHub repository in format 'owner/repo'"
  type        = string
}
```

### Terraform出力

```hcl
output "artifact_registry_repository" {
  description = "Artifact Registry repository URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.key_management_service.name}"
}

output "workload_identity_provider" {
  description = "Workload Identity Provider resource name"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "service_account_email" {
  description = "Service account email for GitHub Actions"
  value       = google_service_account.github_actions_cicd.email
}
```

### ディレクトリ構造（更新）

```
key-management/
├── .github/
│   └── workflows/
│       ├── ci.yml              # CIワークフロー（新規作成）
│       └── cd.yml              # CDワークフロー（新規作成）
├── terraform/                   # Terraformコード（新規作成）
│   ├── main.tf                 # メインリソース定義
│   ├── variables.tf            # 変数定義
│   ├── outputs.tf              # 出力定義
│   ├── providers.tf            # プロバイダー設定
│   └── terraform.tfvars.example # 変数サンプル
└── key-management-service/
    ├── Dockerfile              # 既存
    ├── Makefile                # 既存
    ├── go.mod                  # 既存
    └── ...
```

### 実装の順序（更新）

1. `terraform/` ディレクトリを作成
2. Terraformファイルを作成（providers.tf, variables.tf, main.tf, outputs.tf）
3. `.github/workflows/` ディレクトリを作成
4. `ci.yml` を作成（CIワークフロー）
5. `cd.yml` を作成（CDワークフロー）

### GitHub Secretsの設定

Terraformで作成したリソースの情報をGitHub Secretsに設定:

| Secret名 | 値 | 説明 |
|---------|-----|------|
| `GCP_PROJECT_ID` | Terraform出力から取得 | Google CloudプロジェクトID |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Terraform出力 `workload_identity_provider` | Workload Identity Provider名 |
| `GCP_SERVICE_ACCOUNT` | Terraform出力 `service_account_email` | サービスアカウントメール |
| `ARTIFACT_REGISTRY_REPO` | Terraform出力 `artifact_registry_repository` | Artifact Registryリポジトリ |

### セキュリティ考慮事項（更新）

- **Workload Identity連携**: サービスアカウントキーをシークレットとして保存しない
- **最小権限の原則**: Artifact Registry への書き込み権限のみを付与
- **リポジトリ制限**: attribute_conditionで特定リポジトリからのアクセスのみ許可
- **Terraformステート管理**: 本番運用ではGCS等のリモートバックエンドを使用（このタスクではローカル）
