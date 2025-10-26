terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }

  backend "s3" {
    endpoints = {
      s3 = "https://storage.yandexcloud.net"
    }
    region = "ru-central1"

    bucket = "escalopa-tfstate"
    key    = "prayer-bot/terraform.tfstate"

    workspace_key_prefix = ""

    skip_region_validation      = true
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_s3_checksum            = true
  }

  required_version = ">= 0.13"
}

locals {
  env = terraform.workspace
}

variable "cloud_id" {
  type = string
}

variable "folder_id" {
  type = string
}

variable "region" {
  type    = string
  default = "ru-central1-d"
}

provider "yandex" {
  service_account_key_file = "iam.json"
  cloud_id                 = var.cloud_id
  folder_id                = var.folder_id
  zone                     = var.region
}

###########################
### s3
###########################

resource "yandex_storage_bucket" "bucket" {
  bucket    = "prayer-bot-bucket-${local.env}"
  folder_id = var.folder_id
  max_size  = 1073741824 # 1 GB in bytes
}

output "bucket_name" {
  value = yandex_storage_bucket.bucket.bucket
}

###########################
### ydb
###########################

resource "yandex_ydb_database_serverless" "ydb" {
  name                = "prayer-bot-ydb-${local.env}"
  location_id         = "ru-central1"
  folder_id           = var.folder_id
  deletion_protection = true

  serverless_database {
    enable_throttling_rcu_limit = false
    storage_size_limit          = 5
  }
}

output "ydb_connection_string" {
  value     = yandex_ydb_database_serverless.ydb.ydb_full_endpoint
  sensitive = true
}

###########################
### trigger
###########################

resource "yandex_iam_service_account" "trigger_sa" {
  name = "trigger-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "trigger_sa_invoker" {
  folder_id = var.folder_id
  role      = "functions.functionInvoker"
  member    = "serviceAccount:${yandex_iam_service_account.trigger_sa.id}"
}

###########################
### loader-sa
###########################

resource "yandex_iam_service_account" "loader_sa" {
  name = "loader-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "loader_sa_storage" {
  folder_id = var.folder_id
  role      = "storage.viewer"
  member    = "serviceAccount:${yandex_iam_service_account.loader_sa.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "loader_sa_ydb" {
  folder_id = var.folder_id
  role      = "ydb.editor"
  member    = "serviceAccount:${yandex_iam_service_account.loader_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "loader_sa_keys" {
  service_account_id = yandex_iam_service_account.loader_sa.id
}

###########################
### loader-fn
###########################

data "archive_file" "loader_zip" {
  type        = "zip"
  source_dir  = "${path.module}/serverless/loader"
  output_path = "${path.module}/serverless/loader.zip"
}

resource "yandex_function" "loader_fn" {
  name               = "loader-fn"
  runtime            = "golang121"
  entrypoint         = "main.Handler"
  memory             = 128
  execution_timeout  = 5
  service_account_id = yandex_iam_service_account.loader_sa.id
  folder_id          = var.folder_id
  user_hash          = filemd5(data.archive_file.loader_zip.output_path)

  environment = {
    APP_CONFIG = file("${path.module}/config.json")

    S3_ENDPOINT  = "https://storage.yandexcloud.net"
    YDB_ENDPOINT = yandex_ydb_database_serverless.ydb.ydb_full_endpoint

    REGION     = var.region
    ACCESS_KEY = yandex_iam_service_account_static_access_key.loader_sa_keys.access_key
    SECRET_KEY = yandex_iam_service_account_static_access_key.loader_sa_keys.secret_key
  }

  content {
    zip_filename = data.archive_file.loader_zip.output_path
  }
}

resource "yandex_function_trigger" "loader_trigger" {
  name = "loader"
  function {
    id                 = yandex_function.loader_fn.id
    service_account_id = yandex_iam_service_account.trigger_sa.id
  }

  object_storage {
    bucket_id    = yandex_storage_bucket.bucket.id
    batch_cutoff = "1"
    batch_size   = "1"
    create       = true
    update       = true
    delete       = false
  }
}

