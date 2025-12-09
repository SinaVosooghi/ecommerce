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

variable "image_tag_mutability" {
  description = "Image tag mutability (MUTABLE or IMMUTABLE)"
  type        = string
  default     = "MUTABLE"
}

variable "scan_on_push" {
  description = "Enable image scanning on push"
  type        = bool
  default     = true
}

variable "image_count_to_keep" {
  description = "Number of images to keep"
  type        = number
  default     = 30
}

variable "untagged_image_retention_days" {
  description = "Days to retain untagged images"
  type        = number
  default     = 7
}

variable "allowed_account_ids" {
  description = "List of AWS account IDs allowed to pull images"
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
