variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "ecommerce"
}

variable "service_name" {
  description = "Name of the service"
  type        = string
  default     = "cart-service"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "prod"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

# VPC
variable "existing_vpc_id" {
  description = "Existing VPC ID (leave empty to create new)"
  type        = string
  default     = ""
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.2.0.0/16"
}

# ECS
variable "image_tag" {
  description = "Docker image tag"
  type        = string
}

# Redis
variable "enable_redis" {
  description = "Enable Redis for idempotency"
  type        = bool
  default     = true
}

variable "redis_auth_token" {
  description = "Redis auth token"
  type        = string
  default     = ""
  sensitive   = true
}

# SSL
variable "certificate_arn" {
  description = "ACM certificate ARN"
  type        = string
}

# Alerting
variable "alarm_sns_topic_arns" {
  description = "SNS topic ARNs for alarms"
  type        = list(string)
  default     = []
}

# CI/CD
variable "source_provider" {
  description = "Source provider for CI/CD"
  type        = string
  default     = "CodeStarSourceConnection"
}

variable "codestar_connection_arn" {
  description = "CodeStar connection ARN"
  type        = string
}

variable "repository_id" {
  description = "Repository ID (owner/repo)"
  type        = string
}

# Secrets
variable "secrets" {
  description = "Map of secrets to create"
  type = map(object({
    description = string
    value       = string
  }))
  default   = {}
  sensitive = true
}
