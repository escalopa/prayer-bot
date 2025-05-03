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

provider "yandex" {
  token     = var.access_token
  cloud_id  = var.cloud_id
  folder_id = var.folder_id
  zone      = var.region
}

#######################################
### object storage
#######################################

resource "yandex_storage_bucket" "bucket" {
  bucket    = "prayer-bot-bucket"
  folder_id = var.folder_id
  max_size  = 1073741824 # 1 GB in bytes
}

#######################################
### ydb
#######################################

resource "yandex_ydb_database_serverless" "ydb" {
  name                = "prayer-bot-ydb"
  location_id         = "ru-central1"
  folder_id           = var.folder_id
  deletion_protection = true

  serverless_database {
    enable_throttling_rcu_limit = false
    storage_size_limit          = 5
  }
}

resource "yandex_ydb_table" "users" {
  connection_string = yandex_ydb_database_serverless.ydb.ydb_full_endpoint
  path              = "users"

  primary_key = ["chat_id", "bot_id"]

  column {
    name = "chat_id"
    type = "Uint64"
  }

  column {
    name = "bot_id"
    type = "Uint8"
  }

  column {
    name = "language"
    type = "Utf8"
  }

  column {
    name = "notify_before"
    type = "Uint8"
  }

  column {
    name = "last_notify_message_id"
    type = "Uint64"
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

resource "yandex_ydb_table" "prayers" {
  connection_string = yandex_ydb_database_serverless.ydb.ydb_full_endpoint
  path              = "prayers"

  primary_key = ["bot_id", "prayer_date"]

  column {
    name = "bot_id"
    type = "Uint8"
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

#######################################
### message queue
#######################################

resource "yandex_iam_service_account" "mq_sa" {
  name      = "mq-sa"
  folder_id = var.folder_id
}

resource "yandex_resourcemanager_folder_iam_member" "mq_sa_editor" {
  folder_id = var.folder_id
  role      = "editor"
  member    = "serviceAccount:${yandex_iam_service_account.mq_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "mq_sa_keys" {
  service_account_id = yandex_iam_service_account.mq_sa.id
}

resource "yandex_message_queue" "standard_queue" {
  name                       = "prayer-bot-queue"
  visibility_timeout_seconds = 600
  receive_wait_time_seconds  = 20
  message_retention_seconds  = 1209600 # 14 days
  access_key                 = yandex_iam_service_account_static_access_key.mq_sa_keys.access_key
  secret_key                 = yandex_iam_service_account_static_access_key.mq_sa_keys.secret_key
}

#######################################
### serverless functions - loader
#######################################

resource "yandex_iam_service_account" "loader_sa" {
  name = "loader-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "loader_sa_function_role" {
  folder_id = var.folder_id
  role      = "functions.functionInvoker"
  member    = "serviceAccount:${yandex_iam_service_account.loader_sa.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "loader_sa_storage_role" {
  folder_id = var.folder_id
  role      = "storage.viewer"
  member    = "serviceAccount:${yandex_iam_service_account.loader_sa.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "loader_sa_ydb_role" {
  folder_id = var.folder_id
  role      = "ydb.editor"
  member    = "serviceAccount:${yandex_iam_service_account.loader_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "loader_sa_keys" {
  service_account_id = yandex_iam_service_account.loader_sa.id
}

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
  execution_timeout  = 10
  service_account_id = yandex_iam_service_account.loader_sa.id
  folder_id          = var.folder_id
  user_hash          = "v1"

  environment = {
    S3_BUCKET    = yandex_storage_bucket.bucket.bucket
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
    service_account_id = yandex_iam_service_account.loader_sa.id
  }

  object_storage {
    bucket_id    = yandex_storage_bucket.bucket.id
    batch_cutoff = "1"
    batch_size   = "1"
    create       = true  # trigger on object creation
    update       = true  # trigger on object update
    delete       = false # don't trigger on object deletion
  }
}

#######################################
### serverless functions - handler
#######################################

resource "yandex_iam_service_account" "handler_sa" {
  name = "handler-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "handler_sa_function_role" {
  folder_id = var.folder_id
  role      = "functions.functionInvoker"
  member    = "system:allUsers" // allow public access
}

resource "yandex_resourcemanager_folder_iam_member" "handler_sa_storage_role" {
  folder_id = var.folder_id
  role      = "storage.viewer"
  member    = "serviceAccount:${yandex_iam_service_account.handler_sa.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "handler_sa_queue_role" {
  folder_id = var.folder_id
  role      = "ymq.writer"
  member    = "serviceAccount:${yandex_iam_service_account.handler_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "handler_sa_keys" {
  service_account_id = yandex_iam_service_account.handler_sa.id
}

data "archive_file" "handler_zip" {
  type        = "zip"
  source_dir  = "${path.module}/serverless/handler"
  output_path = "${path.module}/serverless/handler.zip"
}

resource "yandex_function" "handler_fn" {
  name               = "handler-fn"
  runtime            = "golang121"
  entrypoint         = "main.Handler"
  memory             = 128
  execution_timeout  = 10
  service_account_id = yandex_iam_service_account.handler_sa.id
  folder_id          = var.folder_id
  user_hash          = "v1"

  environment = {
    S3_BUCKET  = yandex_storage_bucket.bucket.bucket
    SQS_URL    = yandex_message_queue.standard_queue.id
    SQS_REGION = yandex_message_queue.standard_queue.region_id

    REGION     = var.region
    ACCESS_KEY = yandex_iam_service_account_static_access_key.handler_sa_keys.access_key
    SECRET_KEY = yandex_iam_service_account_static_access_key.handler_sa_keys.secret_key
  }

  content {
    zip_filename = data.archive_file.handler_zip.output_path
  }
}

output "handler_fn_id" {
  value = yandex_function.handler_fn.id
}

#######################################
### serverless functions - sender
#######################################

resource "yandex_iam_service_account" "sender_sa" {
  name = "sender-sa"
}

resource "yandex_resourcemanager_folder_iam_member" "sender_sa_storage_role" {
  folder_id = var.folder_id
  role      = "storage.viewer"
  member    = "serviceAccount:${yandex_iam_service_account.sender_sa.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "sender_sa_ydb_role" {
  folder_id = var.folder_id
  role      = "ydb.editor"
  member    = "serviceAccount:${yandex_iam_service_account.sender_sa.id}"
}

resource "yandex_iam_service_account_static_access_key" "sender_sa_keys" {
  service_account_id = yandex_iam_service_account.sender_sa.id
}

data "archive_file" "sender_zip" {
  type        = "zip"
  source_dir  = "${path.module}/serverless/sender"
  output_path = "${path.module}/serverless/sender.zip"
}

resource "yandex_function" "sender_fn" {
  name               = "sender-fn"
  runtime            = "golang121"
  entrypoint         = "main.Handler"
  memory             = 128
  execution_timeout  = 10
  service_account_id = yandex_iam_service_account.sender_sa.id
  folder_id          = var.folder_id
  user_hash          = "v2"

  environment = {
    S3_BUCKET    = yandex_storage_bucket.bucket.bucket
    YDB_ENDPOINT = yandex_ydb_database_serverless.ydb.ydb_full_endpoint

    REGION     = var.region
    ACCESS_KEY = yandex_iam_service_account_static_access_key.sender_sa_keys.access_key
    SECRET_KEY = yandex_iam_service_account_static_access_key.sender_sa_keys.secret_key
  }

  content {
    zip_filename = data.archive_file.sender_zip.output_path
  }
}

resource "yandex_function_trigger" "sender_trigger" {
  name = "sender"
  function {
    id                 = yandex_function.sender_fn.id
    service_account_id = yandex_iam_service_account.sender_sa.id
  }

  message_queue {
    batch_cutoff       = "1"
    batch_size         = "1"
    service_account_id = yandex_iam_service_account.sender_sa.id
    queue_id           = yandex_message_queue.standard_queue.arn
  }
}

#######################################
### serverless functions - notifier
#######################################
