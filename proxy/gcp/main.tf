locals {
  required_services = toset([
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudfunctions.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "run.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
  ])

  source_bucket_name = "${var.project_id}-prayer-bot-proxy-src-${var.environment}"
}

provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_project_service" "required" {
  for_each = local.required_services

  project            = var.project_id
  service            = each.key
  disable_on_destroy = false
}

data "archive_file" "proxy_zip" {
  type        = "zip"
  source_dir  = "${path.module}/../function"
  output_path = "${path.module}/proxy-function.zip"
}

resource "google_storage_bucket" "source" {
  name                        = local.source_bucket_name
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true
  public_access_prevention    = "enforced"
  labels = {
    app = "prayer-bot"
  }

  depends_on = [google_project_service.required]
}

resource "google_storage_bucket_object" "proxy_zip" {
  name   = "proxy-function-${data.archive_file.proxy_zip.output_md5}.zip"
  bucket = google_storage_bucket.source.name
  source = data.archive_file.proxy_zip.output_path
}

resource "google_service_account" "runtime" {
  account_id   = "prayer-bot-proxy-${var.environment}"
  display_name = "Prayer bot proxy runtime (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_project_iam_member" "runtime_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.runtime.email}"

  depends_on = [google_project_service.required]
}

resource "google_cloudfunctions2_function" "webhook_proxy" {
  name     = "prayer-bot-webhook-proxy-${var.environment}"
  location = var.region

  build_config {
    runtime     = "go125"
    entry_point = "WebhookProxy"

    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.proxy_zip.name
      }
    }
  }

  service_config {
    available_memory      = "256M"
    timeout_seconds       = 30
    ingress_settings      = "ALLOW_ALL"
    service_account_email = google_service_account.runtime.email

    environment_variables = {
      YC_DISPATCHER_URL = var.yc_dispatcher_url
    }
  }

  depends_on = [
    google_project_service.required,
    google_project_iam_member.runtime_logging,
  ]
}

# Gen2 functions run on Cloud Run; roles/run.invoker must be on the Run service, not the function.
resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloudfunctions2_function.webhook_proxy.name
  role     = "roles/run.invoker"
  member   = "allUsers"

  depends_on = [google_cloudfunctions2_function.webhook_proxy]
}
