variable "project_id" {
  type        = string
  description = "Existing GCP project ID for the selected GitHub environment."
}

variable "region" {
  type    = string
  default = "europe-west1"
}

variable "environment" {
  type    = string
  default = "production"

  validation {
    condition     = contains(["testing", "production"], var.environment)
    error_message = "environment must be testing or production"
  }
}

variable "image" {
  type        = string
  description = "Immutable global-bot container image reference."
}

variable "supabase_db_url" {
  type        = string
  sensitive   = true
  description = "Existing Supabase/PostgreSQL connection URL. Data is isolated in the global_bot schema."
}

variable "telegram_token_secret_id" {
  type        = string
  default     = ""
  description = "Optional Secret Manager override. Defaults to global-prayer-bot-token-<environment>."
}

variable "webhook_secret_secret_id" {
  type        = string
  default     = ""
  description = "Optional Secret Manager override. Defaults to global-prayer-bot-webhook-secret-<environment>."
}

variable "owner_id_secret_id" {
  type        = string
  default     = ""
  description = "Optional Secret Manager override. Defaults to global-prayer-bot-owner-id-<environment>."
}

variable "dispatch_schedule" {
  type        = string
  default     = "* * * * *"
  description = "Cloud Scheduler cron for claiming due reminders."
}

variable "max_instances" {
  type    = number
  default = 10
}
