variable "project_name" {
  description = "Name of the project"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "enabled" {
  description = "Enable ElastiCache Redis"
  type        = bool
  default     = true
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "List of private subnet IDs"
  type        = list(string)
}

variable "allowed_security_group_ids" {
  description = "Security group IDs allowed to access Redis"
  type        = list(string)
}

variable "node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.micro"
}

variable "num_cache_clusters" {
  description = "Number of cache clusters (nodes)"
  type        = number
  default     = 1
}

variable "engine_version" {
  description = "Redis engine version"
  type        = string
  default     = "7.0"
}

variable "parameter_group_family" {
  description = "Parameter group family"
  type        = string
  default     = "redis7"
}

variable "maxmemory_policy" {
  description = "Redis maxmemory policy"
  type        = string
  default     = "volatile-lru"
}

variable "multi_az_enabled" {
  description = "Enable Multi-AZ"
  type        = bool
  default     = false
}

variable "transit_encryption_enabled" {
  description = "Enable encryption in transit"
  type        = bool
  default     = true
}

variable "auth_token" {
  description = "Auth token for Redis (required if transit_encryption_enabled)"
  type        = string
  default     = null
  sensitive   = true
}

variable "snapshot_retention_limit" {
  description = "Number of days to retain snapshots"
  type        = number
  default     = 7
}

variable "snapshot_window" {
  description = "Daily snapshot window"
  type        = string
  default     = "05:00-06:00"
}

variable "maintenance_window" {
  description = "Weekly maintenance window"
  type        = string
  default     = "sun:06:00-sun:07:00"
}

variable "apply_immediately" {
  description = "Apply changes immediately"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
