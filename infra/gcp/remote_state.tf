data "terraform_remote_state" "yc" {
  count   = var.dual_write ? 1 : 0
  backend = "s3"

  config = {
    endpoints = {
      s3 = "https://storage.yandexcloud.net"
    }
    bucket                      = var.yc_tfstate_bucket
    key                         = var.yc_tfstate_key
    region                      = "ru-central1"
    workspace_key_prefix        = ""
    skip_region_validation      = true
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_s3_checksum            = true
  }

  workspace = var.environment
}
