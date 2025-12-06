# Cart Service Runbook

## Service Overview

| Property | Value |
|----------|-------|
| Service Name | cart-service |
| Owner | Platform Team |
| Repository | github.com/sinavosooghi/ecommerce/services/cart-service |
| Dashboard | [CloudWatch Dashboard](https://console.aws.amazon.com/cloudwatch) |
| Logs | [CloudWatch Logs](https://console.aws.amazon.com/cloudwatch/home#logs:) |
| On-call | #cart-service-oncall |

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│    ALB      │────▶│  Cart Svc   │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
                    ▼                          ▼                          ▼
             ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
             │  DynamoDB   │          │ EventBridge │          │    Redis    │
             └─────────────┘          └─────────────┘          └─────────────┘
```

## Health Checks

| Endpoint | Purpose | Expected Response |
|----------|---------|-------------------|
| `/health` | Liveness probe | 200 OK |
| `/ready` | Readiness probe | 200 OK (when healthy) |

## Common Issues

### 1. High Latency (p99 > 500ms)

**Symptoms:**
- CloudWatch alarm: `CartService-HighLatency`
- Slow API responses
- X-Ray traces show bottlenecks

**Investigation:**
```bash
# Check DynamoDB metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name SuccessfulRequestLatency \
  --dimensions Name=TableName,Value=cart-service-carts \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 60 \
  --statistics Average

# Check for throttling
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ThrottledRequests \
  --dimensions Name=TableName,Value=cart-service-carts \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 60 \
  --statistics Sum
```

**Resolution:**
- If DynamoDB throttling: Increase provisioned capacity or switch to on-demand
- If application bottleneck: Check circuit breaker states, scale out tasks
- If network: Check VPC flow logs, security group rules

### 2. High Error Rate (5xx > 1%)

**Symptoms:**
- CloudWatch alarm: `CartService-HighErrorRate`
- Error logs in CloudWatch Logs Insights

**Investigation:**
```bash
# Query error logs
aws logs filter-log-events \
  --log-group-name /ecs/cart-service \
  --filter-pattern "ERROR" \
  --start-time $(date -u -d '30 minutes ago' +%s)000

# Check circuit breaker states in logs
aws logs filter-log-events \
  --log-group-name /ecs/cart-service \
  --filter-pattern "circuit" \
  --start-time $(date -u -d '30 minutes ago' +%s)000
```

**Resolution:**
- If circuit breaker open: Wait for recovery or investigate dependency
- If persistent errors: Check DynamoDB table status, EventBridge rules
- If deployment issue: Rollback to previous version

### 3. Memory Leak Suspected

**Symptoms:**
- Gradual memory increase
- OOM kills in ECS events
- Container restarts

**Investigation:**
```bash
# Check ECS task metrics
aws ecs describe-services \
  --cluster ecommerce \
  --services cart-service

# Check container memory utilization
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name MemoryUtilization \
  --dimensions Name=ServiceName,Value=cart-service Name=ClusterName,Value=ecommerce \
  --start-time $(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 300 \
  --statistics Average
```

**Resolution:**
- Identify leaking goroutines or unclosed connections
- Check for unbounded caches
- Deploy fix or rollback

### 4. DynamoDB Capacity Issues

**Symptoms:**
- `ProvisionedThroughputExceededException` errors
- High read/write throttling

**Investigation:**
```bash
# Check table capacity
aws dynamodb describe-table \
  --table-name cart-service-carts \
  --query 'Table.{RCU:ProvisionedThroughput.ReadCapacityUnits,WCU:ProvisionedThroughput.WriteCapacityUnits}'

# Check consumed capacity
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ConsumedReadCapacityUnits \
  --dimensions Name=TableName,Value=cart-service-carts \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 60 \
  --statistics Sum
```

**Resolution:**
```bash
# Increase capacity
aws dynamodb update-table \
  --table-name cart-service-carts \
  --provisioned-throughput ReadCapacityUnits=200,WriteCapacityUnits=100
```

## Rollback Procedure

1. **Identify last known good version:**
```bash
aws ecs describe-task-definition \
  --task-definition cart-service \
  --query 'taskDefinition.revision'
```

2. **Update service to previous revision:**
```bash
aws ecs update-service \
  --cluster ecommerce \
  --service cart-service \
  --task-definition cart-service:PREVIOUS_REVISION \
  --force-new-deployment
```

3. **Monitor rollback:**
```bash
aws ecs wait services-stable \
  --cluster ecommerce \
  --services cart-service
```

## Scaling Procedures

### Manual Scale Up

```bash
aws ecs update-service \
  --cluster ecommerce \
  --service cart-service \
  --desired-count 10
```

### Update Auto Scaling

```bash
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --resource-id service/ecommerce/cart-service \
  --scalable-dimension ecs:service:DesiredCount \
  --min-capacity 3 \
  --max-capacity 20
```

## Contacts

| Role | Contact |
|------|---------|
| Service Owner | @platform-team |
| On-call | PagerDuty: cart-service |
| Escalation | @engineering-leads |
| AWS Support | Case via Console |

## Useful Commands

```bash
# View service logs
aws logs tail /ecs/cart-service --follow

# Check task health
aws ecs describe-tasks \
  --cluster ecommerce \
  --tasks $(aws ecs list-tasks --cluster ecommerce --service-name cart-service --query 'taskArns' --output text)

# Force deployment
aws ecs update-service \
  --cluster ecommerce \
  --service cart-service \
  --force-new-deployment

# View X-Ray traces
aws xray get-trace-summaries \
  --start-time $(date -u -d '1 hour ago' +%s) \
  --end-time $(date -u +%s) \
  --filter-expression 'service(id(name: "cart-service"))'
```
