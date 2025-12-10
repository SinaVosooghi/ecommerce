# Development Environment Configuration
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "ecommerce-terraform-state-762233763891"
    key            = "dev/cart-service/terraform.tfstate"
    region         = "eu-central-1"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "Terraform"
      Service     = var.service_name
    }
  }
}

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    Service     = var.service_name
  }
}

#------------------------------------------------------------------------------
# VPC
#------------------------------------------------------------------------------
module "vpc" {
  source = "../../modules/vpc"

  project_name         = var.project_name
  environment          = var.environment
  aws_region           = var.aws_region
  vpc_id               = var.existing_vpc_id
  vpc_cidr             = var.vpc_cidr
  az_count             = var.az_count
  enable_nat_gateway   = var.enable_nat_gateway
  single_nat_gateway   = true # Cost saving for dev
  enable_vpc_endpoints = var.enable_vpc_endpoints
  tags                 = local.common_tags
}

#------------------------------------------------------------------------------
# ECR
#------------------------------------------------------------------------------
module "ecr" {
  source = "../../modules/ecr"

  project_name    = var.project_name
  service_name    = var.service_name
  environment     = var.environment
  scan_on_push    = true
  tags            = local.common_tags
}

#------------------------------------------------------------------------------
# CloudWatch
#------------------------------------------------------------------------------
module "cloudwatch" {
  source = "../../modules/cloudwatch"

  project_name         = var.project_name
  service_name         = var.service_name
  environment          = var.environment
  aws_region           = var.aws_region
  log_retention_days   = 14
  ecs_cluster_name     = module.ecs.cluster_name
  ecs_service_name     = module.ecs.service_name
  alb_arn_suffix       = split("/", module.alb.alb_arn)[2]
  target_group_arn_suffix = split(":", module.alb.target_group_arn)[5]
  dynamodb_table_name  = module.dynamodb.table_name
  enable_alarms        = false # Disable alarms in dev
  create_dashboard     = true
  tags                 = local.common_tags
}

#------------------------------------------------------------------------------
# ALB
#------------------------------------------------------------------------------
module "alb" {
  source = "../../modules/alb"

  project_name      = var.project_name
  service_name      = var.service_name
  environment       = var.environment
  vpc_id            = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
  certificate_arn   = var.certificate_arn
  health_check_path = "/health"
  tags              = local.common_tags
}

#------------------------------------------------------------------------------
# DynamoDB
#------------------------------------------------------------------------------
module "dynamodb" {
  source = "../../modules/dynamodb"

  project_name   = var.project_name
  service_name   = var.service_name
  environment    = var.environment
  billing_mode   = "PAY_PER_REQUEST" # On-demand for dev
  enable_ttl     = true
  enable_point_in_time_recovery = false # Disable for dev
  tags           = local.common_tags
}

#------------------------------------------------------------------------------
# EventBridge
#------------------------------------------------------------------------------
module "eventbridge" {
  source = "../../modules/eventbridge"

  project_name         = var.project_name
  service_name         = var.service_name
  environment          = var.environment
  aws_region           = var.aws_region
  create_event_bus     = true
  create_target_queue  = true
  create_dlq           = true
  enable_event_logging = true
  log_retention_days   = 14
  tags                 = local.common_tags
}

#------------------------------------------------------------------------------
# IAM
#------------------------------------------------------------------------------
module "iam" {
  source = "../../modules/iam"

  project_name       = var.project_name
  service_name       = var.service_name
  environment        = var.environment
  aws_region         = var.aws_region
  ecr_repository_arn = module.ecr.repository_arn
  log_group_arn      = module.cloudwatch.log_group_arn
  dynamodb_table_arn = module.dynamodb.table_arn
  event_bus_arn      = module.eventbridge.event_bus_arn
  enable_xray        = var.enable_xray
  enable_ecs_exec    = true # Enable for dev debugging
  tags               = local.common_tags
}

#------------------------------------------------------------------------------
# ECS
#------------------------------------------------------------------------------
module "ecs" {
  source = "../../modules/ecs"

  project_name          = var.project_name
  service_name          = var.service_name
  environment           = var.environment
  aws_region            = var.aws_region
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecr_repository_url    = module.ecr.repository_url
  image_tag             = var.image_tag
  container_port        = 8080
  task_cpu              = 256
  task_memory           = 512
  desired_count         = 1
  execution_role_arn    = module.iam.execution_role_arn
  task_role_arn         = module.iam.task_role_arn
  target_group_arn      = module.alb.target_group_arn
  alb_security_group_id = module.alb.security_group_id
  log_group_name        = module.cloudwatch.log_group_name
  use_fargate_spot      = true # Cost saving for dev
  enable_execute_command = true
  enable_autoscaling    = false # Disable for dev
  enable_container_insights = true

  environment_variables = {
    ENV_NAME           = var.environment
    LOG_LEVEL          = "debug"
    AWS_REGION         = var.aws_region
    DYNAMODB_TABLE     = module.dynamodb.table_name
    EVENTBRIDGE_ENABLED = "true"
    EVENTBRIDGE_BUS_NAME = module.eventbridge.event_bus_name
    AWS_XRAY_ENABLED   = tostring(var.enable_xray)
  }

  tags = local.common_tags
}

#------------------------------------------------------------------------------
# CodeStar Connection (for CI/CD)
#------------------------------------------------------------------------------
locals {
  # AWS CodeStar connection names must be <= 32 characters
  codestar_connection_name = var.codestar_connection_name != "" ? var.codestar_connection_name : "${var.project_name}-${var.service_name}-${var.environment}"
}

resource "aws_codestarconnections_connection" "main" {
  count = var.enable_cicd && var.create_codestar_connection ? 1 : 0

  name          = local.codestar_connection_name
  provider_type = var.codestar_provider_type

  # For GitHub Enterprise Server or GitLab Self-Managed
  host_arn = var.codestar_host_arn != "" ? var.codestar_host_arn : null

  tags = merge(local.common_tags, {
    Name = local.codestar_connection_name
  })
}

# Determine which connection ARN to use
locals {
  codestar_connection_arn_to_use = var.enable_cicd ? (
    var.create_codestar_connection && length(aws_codestarconnections_connection.main) > 0
      ? aws_codestarconnections_connection.main[0].arn
      : var.codestar_connection_arn
  ) : ""
}

#------------------------------------------------------------------------------
# CI/CD (optional for dev)
#------------------------------------------------------------------------------
module "cicd" {
  count  = var.enable_cicd ? 1 : 0
  source = "../../modules/cicd"

  project_name            = var.project_name
  service_name            = var.service_name
  aws_region              = var.aws_region
  source_provider         = var.source_provider
  codestar_connection_arn = local.codestar_connection_arn_to_use
  repository_id           = var.repository_id
  branch_name             = "develop"
  buildspec_file          = "services/cart-service/buildspec-build.yml"
  ecr_repository_url      = module.ecr.repository_url
  ecr_repository_arn      = module.ecr.repository_arn
  ecs_cluster_name        = module.ecs.cluster_name
  ecs_service_name        = module.ecs.service_name
  tags                    = local.common_tags
}
