# ElastiCache Module - Redis for Idempotency

locals {
  create_cluster = var.enabled
}

#------------------------------------------------------------------------------
# Subnet Group
#------------------------------------------------------------------------------
resource "aws_elasticache_subnet_group" "main" {
  count = local.create_cluster ? 1 : 0

  name        = "${var.project_name}-${var.environment}-redis"
  description = "Subnet group for ${var.project_name} Redis"
  subnet_ids  = var.private_subnet_ids

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-redis"
  })
}

#------------------------------------------------------------------------------
# Security Group
#------------------------------------------------------------------------------
resource "aws_security_group" "redis" {
  count = local.create_cluster ? 1 : 0

  name        = "${var.project_name}-${var.environment}-redis-sg"
  description = "Security group for Redis"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Redis from ECS tasks"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = var.allowed_security_group_ids
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-redis-sg"
  })
}

#------------------------------------------------------------------------------
# Parameter Group
#------------------------------------------------------------------------------
resource "aws_elasticache_parameter_group" "main" {
  count = local.create_cluster ? 1 : 0

  name        = "${var.project_name}-${var.environment}-redis-params"
  family      = var.parameter_group_family
  description = "Parameter group for ${var.project_name} Redis"

  parameter {
    name  = "maxmemory-policy"
    value = var.maxmemory_policy
  }

  tags = var.tags
}

#------------------------------------------------------------------------------
# Replication Group (Cluster Mode Disabled)
#------------------------------------------------------------------------------
resource "aws_elasticache_replication_group" "main" {
  count = local.create_cluster ? 1 : 0

  replication_group_id = "${var.project_name}-${var.environment}"
  description          = "Redis cluster for ${var.project_name} ${var.environment}"

  node_type            = var.node_type
  num_cache_clusters   = var.num_cache_clusters
  port                 = 6379
  parameter_group_name = aws_elasticache_parameter_group.main[0].name
  subnet_group_name    = aws_elasticache_subnet_group.main[0].name
  security_group_ids   = [aws_security_group.redis[0].id]

  engine               = "redis"
  engine_version       = var.engine_version
  
  automatic_failover_enabled = var.num_cache_clusters > 1 ? true : false
  multi_az_enabled           = var.num_cache_clusters > 1 ? var.multi_az_enabled : false

  at_rest_encryption_enabled = true
  transit_encryption_enabled = var.transit_encryption_enabled
  auth_token                 = var.transit_encryption_enabled ? var.auth_token : null

  snapshot_retention_limit = var.snapshot_retention_limit
  snapshot_window          = var.snapshot_window
  maintenance_window       = var.maintenance_window

  apply_immediately = var.apply_immediately

  auto_minor_version_upgrade = true

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-redis"
  })
}
