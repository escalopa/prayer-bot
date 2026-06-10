terraform {
  required_version = ">= 1.6.0"

  backend "gcs" {}

  required_providers {
    archive = {
      source  = "hashicorp/archive"
      version = ">= 2.4.0"
    }

    google = {
      source  = "hashicorp/google"
      version = ">= 5.0.0"
    }
  }
}
