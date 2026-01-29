variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
  default     = "genai-agent-engine"
}

variable "region" {
  description = "Google Cloud region"
  type        = string
  default     = "asia-northeast1"
}

variable "github_repository" {
  description = "GitHub repository in format 'owner/repo'"
  type        = string
  default     = "makocchan0509/key-management"
}
