output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "alb_dns_name" {
  description = "ALB DNS name"
  value       = module.alb.alb_dns_name
}

output "ecr_repository_url" {
  description = "ECR repository URL"
  value       = module.ecr.repository_url
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "ECS service name"
  value       = module.ecs.service_name
}

output "dynamodb_table_name" {
  description = "DynamoDB table name"
  value       = module.dynamodb.table_name
}

output "eventbridge_bus_name" {
  description = "EventBridge bus name"
  value       = module.eventbridge.event_bus_name
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = var.enable_redis ? module.elasticache.endpoint : null
}

output "log_group_name" {
  description = "CloudWatch log group name"
  value       = module.cloudwatch.log_group_name
}

output "dashboard_name" {
  description = "CloudWatch dashboard name"
  value       = module.cloudwatch.dashboard_name
}

output "pipeline_name" {
  description = "CodePipeline name"
  value       = module.cicd.pipeline_name
}

output "service_discovery_dns" {
  description = "Service discovery DNS name"
  value       = module.ecs.service_discovery_dns_name
}

output "api_endpoint" {
  description = "API endpoint URL"
  value       = "https://${module.alb.alb_dns_name}"
}
