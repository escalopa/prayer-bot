locals {
  required_services = toset([
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudfunctions.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "cloudscheduler.googleapis.com",
    "eventarc.googleapis.com",
    "iam.googleapis.com",
    "pubsub.googleapis.com",
    "run.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
  ])

  source_bucket_name = "${var.project_id}-prayer-bot-src-${var.environment}"
  data_bucket_name   = "prayer-bot-data-${var.environment}"
  runtime_sa_id      = "prayer-bot-${var.environment}"

  app_config_file = coalesce(var.app_config_path, "${path.module}/../../config.json")
  app_config_b64  = base64encode(file(local.app_config_file))

  ydb_endpoint = var.dual_write ? (var.ydb_endpoint != "" ? var.ydb_endpoint : try(data.terraform_remote_state.yc[0].outputs.ydb_endpoint, "")) : ""

  common_env = {
    DB_PRIMARY   = "postgres"
    DUAL_WRITE   = var.dual_write ? "true" : "false"
    YDB_ENDPOINT = local.ydb_endpoint
    APP_CONFIG   = local.app_config_b64
  }

  function_env = merge(
    local.common_env,
    {
      DATABASE_URL = var.supabase_db_url
    },
    var.dual_write && var.ydb_token != "" ? { YDB_TOKEN = var.ydb_token } : {}
  )
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

resource "google_service_account" "runtime" {
  account_id   = local.runtime_sa_id
  display_name = "Prayer bot runtime (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_project_iam_member" "runtime_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_project_iam_member" "runtime_storage" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_storage_bucket_iam_member" "runtime_data_admin" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_project_iam_member" "runtime_eventarc_receiver" {
  count   = var.enable_loader_trigger ? 1 : 0
  project = var.project_id
  role    = "roles/eventarc.eventReceiver"
  member  = "serviceAccount:${google_service_account.runtime.email}"
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

resource "google_storage_bucket" "data" {
  name                        = local.data_bucket_name
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = false
  public_access_prevention    = "enforced"

  labels = {
    app = "prayer-bot"
  }

  depends_on = [google_project_service.required]
}

data "archive_file" "dispatcher_zip" {
  type        = "zip"
  source_dir  = "${path.module}/../../serverless/dispatcher"
  output_path = "${path.module}/dispatcher.zip"
}

data "archive_file" "reminder_zip" {
  type        = "zip"
  source_dir  = "${path.module}/../../serverless/reminder"
  output_path = "${path.module}/reminder.zip"
}

data "archive_file" "loader_zip" {
  type        = "zip"
  source_dir  = "${path.module}/../../serverless/loader"
  output_path = "${path.module}/loader.zip"
}

resource "google_storage_bucket_object" "dispatcher_zip" {
  name   = "dispatcher-${data.archive_file.dispatcher_zip.output_md5}.zip"
  bucket = google_storage_bucket.source.name
  source = data.archive_file.dispatcher_zip.output_path
}

resource "google_storage_bucket_object" "reminder_zip" {
  name   = "reminder-${data.archive_file.reminder_zip.output_md5}.zip"
  bucket = google_storage_bucket.source.name
  source = data.archive_file.reminder_zip.output_path
}

resource "google_storage_bucket_object" "loader_zip" {
  name   = "loader-${data.archive_file.loader_zip.output_md5}.zip"
  bucket = google_storage_bucket.source.name
  source = data.archive_file.loader_zip.output_path
}

resource "google_cloudfunctions2_function" "dispatcher" {
  name     = "prayer-bot-dispatcher-${var.environment}"
  location = var.region

  build_config {
    runtime     = "go121"
    entry_point = "DispatcherHTTP"

    environment_variables = {
      GO_BUILD_TAGS = "gcp"
    }

    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.dispatcher_zip.name
      }
    }
  }

  service_config {
    available_memory      = "256M"
    timeout_seconds       = 30
    ingress_settings      = "ALLOW_ALL"
    service_account_email = google_service_account.runtime.email

    environment_variables = local.function_env
  }

  depends_on = [
    google_project_service.required,
    google_project_iam_member.runtime_logging,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "dispatcher_public" {
  project  = var.project_id
  location = var.region
  name     = google_cloudfunctions2_function.dispatcher.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloudfunctions2_function" "reminder" {
  name     = "prayer-bot-reminder-${var.environment}"
  location = var.region

  build_config {
    runtime     = "go121"
    entry_point = "ReminderHTTP"

    environment_variables = {
      GO_BUILD_TAGS = "gcp"
    }

    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.reminder_zip.name
      }
    }
  }

  service_config {
    available_memory      = "256M"
    timeout_seconds       = 60
    ingress_settings      = "ALLOW_ALL"
    service_account_email = google_service_account.runtime.email

    environment_variables = local.function_env
  }

  depends_on = [
    google_project_service.required,
  ]
}

resource "google_service_account" "scheduler" {
  account_id   = "prayer-bot-scheduler-${var.environment}"
  display_name = "Prayer bot scheduler (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_cloud_run_v2_service_iam_member" "reminder_scheduler" {
  count    = var.enable_scheduler ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloudfunctions2_function.reminder.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler.email}"
}

resource "google_cloud_scheduler_job" "reminder" {
  count    = var.enable_scheduler ? 1 : 0
  name     = "prayer-bot-reminder-${var.environment}"
  schedule = "* * * * *"
  region   = var.region

  http_target {
    http_method = "POST"
    uri         = google_cloudfunctions2_function.reminder.service_config[0].uri

    oidc_token {
      service_account_email = google_service_account.scheduler.email
    }
  }

  depends_on = [
    google_project_service.required,
    google_cloud_run_v2_service_iam_member.reminder_scheduler,
  ]
}

resource "google_cloudfunctions2_function" "loader" {
  name     = "prayer-bot-loader-${var.environment}"
  location = var.region

  build_config {
    runtime     = "go121"
    entry_point = "LoaderCloudEvent"

    environment_variables = {
      GO_BUILD_TAGS = "gcp"
    }

    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.loader_zip.name
      }
    }
  }

  service_config {
    available_memory      = "256M"
    timeout_seconds       = 120
    ingress_settings      = "ALLOW_INTERNAL_ONLY"
    service_account_email = google_service_account.runtime.email

    environment_variables = merge(local.function_env, {
      STORAGE_BACKEND = "gcs"
      GCS_BUCKET      = google_storage_bucket.data.name
    })
  }

  depends_on = [
    google_project_service.required,
  ]
}

resource "google_eventarc_trigger" "loader" {
  count    = var.enable_loader_trigger ? 1 : 0
  name     = "prayer-bot-loader-${var.environment}"
  location = var.region

  matching_criteria {
    attribute = "type"
    value     = "google.cloud.storage.object.v1.finalized"
  }

  matching_criteria {
    attribute = "bucket"
    value     = google_storage_bucket.data.name
  }

  destination {
    cloud_run_service {
      service = google_cloudfunctions2_function.loader.name
      region  = var.region
    }
  }

  service_account = google_service_account.runtime.email

  depends_on = [
    google_project_service.required,
    google_cloudfunctions2_function.loader,
    google_project_iam_member.runtime_eventarc_receiver,
    google_cloud_run_v2_service_iam_member.loader_eventarc,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "loader_eventarc" {
  count    = var.enable_loader_trigger ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloudfunctions2_function.loader.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.runtime.email}"
}
