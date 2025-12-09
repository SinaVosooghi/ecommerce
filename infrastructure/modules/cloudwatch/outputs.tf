output "log_group_name" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.main.name
}

output "log_group_arn" {
  description = "CloudWatch log group ARN"
  value       = aws_cloudwatch_log_group.main.arn
}

output "dashboard_name" {
  description = "CloudWatch dashboard name"
  value       = var.create_dashboard ? aws_cloudwatch_dashboard.main[0].dashboard_name : null
}

output "cpu_alarm_arn" {
  description = "CPU alarm ARN"
  value       = var.enable_alarms ? aws_cloudwatch_metric_alarm.cpu_high[0].arn : null
}

output "memory_alarm_arn" {
  description = "Memory alarm ARN"
  value       = var.enable_alarms ? aws_cloudwatch_metric_alarm.memory_high[0].arn : null
}
