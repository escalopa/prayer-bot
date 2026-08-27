locals {
  global_services = {
    webhook  = google_cloud_run_v2_service.webhook.name
    dispatch = google_cloud_run_v2_service.dispatch.name
    sender   = google_cloud_run_v2_service.sender.name
  }
}

resource "google_monitoring_alert_policy" "cloud_run_5xx" {
  for_each     = local.global_services
  display_name = "Global prayer ${var.environment}: ${each.key} 5xx responses"
  combiner     = "OR"
  severity     = "ERROR"

  notification_channels = var.alert_notification_channels

  conditions {
    display_name = "More than five 5xx responses in five minutes"

    condition_threshold {
      filter = "metric.type=\"run.googleapis.com/request_count\" AND resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${each.value}\" AND metric.labels.response_code_class=\"5xx\""

      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
      }

      comparison      = "COMPARISON_GT"
      threshold_value = 5
      duration        = "300s"

      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Investigate the Cloud Run logs for the ${each.key} service and confirm the latest revision is ready. For sender failures, also inspect the notifications Cloud Tasks queue."
    mime_type = "text/markdown"
  }

  depends_on = [google_project_service.required]
}

resource "google_monitoring_alert_policy" "task_retries" {
  display_name          = "Global prayer ${var.environment}: notification task retries"
  combiner              = "OR"
  severity              = "WARNING"
  notification_channels = var.alert_notification_channels

  conditions {
    display_name = "More than ten failed notification task attempts in five minutes"

    condition_threshold {
      filter = "metric.type=\"cloudtasks.googleapis.com/queue/task_attempt_count\" AND resource.type=\"cloud_tasks_queue\" AND resource.labels.queue_id=\"${google_cloud_tasks_queue.notifications.name}\" AND metric.labels.response_code!=\"ok\""

      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
      }

      comparison      = "COMPARISON_GT"
      threshold_value = 10
      duration        = "300s"

      trigger {
        count = 1
      }
    }
  }

  documentation {
    content   = "Inspect the notifications Cloud Tasks queue and sender Cloud Run logs. Cloud Tasks retries HTTP tasks up to the configured maximum and does not use a dead-letter queue in this deployment."
    mime_type = "text/markdown"
  }

  depends_on = [google_project_service.required]
}
