variable "project_name" {
  description = "Name of the project"
  type        = string
}

variable "service_name" {
  description = "Name of the service"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "ecr_repository_arn" {
  description = "ECR repository ARN"
  type        = string
}

variable "log_group_arn" {
  description = "CloudWatch log group ARN"
  type        = string
}

variable "dynamodb_table_arn" {
  description = "DynamoDB table ARN"
  type        = string
}

variable "event_bus_arn" {
  description = "EventBridge event bus ARN"
  type        = string
}

variable "secrets_arns" {
  description = "List of Secrets Manager secret ARNs"
  type        = list(string)
  default     = []
}

variable "cloudwatch_namespace" {
  description = "CloudWatch metrics namespace"
  type        = string
  default     = "CartService"
}

variable "enable_xray" {
  description = "Enable X-Ray tracing"
  type        = bool
  default     = true
}

variable "enable_ecs_exec" {
  description = "Enable ECS Exec for debugging"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
