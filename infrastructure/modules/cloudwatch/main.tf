# CloudWatch Module - Logs, Dashboards, Alarms

#------------------------------------------------------------------------------
# Log Group
#------------------------------------------------------------------------------
resource "aws_cloudwatch_log_group" "main" {
  name              = "/ecs/${var.project_name}/${var.service_name}/${var.environment}"
  retention_in_days = var.log_retention_days

  tags = merge(var.tags, {
    Name    = "${var.project_name}-${var.service_name}-${var.environment}"
    Service = var.service_name
  })
}

#------------------------------------------------------------------------------
# Metric Alarms
#------------------------------------------------------------------------------

# High CPU Utilization
resource "aws_cloudwatch_metric_alarm" "cpu_high" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.project_name}-${var.service_name}-${var.environment}-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = 60
  statistic           = "Average"
  threshold           = var.cpu_alarm_threshold
  alarm_description   = "CPU utilization is above ${var.cpu_alarm_threshold}%"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = var.ecs_service_name
  }

  alarm_actions = var.alarm_sns_topic_arns
  ok_actions    = var.alarm_sns_topic_arns

  tags = var.tags
}

# High Memory Utilization
resource "aws_cloudwatch_metric_alarm" "memory_high" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.project_name}-${var.service_name}-${var.environment}-memory-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = 60
  statistic           = "Average"
  threshold           = var.memory_alarm_threshold
  alarm_description   = "Memory utilization is above ${var.memory_alarm_threshold}%"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ClusterName = var.ecs_cluster_name
    ServiceName = var.ecs_service_name
  }

  alarm_actions = var.alarm_sns_topic_arns
  ok_actions    = var.alarm_sns_topic_arns

  tags = var.tags
}

# ALB 5xx Errors
resource "aws_cloudwatch_metric_alarm" "alb_5xx" {
  count = var.enable_alarms && var.alb_arn_suffix != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.service_name}-${var.environment}-alb-5xx"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "HTTPCode_Target_5XX_Count"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Sum"
  threshold           = var.error_rate_threshold
  alarm_description   = "High 5xx error rate"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
    TargetGroup  = var.target_group_arn_suffix
  }

  alarm_actions = var.alarm_sns_topic_arns
  ok_actions    = var.alarm_sns_topic_arns

  tags = var.tags
}

# ALB Response Time
resource "aws_cloudwatch_metric_alarm" "alb_latency" {
  count = var.enable_alarms && var.alb_arn_suffix != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.service_name}-${var.environment}-alb-latency"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 3
  metric_name         = "TargetResponseTime"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  extended_statistic  = "p99"
  threshold           = var.latency_threshold
  alarm_description   = "P99 latency is above ${var.latency_threshold}s"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
    TargetGroup  = var.target_group_arn_suffix
  }

  alarm_actions = var.alarm_sns_topic_arns
  ok_actions    = var.alarm_sns_topic_arns

  tags = var.tags
}

# DynamoDB Throttling
resource "aws_cloudwatch_metric_alarm" "dynamodb_throttle" {
  count = var.enable_alarms && var.dynamodb_table_name != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.service_name}-${var.environment}-dynamodb-throttle"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "ThrottledRequests"
  namespace           = "AWS/DynamoDB"
  period              = 60
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "DynamoDB requests are being throttled"
  treat_missing_data  = "notBreaching"

  dimensions = {
    TableName = var.dynamodb_table_name
  }

  alarm_actions = var.alarm_sns_topic_arns
  ok_actions    = var.alarm_sns_topic_arns

  tags = var.tags
}

# Unhealthy Hosts
resource "aws_cloudwatch_metric_alarm" "unhealthy_hosts" {
  count = var.enable_alarms && var.target_group_arn_suffix != "" ? 1 : 0

  alarm_name          = "${var.project_name}-${var.service_name}-${var.environment}-unhealthy-hosts"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "UnHealthyHostCount"
  namespace           = "AWS/ApplicationELB"
  period              = 60
  statistic           = "Average"
  threshold           = 0
  alarm_description   = "There are unhealthy hosts"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = var.alb_arn_suffix
    TargetGroup  = var.target_group_arn_suffix
  }

  alarm_actions = var.alarm_sns_topic_arns

  tags = var.tags
}

#------------------------------------------------------------------------------
# Dashboard
#------------------------------------------------------------------------------
resource "aws_cloudwatch_dashboard" "main" {
  count = var.create_dashboard ? 1 : 0

  dashboard_name = "${var.project_name}-${var.service_name}-${var.environment}"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "text"
        x      = 0
        y      = 0
        width  = 24
        height = 1
        properties = {
          markdown = "# ${var.service_name} - ${var.environment} Dashboard"
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 1
        width  = 8
        height = 6
        properties = {
          title  = "CPU Utilization"
          region = var.aws_region
          metrics = [
            ["AWS/ECS", "CPUUtilization", "ClusterName", var.ecs_cluster_name, "ServiceName", var.ecs_service_name]
          ]
          period = 60
          stat   = "Average"
        }
      },
      {
        type   = "metric"
        x      = 8
        y      = 1
        width  = 8
        height = 6
        properties = {
          title  = "Memory Utilization"
          region = var.aws_region
          metrics = [
            ["AWS/ECS", "MemoryUtilization", "ClusterName", var.ecs_cluster_name, "ServiceName", var.ecs_service_name]
          ]
          period = 60
          stat   = "Average"
        }
      },
      {
        type   = "metric"
        x      = 16
        y      = 1
        width  = 8
        height = 6
        properties = {
          title  = "Running Tasks"
          region = var.aws_region
          metrics = [
            ["ECS/ContainerInsights", "RunningTaskCount", "ClusterName", var.ecs_cluster_name, "ServiceName", var.ecs_service_name]
          ]
          period = 60
          stat   = "Average"
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 7
        width  = 12
        height = 6
        properties = {
          title  = "Request Count"
          region = var.aws_region
          metrics = [
            ["AWS/ApplicationELB", "RequestCount", "LoadBalancer", var.alb_arn_suffix, "TargetGroup", var.target_group_arn_suffix]
          ]
          period = 60
          stat   = "Sum"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 7
        width  = 12
        height = 6
        properties = {
          title  = "Response Time (P99)"
          region = var.aws_region
          metrics = [
            ["AWS/ApplicationELB", "TargetResponseTime", "LoadBalancer", var.alb_arn_suffix, "TargetGroup", var.target_group_arn_suffix]
          ]
          period = 60
          stat   = "p99"
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 13
        width  = 12
        height = 6
        properties = {
          title  = "HTTP Errors"
          region = var.aws_region
          metrics = [
            ["AWS/ApplicationELB", "HTTPCode_Target_4XX_Count", "LoadBalancer", var.alb_arn_suffix, "TargetGroup", var.target_group_arn_suffix, { "color" = "#ff7f0e" }],
            ["AWS/ApplicationELB", "HTTPCode_Target_5XX_Count", "LoadBalancer", var.alb_arn_suffix, "TargetGroup", var.target_group_arn_suffix, { "color" = "#d62728" }]
          ]
          period = 60
          stat   = "Sum"
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 13
        width  = 12
        height = 6
        properties = {
          title  = "DynamoDB Operations"
          region = var.aws_region
          metrics = [
            ["AWS/DynamoDB", "ConsumedReadCapacityUnits", "TableName", var.dynamodb_table_name],
            ["AWS/DynamoDB", "ConsumedWriteCapacityUnits", "TableName", var.dynamodb_table_name]
          ]
          period = 60
          stat   = "Sum"
        }
      }
    ]
  })
}
