output "table_name" {
  description = "DynamoDB table name"
  value       = aws_dynamodb_table.main.name
}

output "table_arn" {
  description = "DynamoDB table ARN"
  value       = aws_dynamodb_table.main.arn
}

output "table_id" {
  description = "DynamoDB table ID"
  value       = aws_dynamodb_table.main.id
}

output "stream_arn" {
  description = "DynamoDB stream ARN"
  value       = var.enable_streams ? aws_dynamodb_table.main.stream_arn : null
}

output "stream_label" {
  description = "DynamoDB stream label"
  value       = var.enable_streams ? aws_dynamodb_table.main.stream_label : null
}
