output "event_bus_name" {
  description = "Event bus name"
  value       = local.event_bus_name
}

output "event_bus_arn" {
  description = "Event bus ARN"
  value       = local.event_bus_arn
}

output "event_rule_arn" {
  description = "Event rule ARN"
  value       = aws_cloudwatch_event_rule.cart_events.arn
}

output "event_rule_name" {
  description = "Event rule name"
  value       = aws_cloudwatch_event_rule.cart_events.name
}

output "target_queue_arn" {
  description = "Target SQS queue ARN"
  value       = var.create_target_queue ? aws_sqs_queue.target[0].arn : null
}

output "target_queue_url" {
  description = "Target SQS queue URL"
  value       = var.create_target_queue ? aws_sqs_queue.target[0].url : null
}

output "dlq_arn" {
  description = "Dead letter queue ARN"
  value       = var.create_dlq ? aws_sqs_queue.dlq[0].arn : null
}

output "log_group_name" {
  description = "CloudWatch log group name for events"
  value       = var.enable_event_logging ? aws_cloudwatch_log_group.events[0].name : null
}
