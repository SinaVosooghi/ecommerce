output "cluster_id" {
  description = "ECS cluster ID"
  value       = aws_ecs_cluster.main.id
}

output "cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "cluster_arn" {
  description = "ECS cluster ARN"
  value       = aws_ecs_cluster.main.arn
}

output "service_id" {
  description = "ECS service ID"
  value       = aws_ecs_service.main.id
}

output "service_name" {
  description = "ECS service name"
  value       = aws_ecs_service.main.name
}

output "task_definition_arn" {
  description = "Task definition ARN"
  value       = aws_ecs_task_definition.main.arn
}

output "task_definition_family" {
  description = "Task definition family"
  value       = aws_ecs_task_definition.main.family
}

output "security_group_id" {
  description = "ECS tasks security group ID"
  value       = aws_security_group.ecs_tasks.id
}

output "service_discovery_namespace_id" {
  description = "Service discovery namespace ID"
  value       = var.create_service_discovery_namespace ? aws_service_discovery_private_dns_namespace.main[0].id : var.service_discovery_namespace_id
}

output "service_discovery_service_arn" {
  description = "Service discovery service ARN"
  value       = aws_service_discovery_service.main.arn
}

output "service_discovery_dns_name" {
  description = "Service discovery DNS name"
  value       = "${var.service_name}.${var.project_name}.local"
}
