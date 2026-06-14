variable "project_id" {
  type        = string
  description = "GCP project ID. The CI deploy service account (GCP_SA_KEY) needs: roles/serviceusage.serviceUsageAdmin, roles/resourcemanager.projectIamAdmin, roles/iam.serviceAccountAdmin, roles/secretmanager.admin, roles/cloudfunctions.admin, roles/run.admin, roles/storage.admin on this project and the tfstate bucket."
}

variable "region" {
  type    = string
  default = "europe-west1"
}

variable "environment" {
  type    = string
  default = "prod"
}

variable "yc_dispatcher_url" {
  type = string
}
