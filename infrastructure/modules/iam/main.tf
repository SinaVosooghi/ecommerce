# IAM Module - Roles and Policies for ECS

#------------------------------------------------------------------------------
# ECS Task Execution Role
#------------------------------------------------------------------------------
resource "aws_iam_role" "ecs_execution" {
  name = "${var.project_name}-${var.service_name}-${var.environment}-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.service_name}-${var.environment}-execution"
  })
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Policy for pulling from ECR and accessing secrets
resource "aws_iam_role_policy" "ecs_execution_custom" {
  name = "${var.project_name}-${var.service_name}-execution-custom"
  role = aws_iam_role.ecs_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat([
      {
        Sid    = "ECRAuth"
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken"
        ]
        Resource = "*"
      },
      {
        Sid    = "ECRPull"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage"
        ]
        Resource = var.ecr_repository_arn
      },
      {
        Sid    = "Logs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${var.log_group_arn}:*"
      },
      {
        Sid    = "SSMParameters"
        Effect = "Allow"
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/${var.project_name}/${var.environment}/*"
      }
    ],
    length(var.secrets_arns) > 0 ? [
      {
        Sid    = "SecretsManager"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = var.secrets_arns
      }
    ] : [])
  })
}

#------------------------------------------------------------------------------
# ECS Task Role (what the application can do)
#------------------------------------------------------------------------------
resource "aws_iam_role" "ecs_task" {
  name = "${var.project_name}-${var.service_name}-${var.environment}-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.service_name}-${var.environment}-task"
  })
}

data "aws_caller_identity" "current" {}

# DynamoDB Access
resource "aws_iam_role_policy" "dynamodb" {
  name = "${var.project_name}-${var.service_name}-dynamodb"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "DynamoDBAccess"
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:Scan",
          "dynamodb:BatchGetItem",
          "dynamodb:BatchWriteItem",
          "dynamodb:ConditionCheckItem"
        ]
        Resource = [
          var.dynamodb_table_arn,
          "${var.dynamodb_table_arn}/index/*"
        ]
      }
    ]
  })
}

# EventBridge Access
resource "aws_iam_role_policy" "eventbridge" {
  name = "${var.project_name}-${var.service_name}-eventbridge"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "EventBridgeAccess"
        Effect = "Allow"
        Action = [
          "events:PutEvents"
        ]
        Resource = var.event_bus_arn
      }
    ]
  })
}

# X-Ray Access
resource "aws_iam_role_policy" "xray" {
  count = var.enable_xray ? 1 : 0
  
  name = "${var.project_name}-${var.service_name}-xray"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "XRayAccess"
        Effect = "Allow"
        Action = [
          "xray:PutTraceSegments",
          "xray:PutTelemetryRecords",
          "xray:GetSamplingRules",
          "xray:GetSamplingTargets",
          "xray:GetSamplingStatisticSummaries"
        ]
        Resource = "*"
      }
    ]
  })
}

# CloudWatch Access
resource "aws_iam_role_policy" "cloudwatch" {
  name = "${var.project_name}-${var.service_name}-cloudwatch"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "CloudWatchMetrics"
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = var.cloudwatch_namespace
          }
        }
      }
    ]
  })
}

# Secrets Manager Access (runtime)
resource "aws_iam_role_policy" "secrets" {
  count = length(var.secrets_arns) > 0 ? 1 : 0

  name = "${var.project_name}-${var.service_name}-secrets"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SecretsAccess"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = var.secrets_arns
      }
    ]
  })
}

# ECS Exec for debugging (optional)
resource "aws_iam_role_policy" "ecs_exec" {
  count = var.enable_ecs_exec ? 1 : 0

  name = "${var.project_name}-${var.service_name}-ecs-exec"
  role = aws_iam_role.ecs_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ECSExec"
        Effect = "Allow"
        Action = [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel"
        ]
        Resource = "*"
      }
    ]
  })
}
