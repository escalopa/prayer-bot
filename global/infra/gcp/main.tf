locals {
  environment_code         = var.environment == "production" ? "prod" : "test"
  name                     = "global-prayer-${local.environment_code}"
  database_schema          = "global_bot_${var.environment}"
  telegram_token_secret_id = var.telegram_token_secret_id != "" ? var.telegram_token_secret_id : "global-prayer-bot-token-${var.environment}"
  webhook_secret_secret_id = var.webhook_secret_secret_id != "" ? var.webhook_secret_secret_id : "global-prayer-bot-webhook-secret-${var.environment}"
  owner_id_secret_id       = var.owner_id_secret_id != "" ? var.owner_id_secret_id : "global-prayer-bot-owner-id-${var.environment}"
  labels = {
    app         = "global-prayer-bot"
    environment = var.environment
  }
  required_services = toset([
    "apikeys.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "cloudscheduler.googleapis.com",
    "cloudtasks.googleapis.com",
    "geocoding-backend.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "serviceusage.googleapis.com",
    "timezone-backend.googleapis.com",
  ])
}

resource "google_project_service" "required" {
  for_each = local.required_services

  project            = var.project_id
  service            = each.key
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "global" {
  location      = var.region
  repository_id = "global-prayer-bot-${local.environment_code}"
  description   = "Container images for the isolated global prayer bot"
  format        = "DOCKER"
  labels        = local.labels

  depends_on = [google_project_service.required]
}

resource "google_service_account" "webhook" {
  account_id   = "global-prayer-hook-${local.environment_code}"
  display_name = "Global prayer webhook (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_service_account" "dispatch" {
  account_id   = "global-prayer-dispatch-${local.environment_code}"
  display_name = "Global prayer dispatch (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_service_account" "sender" {
  account_id   = "global-prayer-sender-${local.environment_code}"
  display_name = "Global prayer sender (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_service_account" "scheduler" {
  account_id   = "global-prayer-cron-${local.environment_code}"
  display_name = "Global prayer scheduler (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_service_account" "task_caller" {
  account_id   = "global-prayer-task-${local.environment_code}"
  display_name = "Global prayer Cloud Tasks caller (${var.environment})"

  depends_on = [google_project_service.required]
}

resource "google_project_iam_member" "runtime_logging" {
  for_each = toset([
    google_service_account.webhook.email,
    google_service_account.dispatch.email,
    google_service_account.sender.email,
  ])

  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "dispatch_task_enqueuer" {
  project = var.project_id
  role    = "roles/cloudtasks.enqueuer"
  member  = "serviceAccount:${google_service_account.dispatch.email}"
}

resource "google_service_account_iam_member" "dispatch_can_use_task_caller" {
  service_account_id = google_service_account.task_caller.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.dispatch.email}"
}

data "google_project" "current" {
  project_id = var.project_id
}

resource "google_service_account_iam_member" "cloudtasks_can_mint_task_token" {
  service_account_id = google_service_account.task_caller.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-cloudtasks.iam.gserviceaccount.com"

  depends_on = [google_project_service.required]
}

resource "google_service_account_iam_member" "cloudscheduler_can_mint_scheduler_token" {
  service_account_id = google_service_account.scheduler.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-cloudscheduler.iam.gserviceaccount.com"

  depends_on = [google_project_service.required]
}

data "google_secret_manager_secret" "telegram_token" {
  secret_id = local.telegram_token_secret_id
  project   = var.project_id
}

data "google_secret_manager_secret" "webhook_secret" {
  secret_id = local.webhook_secret_secret_id
  project   = var.project_id
}

data "google_secret_manager_secret" "owner_id" {
  secret_id = local.owner_id_secret_id
  project   = var.project_id
}

resource "google_secret_manager_secret" "database_url" {
  secret_id = "${local.name}-database-url"
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "database_url" {
  secret      = google_secret_manager_secret.database_url.id
  secret_data = var.supabase_db_url
}

resource "google_apikeys_key" "maps" {
  name         = "global-prayer-maps-${local.environment_code}"
  display_name = "Global prayer bot Maps APIs (${var.environment})"

  restrictions {
    api_targets {
      service = "timezone-backend.googleapis.com"
    }
    api_targets {
      service = "geocoding-backend.googleapis.com"
    }
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "maps_api_key" {
  secret_id = "${local.name}-maps-api-key"
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "maps_api_key" {
  secret      = google_secret_manager_secret.maps_api_key.id
  secret_data = google_apikeys_key.maps.key_string
}

locals {
  runtime_secret_access = {
    webhook_token  = { secret = data.google_secret_manager_secret.telegram_token.secret_id, member = google_service_account.webhook.email }
    webhook_secret = { secret = data.google_secret_manager_secret.webhook_secret.secret_id, member = google_service_account.webhook.email }
    webhook_owner  = { secret = data.google_secret_manager_secret.owner_id.secret_id, member = google_service_account.webhook.email }
    webhook_maps   = { secret = google_secret_manager_secret.maps_api_key.secret_id, member = google_service_account.webhook.email }
    webhook_db     = { secret = google_secret_manager_secret.database_url.secret_id, member = google_service_account.webhook.email }
    dispatch_db    = { secret = google_secret_manager_secret.database_url.secret_id, member = google_service_account.dispatch.email }
    sender_token   = { secret = data.google_secret_manager_secret.telegram_token.secret_id, member = google_service_account.sender.email }
    sender_db      = { secret = google_secret_manager_secret.database_url.secret_id, member = google_service_account.sender.email }
  }
}

resource "google_secret_manager_secret_iam_member" "runtime" {
  for_each = local.runtime_secret_access

  project   = var.project_id
  secret_id = each.value.secret
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${each.value.member}"
}

resource "google_cloud_tasks_queue" "notifications" {
  name     = "${local.name}-notifications"
  location = var.region

  rate_limits {
    max_concurrent_dispatches = 50
    max_dispatches_per_second = 20
  }

  retry_config {
    max_attempts       = 8
    max_retry_duration = "3600s"
    min_backoff        = "5s"
    max_backoff        = "300s"
    max_doublings      = 5
  }

  depends_on = [google_project_service.required]
}

resource "google_cloud_run_v2_service" "webhook" {
  name                = "${local.name}-webhook"
  location            = var.region
  deletion_protection = false
  ingress             = "INGRESS_TRAFFIC_ALL"
  labels              = local.labels

  template {
    service_account = google_service_account.webhook.email
    timeout         = "30s"

    scaling {
      min_instance_count = 0
      max_instance_count = var.max_instances
    }

    containers {
      image   = var.image
      command = ["/webhook"]

      resources {
        limits   = { cpu = "1", memory = "256Mi" }
        cpu_idle = true
      }

      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "GLOBAL_DB_SCHEMA"
        value = local.database_schema
      }
      env {
        name = "GLOBAL_BOT_TOKEN"
        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.telegram_token.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "GLOBAL_WEBHOOK_SECRET"
        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.webhook_secret.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "GLOBAL_OWNER_ID"
        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.owner_id.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "GOOGLE_MAPS_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.maps_api_key.secret_id
            version = "latest"
          }
        }
      }
    }

    max_instance_request_concurrency = 40
  }

  depends_on = [google_project_service.required, google_secret_manager_secret_iam_member.runtime]
}

resource "google_cloud_run_v2_service" "dispatch" {
  name                = "${local.name}-dispatch"
  location            = var.region
  deletion_protection = false
  ingress             = "INGRESS_TRAFFIC_ALL"
  labels              = local.labels

  template {
    service_account = google_service_account.dispatch.email
    timeout         = "60s"

    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image   = var.image
      command = ["/dispatch"]

      resources {
        limits   = { cpu = "1", memory = "256Mi" }
        cpu_idle = true
      }

      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "GLOBAL_DB_SCHEMA"
        value = local.database_schema
      }
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "GCP_REGION"
        value = var.region
      }
      env {
        name  = "CLOUD_TASKS_QUEUE"
        value = google_cloud_tasks_queue.notifications.name
      }
      env {
        name  = "GLOBAL_SENDER_URL"
        value = google_cloud_run_v2_service.sender.uri
      }
      env {
        name  = "TASK_CALLER_SERVICE_ACCOUNT"
        value = google_service_account.task_caller.email
      }
    }
  }

  depends_on = [google_project_service.required, google_secret_manager_secret_iam_member.runtime]
}

resource "google_cloud_run_v2_service" "sender" {
  name                = "${local.name}-sender"
  location            = var.region
  deletion_protection = false
  ingress             = "INGRESS_TRAFFIC_ALL"
  labels              = local.labels

  template {
    service_account = google_service_account.sender.email
    timeout         = "30s"

    scaling {
      min_instance_count = 0
      max_instance_count = var.max_instances
    }

    containers {
      image   = var.image
      command = ["/send"]

      resources {
        limits   = { cpu = "1", memory = "256Mi" }
        cpu_idle = true
      }

      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "GLOBAL_DB_SCHEMA"
        value = local.database_schema
      }
      env {
        name = "GLOBAL_BOT_TOKEN"
        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.telegram_token.secret_id
            version = "latest"
          }
        }
      }
    }

    max_instance_request_concurrency = 20
  }

  depends_on = [google_project_service.required, google_secret_manager_secret_iam_member.runtime]
}

