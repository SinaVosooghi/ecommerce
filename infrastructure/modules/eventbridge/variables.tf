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

variable "create_event_bus" {
  description = "Create a new event bus"
  type        = bool
  default     = true
}

variable "event_bus_name" {
  description = "Name of existing event bus (if not creating new)"
  type        = string
  default     = "default"
}

variable "event_source" {
  description = "Event source pattern"
  type        = string
  default     = "cart-service"
}

variable "event_types" {
  description = "List of event types to capture"
  type        = list(string)
  default = [
    "cart.created",
    "cart.item_added",
    "cart.item_removed",
    "cart.item_updated",
    "cart.cleared",
    "cart.abandoned"
  ]
}

# SQS Configuration
variable "create_target_queue" {
  description = "Create a target SQS queue"
  type        = bool
  default     = true
}

variable "create_dlq" {
  description = "Create a dead-letter queue"
  type        = bool
  default     = true
}

variable "visibility_timeout" {
  description = "SQS visibility timeout in seconds"
  type        = number
  default     = 300
}

variable "message_retention_seconds" {
  description = "SQS message retention in seconds"
  type        = number
  default     = 1209600 # 14 days
}

variable "dlq_retention_seconds" {
  description = "DLQ message retention in seconds"
  type        = number
  default     = 1209600 # 14 days
}

variable "max_receive_count" {
  description = "Max receive count before sending to DLQ"
  type        = number
  default     = 3
}

# Retry Policy
variable "max_event_age" {
  description = "Maximum event age in seconds"
  type        = number
  default     = 3600
}

variable "max_retry_attempts" {
  description = "Maximum retry attempts"
  type        = number
  default     = 3
}

# Logging
variable "enable_event_logging" {
  description = "Enable event logging to CloudWatch"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "Log retention in days"
  type        = number
  default     = 30
}

# Archive
variable "enable_archive" {
  description = "Enable event archive"
  type        = bool
  default     = true
}

variable "archive_retention_days" {
  description = "Archive retention in days"
  type        = number
  default     = 30
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
