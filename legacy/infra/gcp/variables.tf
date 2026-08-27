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
  default     = true
  description = "Enable Eventarc loader trigger on the data bucket."
}

variable "enable_scheduler" {
  type        = bool
  default     = true
  description = "Enable Cloud Scheduler reminder cron."
}

variable "app_config_path" {
  type        = string
  description = "Path to bot config JSON; base64-encoded into APP_CONFIG for functions."
  default     = null
}

variable "supabase_db_url" {
  type        = string
  sensitive   = true
  description = "Supabase transaction pooler URL (port 6543) for runtime DATABASE_URL on Cloud Functions."
}

variable "deploy_service_account_email" {
  type        = string
  description = "Email of the CI/Terraform deploy SA (GCP_SA_KEY). IAM bindings applied in phase 1."
  default     = ""
}

variable "dispatcher_timeout_seconds" {
  type        = number
  description = "Request timeout for the Telegram webhook dispatcher (Cloud Functions Gen2 / Cloud Run)."
  default     = 60
}
