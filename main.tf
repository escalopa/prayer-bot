terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }
  required_version = ">= 0.13"
}

variable "access_token" {
  type      = string
  sensitive = true
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

output "region" {
  value = var.region
}

provider "yandex" {
  token     = var.access_token
  cloud_id  = var.cloud_id
  folder_id = var.folder_id
  zone      = var.region
}

###########################
### s3
###########################

resource "yandex_storage_bucket" "bucket" {
  bucket    = "prayer-bot-bucket-${terraform.workspace}"
  folder_id = var.folder_id
  max_size  = 1073741824 # 1 GB in bytes
}

###########################
### ydb
###########################

resource "yandex_ydb_database_serverless" "ydb" {
  name                = "prayer-bot-ydb-${terraform.workspace}"
  location_id         = "ru-central1"
  folder_id           = var.folder_id
  deletion_protection = true

  serverless_database {
    enable_throttling_rcu_limit = false
    storage_size_limit          = 5
  }
}

resource "yandex_ydb_table" "ydb_table_chats" {
  connection_string = yandex_ydb_database_serverless.ydb.ydb_full_endpoint
  path              = "chats"

  primary_key = ["bot_id", "chat_id"]

  column {
    name = "chat_id"
    type = "Int64"
  }

  column {
    name = "bot_id"
    type = "Int64"
  }

  column {
    name = "language_code"
    type = "Utf8"
  }

  column {
    name = "state"
    type = "Utf8"
  }

  column {
    name = "reminder_offset"
    type = "Int32"
  }

  column {
    name = "reminder_message_id"
    type = "Int32"
  }

  column {
    name = "subscribed"
    type = "Bool"
  }

  column {
    name = "subscribed_at"
    type = "Datetime"
  }

  column {
    name = "created_at"
    type = "Datetime"
  }
}

resource "yandex_ydb_table" "ydb_table_prayers" {
  connection_string = yandex_ydb_database_serverless.ydb.ydb_full_endpoint
  path              = "prayers"

  primary_key = ["bot_id", "prayer_date"]

  column {
    name = "bot_id"
    type = "Int64"
  }

  column {
    name = "prayer_date"
    type = "Date"
  }

  column {
    name = "fajr"
    type = "Datetime"
  }
  column {
    name = "shuruq"
    type = "Datetime"
  }
  column {
    name = "dhuhr"
    type = "Datetime"
  }
  column {
    name = "asr"
    type = "Datetime"
  }
  column {
    name = "maghrib"
    type = "Datetime"
  }
  column {
    name = "isha"
    type = "Datetime"
  }
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
  user_hash          = "v1"

  environment = {
    APP_CONFIG = file("${path.module}/_config/${terraform.workspace}/config.json")

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
  user_hash          = "v3"

  environment = {
    APP_CONFIG = file("${path.module}/_config/${terraform.workspace}/config.json")

    YDB_ENDPOINT = yandex_ydb_database_serverless.ydb.ydb_full_endpoint

    REGION     = var.region
    ACCESS_KEY = yandex_iam_service_account_static_access_key.dispatcher_sa_keys.access_key
    SECRET_KEY = yandex_iam_service_account_static_access_key.dispatcher_sa_keys.secret_key
  }

  content {
    zip_filename = data.archive_file.dispatcher_zip.output_path
  }
}

output "dispatcher_fn_id" {
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
  user_hash          = "v1"

  environment = {
    APP_CONFIG = file("${path.module}/_config/${terraform.workspace}/config.json")

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
