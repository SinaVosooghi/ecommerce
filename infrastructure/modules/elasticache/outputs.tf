output "endpoint" {
  description = "Redis primary endpoint"
  value       = var.enabled ? aws_elasticache_replication_group.main[0].primary_endpoint_address : null
}

output "reader_endpoint" {
  description = "Redis reader endpoint"
  value       = var.enabled ? aws_elasticache_replication_group.main[0].reader_endpoint_address : null
}

output "port" {
  description = "Redis port"
  value       = var.enabled ? 6379 : null
}

output "security_group_id" {
  description = "Redis security group ID"
  value       = var.enabled ? aws_security_group.redis[0].id : null
}

output "replication_group_id" {
  description = "Replication group ID"
  value       = var.enabled ? aws_elasticache_replication_group.main[0].id : null
}

output "connection_url" {
  description = "Redis connection URL (without auth)"
  value       = var.enabled ? "redis://${aws_elasticache_replication_group.main[0].primary_endpoint_address}:6379" : null
}
