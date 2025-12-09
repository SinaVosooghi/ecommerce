output "secret_arns" {
  description = "Map of secret ARNs"
  value       = { for k, v in aws_secretsmanager_secret.main : k => v.arn }
}

output "secret_names" {
  description = "Map of secret names"
  value       = { for k, v in aws_secretsmanager_secret.main : k => v.name }
}

output "kms_key_arn" {
  description = "KMS key ARN"
  value       = var.create_kms_key ? aws_kms_key.secrets[0].arn : null
}

output "kms_key_id" {
  description = "KMS key ID"
  value       = var.create_kms_key ? aws_kms_key.secrets[0].key_id : null
}
