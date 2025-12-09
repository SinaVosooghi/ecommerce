variable "project_name" {
  description = "Name of the project"
  type        = string
}

variable "service_name" {
  description = "Name of the service"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

# Source Configuration
variable "source_provider" {
  description = "Source provider (CodeStarSourceConnection, CodeCommit)"
  type        = string
  default     = "CodeStarSourceConnection"
}

variable "codestar_connection_arn" {
  description = "CodeStar connection ARN for GitHub/GitLab"
  type        = string
  default     = ""
}

variable "repository_id" {
  description = "Full repository ID (e.g., owner/repo)"
  type        = string
  default     = ""
}

variable "repository_name" {
  description = "CodeCommit repository name"
  type        = string
  default     = ""
}

variable "branch_name" {
  description = "Branch to build"
  type        = string
  default     = "main"
}

# ECR Configuration
variable "ecr_repository_url" {
  description = "ECR repository URL"
  type        = string
}

variable "ecr_repository_arn" {
  description = "ECR repository ARN"
  type        = string
}

# ECS Configuration
variable "ecs_cluster_name" {
  description = "ECS cluster name"
  type        = string
}

variable "ecs_service_name" {
  description = "ECS service name"
  type        = string
}

# Build Configuration
variable "buildspec_file" {
  description = "Path to buildspec file"
  type        = string
  default     = "buildspec-build.yml"
}

variable "build_timeout" {
  description = "Build timeout in minutes"
  type        = number
  default     = 30
}

variable "build_compute_type" {
  description = "CodeBuild compute type"
  type        = string
  default     = "BUILD_GENERAL1_SMALL"
}

variable "build_image" {
  description = "CodeBuild image"
  type        = string
  default     = "aws/codebuild/amazonlinux2-x86_64-standard:5.0"
}

variable "log_retention_days" {
  description = "Log retention in days"
  type        = number
  default     = 30
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