###########################
### dispatcher-sa
###########################

resource "yandex_iam_service_account" "dispatcher_sa" {
  name = "dispatcher-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "dispatcher_sa_invoker" {
  folder_id = var.folder_id
  role      = "functions.functionInvoker"
  member    = "system:allUsers" // allow public access
}

resource "yandex_resourcemanager_folder_iam_member" "dispatcher_sa_ydb" {
  folder_id = var.folder_id
  role      = "ydb.editor"
  member    = "serviceAccount:${yandex_iam_service_account.dispatcher_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "dispatcher_sa_keys" {
  service_account_id = yandex_iam_service_account.dispatcher_sa.id
}

###########################
### dispatcher-fn
###########################

data "archive_file" "dispatcher_zip" {
  type        = "zip"
  source_dir  = "${path.module}/serverless/dispatcher"
  output_path = "${path.module}/serverless/dispatcher.zip"
}

resource "yandex_function" "dispatcher_fn" {
  name               = "dispatcher-fn"
  runtime            = "golang121"
  entrypoint         = "main.Handler"
  memory             = 128
  execution_timeout  = 5
  service_account_id = yandex_iam_service_account.dispatcher_sa.id
  folder_id          = var.folder_id
  user_hash          = filemd5(data.archive_file.dispatcher_zip.output_path)

  environment = {
    APP_CONFIG = file("${path.module}/config.json")

    YDB_ENDPOINT = yandex_ydb_database_serverless.ydb.ydb_full_endpoint

    REGION     = var.region
    ACCESS_KEY = yandex_iam_service_account_static_access_key.dispatcher_sa_keys.access_key
    SECRET_KEY = yandex_iam_service_account_static_access_key.dispatcher_sa_keys.secret_key
  }

  content {
    zip_filename = data.archive_file.dispatcher_zip.output_path
  }
}

output "dispatcher_function_id" {
  value = yandex_function.dispatcher_fn.id // used to set Webhook URL on Telegram
}

###########################
### reminder-sa
###########################

resource "yandex_iam_service_account" "reminder_sa" {
  name = "reminder-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "reminder_sa_ydb" {
  folder_id = var.folder_id
  role      = "ydb.editor"
  member    = "serviceAccount:${yandex_iam_service_account.reminder_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "reminder_sa_keys" {
  service_account_id = yandex_iam_service_account.reminder_sa.id
}

###########################
### reminder-fn
###########################

data "archive_file" "reminder_zip" {
  type        = "zip"
  source_dir  = "${path.module}/serverless/reminder"
  output_path = "${path.module}/serverless/reminder.zip"
}

resource "yandex_function" "reminder_fn" {
  name               = "reminder-fn"
  runtime            = "golang121"
  entrypoint         = "main.Handler"
  memory             = 128
  execution_timeout  = 5
  service_account_id = yandex_iam_service_account.reminder_sa.id
  folder_id          = var.folder_id
  user_hash          = filemd5(data.archive_file.reminder_zip.output_path)

  environment = {
    APP_CONFIG = file("${path.module}/config.json")

    YDB_ENDPOINT = yandex_ydb_database_serverless.ydb.ydb_full_endpoint

    REGION     = var.region
    ACCESS_KEY = yandex_iam_service_account_static_access_key.reminder_sa_keys.access_key
    SECRET_KEY = yandex_iam_service_account_static_access_key.reminder_sa_keys.secret_key
  }

  content {
    zip_filename = data.archive_file.reminder_zip.output_path
  }
}

resource "yandex_function_trigger" "reminder_trigger" {
  name = "reminder"
  function {
    id                 = yandex_function.reminder_fn.id
    service_account_id = yandex_iam_service_account.trigger_sa.id
  }

  timer {
    cron_expression = "* * * * ? *" # every minute
  }
}
