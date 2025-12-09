# Cart Service Infrastructure

Terraform infrastructure for the Cart Service on AWS ECS Fargate.

## Quick Start

```bash
# 1. Deploy state backend (first time only)
cd infrastructure/backend
terraform init && terraform apply

# 2. Deploy dev environment
cd infrastructure/environments/dev
terraform init && terraform apply
```

## Architecture

```
VPC (2 AZs) → ALB → ECS Fargate → DynamoDB
                 ↓
            EventBridge → SQS
```

**Components**: VPC, ALB, ECS, ECR, DynamoDB, EventBridge, CloudWatch, IAM

## Directory Structure

```
infrastructure/
├── backend/           # S3 state + DynamoDB lock
├── environments/
│   ├── dev/          # Development
│   ├── staging/      # Staging
│   └── prod/         # Production
└── modules/          # Reusable modules (vpc, ecs, alb, etc.)
```

## Environment Configuration

| Setting | Dev | Prod |
|---------|-----|------|
| NAT Gateway | Single | Per AZ |
| ECS Tasks | 1 | 3+ |
| DynamoDB | On-Demand | Provisioned |
| Auto-Scaling | Off | On |
| Fargate Spot | Yes | No |

## Outputs

After deployment, key outputs:
- `alb_dns_name` - API endpoint
- `ecr_repository_url` - Docker registry
- `dynamodb_table_name` - Cart data table

## Build & Deploy Container

```bash
# Login to ECR
aws ecr get-login-password --region eu-central-1 | docker login --username AWS --password-stdin <ACCOUNT>.dkr.ecr.eu-central-1.amazonaws.com

# Build and push
cd services/cart-service
docker build -t cart-service .
docker tag cart-service:latest <ECR_URL>:latest
docker push <ECR_URL>:latest

# Force new deployment
aws ecs update-service --cluster ecommerce-dev --service cart-service-dev --force-new-deployment
```

## Cleanup

⚠️ **CRITICAL ORDER**: You MUST destroy environments BEFORE destroying the backend. Destroying the backend first will delete the state files and make it impossible to properly destroy the environments.

```bash
# 1. Destroy ALL environments first (removes all resources)
cd infrastructure/environments/dev && terraform destroy
# Repeat for staging/prod if they exist:
# cd infrastructure/environments/staging && terraform destroy
# cd infrastructure/environments/prod && terraform destroy

# 2. Destroy backend LAST (S3 bucket and DynamoDB table)
# Note: The S3 bucket must be empty or have force_destroy=true
cd infrastructure/backend && terraform destroy
```

**Important**: 
- **Order is critical**: Destroy environments → Then backend
- The S3 state bucket must be empty or `force_destroy = true` is set in `backend/main.tf`
- If you accidentally destroy the backend first, you'll need to manually clean up any orphaned AWS resources

### Manual Cleanup (if needed)

If Terraform state was lost and resources remain, clean them up manually:

```bash
# Delete ECR repository (with images)
aws ecr batch-delete-image --repository-name ecommerce-cart-service --region eu-central-1 --image-ids imageTag=latest
aws ecr delete-repository --repository-name ecommerce-cart-service --region eu-central-1 --force

# Delete CloudWatch log groups
aws logs delete-log-group --log-group-name /aws/ecs/containerinsights/ecommerce-dev/performance --region eu-central-1
```