resource "google_cloud_run_v2_service_iam_member" "webhook_public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webhook.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "sender_task_caller" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.sender.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.task_caller.email}"
}

resource "google_cloud_run_v2_service_iam_member" "dispatch_scheduler" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.dispatch.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler.email}"
}

resource "google_cloud_scheduler_job" "dispatch" {
  name             = "${local.name}-dispatch"
  region           = var.region
  schedule         = var.dispatch_schedule
  time_zone        = "Etc/UTC"
  attempt_deadline = "60s"

  http_target {
    http_method = "POST"
    uri         = "${google_cloud_run_v2_service.dispatch.uri}/dispatch"

    oidc_token {
      service_account_email = google_service_account.scheduler.email
      audience              = google_cloud_run_v2_service.dispatch.uri
    }
  }

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
    max_doublings        = 3
  }

  depends_on = [google_cloud_run_v2_service_iam_member.dispatch_scheduler]
}

resource "google_cloud_scheduler_job" "maintenance" {
  name             = "${local.name}-maintenance"
  region           = var.region
  schedule         = "17 3 * * *"
  time_zone        = "Etc/UTC"
  attempt_deadline = "60s"

  http_target {
    http_method = "POST"
    uri         = "${google_cloud_run_v2_service.dispatch.uri}/maintenance"

    oidc_token {
      service_account_email = google_service_account.scheduler.email
      audience              = google_cloud_run_v2_service.dispatch.uri
    }
  }

  retry_config {
    retry_count          = 3
    min_backoff_duration = "5s"
    max_backoff_duration = "60s"
    max_doublings        = 3
  }

  depends_on = [google_cloud_run_v2_service_iam_member.dispatch_scheduler]
}
