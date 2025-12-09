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

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "public_subnet_ids" {
  description = "List of public subnet IDs"
  type        = list(string)
}

variable "internal" {
  description = "Whether the ALB is internal"
  type        = bool
  default     = false
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS"
  type        = string
  default     = ""
}

variable "ssl_policy" {
  description = "SSL policy for HTTPS listener"
  type        = string
  default     = "ELBSecurityPolicy-TLS13-1-2-2021-06"
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access the ALB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "target_port" {
  description = "Target port for the target group"
  type        = number
  default     = 8080
}

variable "path_patterns" {
  description = "Path patterns for listener rule"
  type        = list(string)
  default     = ["/*"]
}

variable "listener_rule_priority" {
  description = "Priority for the listener rule"
  type        = number
  default     = 100
}

# Health Check
variable "health_check_path" {
  description = "Health check path"
  type        = string
  default     = "/health"
}

variable "health_check_matcher" {
  description = "Health check response codes"
  type        = string
  default     = "200"
}

variable "health_check_interval" {
  description = "Health check interval in seconds"
  type        = number
  default     = 30
}

variable "health_check_timeout" {
  description = "Health check timeout in seconds"
  type        = number
  default     = 5
}

variable "healthy_threshold" {
  description = "Healthy threshold count"
  type        = number
  default     = 2
}

variable "unhealthy_threshold" {
  description = "Unhealthy threshold count"
  type        = number
  default     = 3
}

variable "deregistration_delay" {
  description = "Deregistration delay in seconds"
  type        = number
  default     = 30
}

# Stickiness
variable "enable_stickiness" {
  description = "Enable session stickiness"
  type        = bool
  default     = false
}

variable "stickiness_duration" {
  description = "Stickiness duration in seconds"
  type        = number
  default     = 3600
}

# Access Logs
variable "enable_access_logs" {
  description = "Enable access logs"
  type        = bool
  default     = false
}

variable "access_logs_bucket" {
  description = "S3 bucket for access logs"
  type        = string
  default     = ""
}

# Other
variable "idle_timeout" {
  description = "Idle timeout in seconds"
  type        = number
  default     = 60
}

variable "enable_deletion_protection" {
  description = "Enable deletion protection"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
