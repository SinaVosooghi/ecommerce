# DynamoDB Module - Cart Service Table

resource "aws_dynamodb_table" "main" {
  name         = "${var.project_name}-${var.service_name}-${var.environment}"
  billing_mode = var.billing_mode

  # Only set these if using PROVISIONED billing mode
  read_capacity  = var.billing_mode == "PROVISIONED" ? var.read_capacity : null
  write_capacity = var.billing_mode == "PROVISIONED" ? var.write_capacity : null

  hash_key  = "PK"
  range_key = "SK"

  # Primary key attributes
  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  # GSI attributes
  attribute {
    name = "GSI1PK"
    type = "S"
  }

  attribute {
    name = "GSI1SK"
    type = "S"
  }

  # GSI for abandoned carts query
  global_secondary_index {
    name            = "GSI1"
    hash_key        = "GSI1PK"
    range_key       = "GSI1SK"
    projection_type = "ALL"

    read_capacity  = var.billing_mode == "PROVISIONED" ? var.gsi_read_capacity : null
    write_capacity = var.billing_mode == "PROVISIONED" ? var.gsi_write_capacity : null
  }

  # TTL for automatic expiration
  ttl {
    attribute_name = "ttl"
    enabled        = var.enable_ttl
  }

  # Point-in-time recovery
  point_in_time_recovery {
    enabled = var.enable_point_in_time_recovery
  }

  # Server-side encryption
  server_side_encryption {
    enabled     = true
    kms_key_arn = var.kms_key_arn
  }

  # Stream for change data capture (optional)
  stream_enabled   = var.enable_streams
  stream_view_type = var.enable_streams ? var.stream_view_type : null

  tags = merge(var.tags, {
    Name        = "${var.project_name}-${var.service_name}-${var.environment}"
    Service     = var.service_name
    Environment = var.environment
  })

  lifecycle {
    prevent_destroy = false # Set to true in production
  }
}

#------------------------------------------------------------------------------
# Auto Scaling (for PROVISIONED mode)
#------------------------------------------------------------------------------
resource "aws_appautoscaling_target" "read" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  max_capacity       = var.read_max_capacity
  min_capacity       = var.read_capacity
  resource_id        = "table/${aws_dynamodb_table.main.name}"
  scalable_dimension = "dynamodb:table:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "read" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  name               = "${var.project_name}-${var.service_name}-read-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.read[0].resource_id
  scalable_dimension = aws_appautoscaling_target.read[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.read[0].service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }
    target_value = var.target_utilization
  }
}

resource "aws_appautoscaling_target" "write" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  max_capacity       = var.write_max_capacity
  min_capacity       = var.write_capacity
  resource_id        = "table/${aws_dynamodb_table.main.name}"
  scalable_dimension = "dynamodb:table:WriteCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "write" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  name               = "${var.project_name}-${var.service_name}-write-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.write[0].resource_id
  scalable_dimension = aws_appautoscaling_target.write[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.write[0].service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBWriteCapacityUtilization"
    }
    target_value = var.target_utilization
  }
}

# GSI Auto Scaling
resource "aws_appautoscaling_target" "gsi_read" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  max_capacity       = var.gsi_read_max_capacity
  min_capacity       = var.gsi_read_capacity
  resource_id        = "table/${aws_dynamodb_table.main.name}/index/GSI1"
  scalable_dimension = "dynamodb:index:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "gsi_read" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  name               = "${var.project_name}-${var.service_name}-gsi-read-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.gsi_read[0].resource_id
  scalable_dimension = aws_appautoscaling_target.gsi_read[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.gsi_read[0].service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }
    target_value = var.target_utilization
  }
}

resource "aws_appautoscaling_target" "gsi_write" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  max_capacity       = var.gsi_write_max_capacity
  min_capacity       = var.gsi_write_capacity
  resource_id        = "table/${aws_dynamodb_table.main.name}/index/GSI1"
  scalable_dimension = "dynamodb:index:WriteCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "gsi_write" {
  count = var.billing_mode == "PROVISIONED" && var.enable_autoscaling ? 1 : 0

  name               = "${var.project_name}-${var.service_name}-gsi-write-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.gsi_write[0].resource_id
  scalable_dimension = aws_appautoscaling_target.gsi_write[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.gsi_write[0].service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBWriteCapacityUtilization"
    }
    target_value = var.target_utilization
  }
}
