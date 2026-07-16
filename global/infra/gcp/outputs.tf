output "artifact_registry_repository" {
  value = google_artifact_registry_repository.global.repository_id
}

output "webhook_url" {
  value = google_cloud_run_v2_service.webhook.uri
}

output "dispatch_url" {
  value = google_cloud_run_v2_service.dispatch.uri
}

output "sender_url" {
  value = google_cloud_run_v2_service.sender.uri
}

output "maps_secret_id" {
  value = google_secret_manager_secret.maps_api_key.secret_id
}

output "maps_api_key_id" {
  value = google_apikeys_key.maps.uid
}
