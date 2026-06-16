variable "project_id" {
  type        = string
  description = "GCP project ID. The CI deploy service account (GCP_SA_KEY) needs: roles/serviceusage.serviceUsageAdmin, roles/resourcemanager.projectIamAdmin, roles/iam.serviceAccountAdmin, roles/iam.serviceAccountUser, roles/cloudfunctions.admin, roles/run.admin, roles/storage.admin, roles/cloudscheduler.admin, roles/eventarc.admin, roles/pubsub.admin on this project and the tfstate bucket."
}

variable "region" {
  type    = string
  default = "europe-west1"
}

variable "environment" {
  type    = string
  default = "prod"
}

variable "enable_loader_trigger" {
  type        = bool
  default     = false
  description = "Enable Eventarc loader trigger after bucket migration."
}

variable "enable_scheduler" {
  type        = bool
  default     = false
  description = "Enable Cloud Scheduler reminder cron (phase 2 / gcp-only)."
}

variable "dual_write" {
  type    = bool
  default = true
}

variable "ydb_endpoint" {
  type        = string
  description = "Optional YDB endpoint override; defaults to YC Terraform remote state."
  default     = ""
}

variable "app_config_path" {
  type        = string
  description = "Path to bot config JSON; base64-encoded into APP_CONFIG for functions."
  default     = null
}

variable "yc_tfstate_bucket" {
  type        = string
  description = "S3 bucket holding Yandex Cloud Terraform state."
  default     = "escalopa-tfstate"
}

variable "yc_tfstate_key" {
  type        = string
  description = "State key for Yandex Cloud Terraform."
  default     = "prayer-bot/terraform.tfstate"
}

variable "supabase_db_url" {
  type        = string
  sensitive   = true
  description = "Supabase Postgres connection URL."
}

variable "ydb_token" {
  type        = string
  sensitive   = true
  description = "YDB IAM token for dual-write from GCP."
  default     = ""
}

variable "deploy_service_account_email" {
  type        = string
  description = "Email of the CI/Terraform deploy SA (GCP_SA_KEY). IAM bindings applied in phase 1."
  default     = ""
}
