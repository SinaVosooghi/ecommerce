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

variable "billing_mode" {
  description = "DynamoDB billing mode (PROVISIONED or PAY_PER_REQUEST)"
  type        = string
  default     = "PAY_PER_REQUEST"

  validation {
    condition     = contains(["PROVISIONED", "PAY_PER_REQUEST"], var.billing_mode)
    error_message = "Billing mode must be PROVISIONED or PAY_PER_REQUEST."
  }
}

# Provisioned capacity (used when billing_mode = PROVISIONED)
variable "read_capacity" {
  description = "Read capacity units"
  type        = number
  default     = 10
}

variable "write_capacity" {
  description = "Write capacity units"
  type        = number
  default     = 5
}

variable "read_max_capacity" {
  description = "Maximum read capacity units for auto scaling"
  type        = number
  default     = 100
}

variable "write_max_capacity" {
  description = "Maximum write capacity units for auto scaling"
  type        = number
  default     = 50
}

# GSI capacity
variable "gsi_read_capacity" {
  description = "GSI read capacity units"
  type        = number
  default     = 5
}

variable "gsi_write_capacity" {
  description = "GSI write capacity units"
  type        = number
  default     = 5
}

variable "gsi_read_max_capacity" {
  description = "GSI maximum read capacity units"
  type        = number
  default     = 50
}

variable "gsi_write_max_capacity" {
  description = "GSI maximum write capacity units"
  type        = number
  default     = 25
}

variable "enable_autoscaling" {
  description = "Enable auto scaling (for PROVISIONED mode)"
  type        = bool
  default     = true
}

variable "target_utilization" {
  description = "Target utilization percentage for auto scaling"
  type        = number
  default     = 70
}

variable "enable_ttl" {
  description = "Enable TTL"
  type        = bool
  default     = true
}

variable "enable_point_in_time_recovery" {
  description = "Enable point-in-time recovery"
  type        = bool
  default     = true
}

variable "enable_streams" {
  description = "Enable DynamoDB streams"
  type        = bool
  default     = false
}

variable "stream_view_type" {
  description = "Stream view type (KEYS_ONLY, NEW_IMAGE, OLD_IMAGE, NEW_AND_OLD_IMAGES)"
  type        = string
  default     = "NEW_AND_OLD_IMAGES"
}

variable "kms_key_arn" {
  description = "KMS key ARN for encryption (optional, uses AWS managed key if not specified)"
  type        = string
  default     = null
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
