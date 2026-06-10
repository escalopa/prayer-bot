output "webhook_url" {
  value = google_cloudfunctions2_function.webhook_proxy.service_config[0].uri
}
