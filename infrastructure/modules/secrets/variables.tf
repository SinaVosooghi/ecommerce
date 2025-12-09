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

variable "secrets" {
  description = "Map of secrets to create (keys are not sensitive, values are)"
  type = map(object({
    description = string
    value       = string
  }))
  default = {}
}

variable "recovery_window_days" {
  description = "Number of days before permanent deletion"
  type        = number
  default     = 7
}

variable "create_kms_key" {
  description = "Create a KMS key for secrets encryption"
  type        = bool
  default     = false
}

variable "task_role_arn" {
  description = "ECS task role ARN for KMS access"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
