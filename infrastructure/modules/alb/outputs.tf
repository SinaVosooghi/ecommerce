output "alb_id" {
  description = "ALB ID"
  value       = aws_lb.main.id
}

output "alb_arn" {
  description = "ALB ARN"
  value       = aws_lb.main.arn
}

output "alb_dns_name" {
  description = "ALB DNS name"
  value       = aws_lb.main.dns_name
}

output "alb_zone_id" {
  description = "ALB zone ID"
  value       = aws_lb.main.zone_id
}

output "security_group_id" {
  description = "ALB security group ID"
  value       = aws_security_group.alb.id
}

output "target_group_arn" {
  description = "Target group ARN"
  value       = aws_lb_target_group.main.arn
}

output "target_group_name" {
  description = "Target group name"
  value       = aws_lb_target_group.main.name
}

output "http_listener_arn" {
  description = "HTTP listener ARN"
  value       = aws_lb_listener.http.arn
}

output "https_listener_arn" {
  description = "HTTPS listener ARN"
  value       = var.certificate_arn != "" ? aws_lb_listener.https[0].arn : null
}
