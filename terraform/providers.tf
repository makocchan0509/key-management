terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "7.17.0"
    }
  }

  # Production: Configure remote backend (e.g., GCS)
  # backend "gcs" {
  #   bucket = "your-terraform-state-bucket"
  #   prefix = "key-management/terraform.tfstate"
  # }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
