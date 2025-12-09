# Production Environment Configuration
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "ecommerce-terraform-state"
    key            = "prod/cart-service/terraform.tfstate"
    region         = "us-east-1"
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
  az_count             = 3 # HA across 3 AZs
  enable_nat_gateway   = true
  single_nat_gateway   = false # NAT per AZ for HA
  enable_vpc_endpoints = true
  tags                 = local.common_tags
}

#------------------------------------------------------------------------------
# ECR
#------------------------------------------------------------------------------
module "ecr" {
  source = "../../modules/ecr"

  project_name         = var.project_name
  service_name         = var.service_name
  environment          = var.environment
  image_tag_mutability = "IMMUTABLE" # Immutable tags in prod
  scan_on_push         = true
  image_count_to_keep  = 50
  tags                 = local.common_tags
}

#------------------------------------------------------------------------------
# CloudWatch
#------------------------------------------------------------------------------
module "cloudwatch" {
  source = "../../modules/cloudwatch"

  project_name            = var.project_name
  service_name            = var.service_name
  environment             = var.environment
  aws_region              = var.aws_region
  log_retention_days      = 90
  ecs_cluster_name        = module.ecs.cluster_name
  ecs_service_name        = module.ecs.service_name
  alb_arn_suffix          = split("/", module.alb.alb_arn)[2]
  target_group_arn_suffix = split(":", module.alb.target_group_arn)[5]
  dynamodb_table_name     = module.dynamodb.table_name
  enable_alarms           = true
  alarm_sns_topic_arns    = var.alarm_sns_topic_arns
  cpu_alarm_threshold     = 70
  memory_alarm_threshold  = 80
  latency_threshold       = 0.5
  create_dashboard        = true
  tags                    = local.common_tags
}

#------------------------------------------------------------------------------
# ALB
#------------------------------------------------------------------------------
module "alb" {
  source = "../../modules/alb"

  project_name               = var.project_name
  service_name               = var.service_name
  environment                = var.environment
  vpc_id                     = module.vpc.vpc_id
  public_subnet_ids          = module.vpc.public_subnet_ids
  certificate_arn            = var.certificate_arn
  health_check_path          = "/health"
  enable_deletion_protection = true
  deregistration_delay       = 60
  tags                       = local.common_tags
}

#------------------------------------------------------------------------------
# DynamoDB
#------------------------------------------------------------------------------
module "dynamodb" {
  source = "../../modules/dynamodb"

  project_name                  = var.project_name
  service_name                  = var.service_name
  environment                   = var.environment
  billing_mode                  = "PROVISIONED"
  read_capacity                 = 100
  write_capacity                = 50
  read_max_capacity             = 1000
  write_max_capacity            = 500
  enable_autoscaling            = true
  target_utilization            = 70
  enable_ttl                    = true
  enable_point_in_time_recovery = true
  tags                          = local.common_tags
}

#------------------------------------------------------------------------------
# ElastiCache (for idempotency in production)
#------------------------------------------------------------------------------
module "elasticache" {
  source = "../../modules/elasticache"

  project_name               = var.project_name
  environment                = var.environment
  enabled                    = var.enable_redis
  vpc_id                     = module.vpc.vpc_id
  private_subnet_ids         = module.vpc.private_subnet_ids
  allowed_security_group_ids = [module.ecs.security_group_id]
  node_type                  = "cache.t3.small"
  num_cache_clusters         = 2
  multi_az_enabled           = true
  transit_encryption_enabled = true
  auth_token                 = var.redis_auth_token
  snapshot_retention_limit   = 7
  tags                       = local.common_tags
}

#------------------------------------------------------------------------------
# EventBridge
#------------------------------------------------------------------------------
module "eventbridge" {
  source = "../../modules/eventbridge"

  project_name           = var.project_name
  service_name           = var.service_name
  environment            = var.environment
  aws_region             = var.aws_region
  create_event_bus       = true
  create_target_queue    = true
  create_dlq             = true
  enable_event_logging   = true
  enable_archive         = true
  archive_retention_days = 90
  log_retention_days     = 90
  tags                   = local.common_tags
}

#------------------------------------------------------------------------------
# Secrets
#------------------------------------------------------------------------------
module "secrets" {
  source = "../../modules/secrets"

  project_name   = var.project_name
  service_name   = var.service_name
  environment    = var.environment
  create_kms_key = true
  task_role_arn  = module.iam.task_role_arn
  secrets        = var.secrets
  tags           = local.common_tags
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
  secrets_arns       = values(module.secrets.secret_arns)
  enable_xray        = true
  enable_ecs_exec    = false # Disable in prod
  tags               = local.common_tags
}

#------------------------------------------------------------------------------
# ECS
#------------------------------------------------------------------------------
module "ecs" {
  source = "../../modules/ecs"

  project_name           = var.project_name
  service_name           = var.service_name
  environment            = var.environment
  aws_region             = var.aws_region
  vpc_id                 = module.vpc.vpc_id
  private_subnet_ids     = module.vpc.private_subnet_ids
  ecr_repository_url     = module.ecr.repository_url
  image_tag              = var.image_tag
  container_port         = 8080
  task_cpu               = 1024
  task_memory            = 2048
  desired_count          = 3
  execution_role_arn     = module.iam.execution_role_arn
  task_role_arn          = module.iam.task_role_arn
  target_group_arn       = module.alb.target_group_arn
  alb_security_group_id  = module.alb.security_group_id
  log_group_name         = module.cloudwatch.log_group_name
  use_fargate_spot       = false # Use regular Fargate in prod
  enable_execute_command = false
  enable_autoscaling     = true
  min_capacity           = 3
  max_capacity           = 20
  cpu_target_value       = 70
  memory_target_value    = 80
  enable_container_insights = true

  environment_variables = {
    ENV_NAME             = var.environment
    LOG_LEVEL            = "info"
    AWS_REGION           = var.aws_region
    DYNAMODB_TABLE       = module.dynamodb.table_name
    EVENTBRIDGE_ENABLED  = "true"
    EVENTBRIDGE_BUS_NAME = module.eventbridge.event_bus_name
    AWS_XRAY_ENABLED     = "true"
    REDIS_ENABLED        = tostring(var.enable_redis)
    REDIS_URL            = var.enable_redis ? "redis://${module.elasticache.endpoint}:6379" : ""
  }

  secrets = var.enable_redis && var.redis_auth_token != "" ? {
    REDIS_AUTH_TOKEN = module.secrets.secret_arns["redis-auth-token"]
  } : {}

  tags = local.common_tags
}

#------------------------------------------------------------------------------
# CI/CD
#------------------------------------------------------------------------------
module "cicd" {
  source = "../../modules/cicd"

  project_name            = var.project_name
  service_name            = var.service_name
  aws_region              = var.aws_region
  source_provider         = var.source_provider
  codestar_connection_arn = var.codestar_connection_arn
  repository_id           = var.repository_id
  branch_name             = "main"
  ecr_repository_url      = module.ecr.repository_url
  ecr_repository_arn      = module.ecr.repository_arn
  ecs_cluster_name        = module.ecs.cluster_name
  ecs_service_name        = module.ecs.service_name
  tags                    = local.common_tags
}
