# Artifact Registry Repository
resource "google_artifact_registry_repository" "key_management_service" {
  location      = var.region
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

# Workload Identity Pool
resource "google_iam_workload_identity_pool" "github_actions" {
  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"
  description               = "Workload Identity Pool for GitHub Actions"
}

# Workload Identity Provider
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

# Service Account
resource "google_service_account" "github_actions_cicd" {
  account_id   = "github-actions-cicd"
  display_name = "GitHub Actions CI/CD"
  description  = "Service account for GitHub Actions CI/CD pipeline"
}

# Artifact Registry IAM Binding
resource "google_artifact_registry_repository_iam_member" "github_actions_writer" {
  location   = google_artifact_registry_repository.key_management_service.location
  repository = google_artifact_registry_repository.key_management_service.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.github_actions_cicd.email}"
}

# Workload Identity IAM Binding
resource "google_service_account_iam_member" "workload_identity_user" {
  service_account_id = google_service_account.github_actions_cicd.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_actions.name}/attribute.repository/${var.github_repository}"
}
