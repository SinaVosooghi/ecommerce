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
  default     = "dev"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "eu-central-1"
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
  default     = "10.0.0.0/16"
}

variable "az_count" {
  description = "Number of availability zones"
  type        = number
  default     = 2
}

variable "enable_nat_gateway" {
  description = "Enable NAT Gateway"
  type        = bool
  default     = true
}

variable "enable_vpc_endpoints" {
  description = "Enable VPC endpoints"
  type        = bool
  default     = false
}

# ECS
variable "image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "latest"
}

# Features
variable "enable_xray" {
  description = "Enable X-Ray tracing"
  type        = bool
  default     = false
}

variable "certificate_arn" {
  description = "ACM certificate ARN"
  type        = string
  default     = ""
}

# CI/CD
variable "enable_cicd" {
  description = "Enable CI/CD pipeline"
  type        = bool
  default     = false
}

variable "source_provider" {
  description = "Source provider for CI/CD"
  type        = string
  default     = "CodeStarSourceConnection"
}

variable "codestar_connection_arn" {
  description = "CodeStar connection ARN (use if create_codestar_connection is false)"
  type        = string
  default     = ""
}

variable "create_codestar_connection" {
  description = "Create CodeStar connection via Terraform (requires manual authorization in AWS Console)"
  type        = bool
  default     = true
}

variable "codestar_connection_name" {
  description = "Name for the CodeStar connection (used when create_codestar_connection is true)"
  type        = string
  default     = ""
}

variable "codestar_provider_type" {
  description = "CodeStar connection provider type: GitHub, Bitbucket, GitLab, GitHubEnterpriseServer, GitLabSelfManaged"
  type        = string
  default     = "GitHub"
}

variable "codestar_host_arn" {
  description = "Host ARN for GitHub Enterprise Server or GitLab Self-Managed (required for enterprise providers)"
  type        = string
  default     = ""
}

variable "repository_id" {
  description = "Repository ID (owner/repo)"
  type        = string
  default     = ""
}
