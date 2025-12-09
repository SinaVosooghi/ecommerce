output "vpc_id" {
  description = "VPC ID"
  value       = local.vpc_id
}

output "vpc_cidr" {
  description = "VPC CIDR block"
  value       = local.create_vpc ? aws_vpc.main[0].cidr_block : data.aws_vpc.existing[0].cidr_block
}

output "public_subnet_ids" {
  description = "List of public subnet IDs"
  value       = local.create_vpc ? aws_subnet.public[*].id : data.aws_subnets.existing_public[0].ids
}

output "private_subnet_ids" {
  description = "List of private subnet IDs"
  value       = local.create_vpc ? aws_subnet.private[*].id : data.aws_subnets.existing_private[0].ids
}

output "availability_zones" {
  description = "List of availability zones"
  value       = local.azs
}

output "nat_gateway_ids" {
  description = "List of NAT Gateway IDs"
  value       = aws_nat_gateway.main[*].id
}

output "vpc_endpoints_security_group_id" {
  description = "Security group ID for VPC endpoints"
  value       = local.create_vpc && var.enable_vpc_endpoints ? aws_security_group.vpc_endpoints[0].id : null
}
