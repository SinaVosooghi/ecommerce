# EventBridge Module - Event Bus and Rules

#------------------------------------------------------------------------------
# Event Bus
#------------------------------------------------------------------------------
resource "aws_cloudwatch_event_bus" "main" {
  count = var.create_event_bus ? 1 : 0

  name = "${var.project_name}-${var.environment}"

  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}"
  })
}

locals {
  event_bus_name = var.create_event_bus ? aws_cloudwatch_event_bus.main[0].name : var.event_bus_name
  event_bus_arn  = var.create_event_bus ? aws_cloudwatch_event_bus.main[0].arn : "arn:aws:events:${var.aws_region}:${data.aws_caller_identity.current.account_id}:event-bus/${var.event_bus_name}"
}

data "aws_caller_identity" "current" {}

#------------------------------------------------------------------------------
# Event Rules
#------------------------------------------------------------------------------
resource "aws_cloudwatch_event_rule" "cart_events" {
  name           = "${var.project_name}-${var.service_name}-events-${var.environment}"
  description    = "Capture cart events from ${var.service_name}"
  event_bus_name = local.event_bus_name

  event_pattern = jsonencode({
    source = [var.event_source]
    "detail-type" = var.event_types
  })

  tags = merge(var.tags, {
    Name    = "${var.project_name}-${var.service_name}-events"
    Service = var.service_name
  })
}

#------------------------------------------------------------------------------
# SQS Dead Letter Queue
#------------------------------------------------------------------------------
resource "aws_sqs_queue" "dlq" {
  count = var.create_dlq ? 1 : 0

  name = "${var.project_name}-${var.service_name}-events-dlq-${var.environment}"

  message_retention_seconds = var.dlq_retention_seconds

  tags = merge(var.tags, {
    Name    = "${var.project_name}-${var.service_name}-events-dlq"
    Service = var.service_name
  })
}

#------------------------------------------------------------------------------
# SQS Target Queue (for downstream consumers)
#------------------------------------------------------------------------------
resource "aws_sqs_queue" "target" {
  count = var.create_target_queue ? 1 : 0

  name                       = "${var.project_name}-${var.service_name}-events-${var.environment}"
  visibility_timeout_seconds = var.visibility_timeout
  message_retention_seconds  = var.message_retention_seconds

  redrive_policy = var.create_dlq ? jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq[0].arn
    maxReceiveCount     = var.max_receive_count
  }) : null

  tags = merge(var.tags, {
    Name    = "${var.project_name}-${var.service_name}-events"
    Service = var.service_name
  })
}

resource "aws_sqs_queue_policy" "target" {
  count = var.create_target_queue ? 1 : 0

  queue_url = aws_sqs_queue.target[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowEventBridge"
        Effect    = "Allow"
        Principal = { Service = "events.amazonaws.com" }
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.target[0].arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = aws_cloudwatch_event_rule.cart_events.arn
          }
        }
      }
    ]
  })
}

#------------------------------------------------------------------------------
# Event Target - SQS
#------------------------------------------------------------------------------
resource "aws_cloudwatch_event_target" "sqs" {
  count = var.create_target_queue ? 1 : 0

  rule           = aws_cloudwatch_event_rule.cart_events.name
  event_bus_name = local.event_bus_name
  target_id      = "sqs-target"
  arn            = aws_sqs_queue.target[0].arn

  dead_letter_config {
    arn = var.create_dlq ? aws_sqs_queue.dlq[0].arn : null
  }

  retry_policy {
    maximum_event_age_in_seconds = var.max_event_age
    maximum_retry_attempts       = var.max_retry_attempts
  }
}

#------------------------------------------------------------------------------
# CloudWatch Log Group for Event Archive
#------------------------------------------------------------------------------
resource "aws_cloudwatch_log_group" "events" {
  count = var.enable_event_logging ? 1 : 0

  name              = "/aws/events/${var.project_name}/${var.service_name}/${var.environment}"
  retention_in_days = var.log_retention_days

  tags = merge(var.tags, {
    Name    = "${var.project_name}-${var.service_name}-events"
    Service = var.service_name
  })
}

resource "aws_cloudwatch_event_target" "logs" {
  count = var.enable_event_logging ? 1 : 0

  rule           = aws_cloudwatch_event_rule.cart_events.name
  event_bus_name = local.event_bus_name
  target_id      = "logs-target"
  arn            = aws_cloudwatch_log_group.events[0].arn
}

resource "aws_cloudwatch_log_resource_policy" "events" {
  count = var.enable_event_logging ? 1 : 0

  policy_name = "${var.project_name}-${var.service_name}-events-policy"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "EventBridgeLogs"
        Effect    = "Allow"
        Principal = { Service = "events.amazonaws.com" }
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${aws_cloudwatch_log_group.events[0].arn}:*"
      }
    ]
  })
}

#------------------------------------------------------------------------------
# Event Archive
#------------------------------------------------------------------------------
resource "aws_cloudwatch_event_archive" "main" {
  count = var.enable_archive ? 1 : 0

  name             = "${var.project_name}-${var.service_name}-${var.environment}"
  description      = "Archive for ${var.service_name} events"
  event_source_arn = local.event_bus_arn
  retention_days   = var.archive_retention_days

  event_pattern = jsonencode({
    source = [var.event_source]
  })
}
