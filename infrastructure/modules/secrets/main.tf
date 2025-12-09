# Secrets Manager Module

#------------------------------------------------------------------------------
# Secrets
#------------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "main" {
  for_each = var.secrets

  name        = "${var.project_name}/${var.environment}/${each.key}"
  description = each.value.description

  recovery_window_in_days = var.recovery_window_days

  tags = merge(var.tags, {
    Name        = "${var.project_name}-${var.environment}-${each.key}"
    Service     = var.service_name
    Environment = var.environment
  })
}

resource "aws_secretsmanager_secret_version" "main" {
  for_each = var.secrets

  secret_id     = aws_secretsmanager_secret.main[each.key].id
  secret_string = each.value.value
}

#------------------------------------------------------------------------------
# KMS Key (optional)
#------------------------------------------------------------------------------
resource "aws_kms_key" "secrets" {
  count = var.create_kms_key ? 1 : 0

  description             = "KMS key for ${var.project_name} ${var.environment} secrets"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow ECS Task Role"
        Effect = "Allow"
        Principal = {
          AWS = var.task_role_arn
        }
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ]
        Resource = "*"
      }
    ]
  })

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-secrets-key"
  })
}

resource "aws_kms_alias" "secrets" {
  count = var.create_kms_key ? 1 : 0

  name          = "alias/${var.project_name}-${var.environment}-secrets"
  target_key_id = aws_kms_key.secrets[0].key_id
}

data "aws_caller_identity" "current" {}
