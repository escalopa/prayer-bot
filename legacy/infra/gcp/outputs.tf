output "dispatcher_url" {
  value = google_cloudfunctions2_function.dispatcher.service_config[0].uri
}

output "data_bucket_name" {
  value = google_storage_bucket.data.name
}

output "reminder_url" {
  value = google_cloudfunctions2_function.reminder.service_config[0].uri
}

output "loader_name" {
  value = google_cloudfunctions2_function.loader.name
}
