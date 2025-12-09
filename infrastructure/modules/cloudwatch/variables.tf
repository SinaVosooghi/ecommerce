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

variable "log_retention_days" {
  description = "Log retention in days"
  type        = number
  default     = 30
}

# ECS Information
variable "ecs_cluster_name" {
  description = "ECS cluster name"
  type        = string
}

variable "ecs_service_name" {
  description = "ECS service name"
  type        = string
}

# ALB Information
variable "alb_arn_suffix" {
  description = "ALB ARN suffix for metrics"
  type        = string
  default     = ""
}

variable "target_group_arn_suffix" {
  description = "Target group ARN suffix for metrics"
  type        = string
  default     = ""
}

# DynamoDB Information
variable "dynamodb_table_name" {
  description = "DynamoDB table name"
  type        = string
  default     = ""
}

# Alarms
variable "enable_alarms" {
  description = "Enable CloudWatch alarms"
  type        = bool
  default     = true
}

variable "alarm_sns_topic_arns" {
  description = "List of SNS topic ARNs for alarm notifications"
  type        = list(string)
  default     = []
}

variable "cpu_alarm_threshold" {
  description = "CPU alarm threshold percentage"
  type        = number
  default     = 80
}

variable "memory_alarm_threshold" {
  description = "Memory alarm threshold percentage"
  type        = number
  default     = 80
}

variable "error_rate_threshold" {
  description = "5xx error rate threshold"
  type        = number
  default     = 10
}

variable "latency_threshold" {
  description = "P99 latency threshold in seconds"
  type        = number
  default     = 0.5
}

# Dashboard
variable "create_dashboard" {
  description = "Create CloudWatch dashboard"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
