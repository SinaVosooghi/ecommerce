# CI/CD Pipeline Status Report
**Date**: 2025-12-10  
**Environment**: Dev  
**Region**: eu-central-1

## Executive Summary

CI/CD pipeline is **partially functional**:
- ✅ **Source Stage**: Working perfectly - pulls code from GitHub
- ✅ **Build Stage**: Working perfectly - builds Docker image and pushes to ECR
- ❌ **Deploy Stage**: **FAILING** - ECS tasks fail container health checks

## Current Pipeline Status

**Pipeline Name**: `ecommerce-cart-service`  
**Latest Execution**: In Progress (Deploy stage failing)

| Stage | Status | Notes |
|-------|--------|-------|
| Source | ✅ Succeeded | Pulls from `SinaVosooghi/ecommerce` repo, `develop` branch |
| Build | ✅ Succeeded | Builds Go 1.24 binary, creates Docker image, pushes to ECR |
| Deploy | ❌ Failed | ECS tasks start but fail container health checks |

## Root Cause Analysis

### Primary Issue: Container Health Check Failure

**Problem**: ECS tasks are being marked as unhealthy and terminated due to failed container health checks.

**Symptoms**:
- Tasks start successfully (container runs, application listens on port 8080)
- Application logs show: "Server listening on port 8080"
- Health checks fail after ~60 seconds (startPeriod)
- Tasks are stopped: "failed container health checks"
- Deployment fails: "tasks failed to start"

**Technical Details**:
- **Image**: Uses `gcr.io/distroless/static:nonroot` (minimal image, no shell, no utilities)
- **Health Check Command**: `["CMD", "/healthcheck"]` 
- **Health Check Binary**: Custom Go binary built in Dockerfile that makes HTTP GET to `http://localhost:8080/health`
- **Current Task Definition**: `ecommerce-cart-service-dev:7`
- **Health Check Config**: interval=30s, timeout=5s, retries=3, startPeriod=60s

### Attempted Solutions (All Failed)

1. ✅ **Added healthcheck binary** - Built minimal HTTP client Go binary
2. ✅ **Fixed binary permissions** - Added `--chmod=755` to COPY command
3. ✅ **Updated ECS health check** - Changed from `CMD-SHELL` with `wget` to `CMD` with `/healthcheck`
4. ✅ **Static linking** - Used `CGO_ENABLED=0` and `netgo` tags
5. ❌ **Still failing** - Tasks continue to fail health checks

### Current Configuration

**Dockerfile** (`services/cart-service/Dockerfile`):
```dockerfile
# Builds healthcheck binary:
RUN printf 'package main\nimport ("net/http";"os";"io";"time")\nfunc main(){client:=&http.Client{Timeout:5*time.Second};resp,err:=client.Get("http://localhost:8080/health");if err!=nil{os.Exit(1)};defer resp.Body.Close();io.Copy(io.Discard,resp.Body);if resp.StatusCode!=200{os.Exit(1)}}\n' > /tmp/healthcheck.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -tags netgo -installsuffix netgo -o /app/healthcheck /tmp/healthcheck.go

# Copies to distroless image:
COPY --from=builder --chmod=755 /app/healthcheck /healthcheck
```

**ECS Task Definition Health Check**:
```json
{
  "command": ["CMD", "/healthcheck"],
  "interval": 30,
  "timeout": 5,
  "retries": 3,
  "startPeriod": 60
}
```

## Infrastructure State

### Successfully Deployed Resources

1. **CodeStar Connection**: ✅ AVAILABLE
   - ARN: `arn:aws:codestar-connections:eu-central-1:762233763891:connection/9ebbea6c-dcf3-45be-86c8-9b2bdcbe772e`
   - Name: `ecommerce-cart-service-dev`
   - Provider: GitHub

2. **CodePipeline**: ✅ Created
   - Name: `ecommerce-cart-service`
   - Source: GitHub `SinaVosooghi/ecommerce` (develop branch)
   - Build: CodeBuild project `ecommerce-cart-service-build`
   - Deploy: ECS deployment to `ecommerce-dev` cluster

3. **CodeBuild**: ✅ Working
   - Project: `ecommerce-cart-service-build`
   - Buildspec: `services/cart-service/buildspec-build.yml`
   - Successfully builds and pushes to ECR

4. **ECS Service**: ⚠️ Deployed but Unhealthy
   - Cluster: `ecommerce-dev`
   - Service: `cart-service-dev`
   - Task Definition: `ecommerce-cart-service-dev:7`
   - Status: Tasks start but fail health checks

5. **ECR Repository**: ✅ Working
   - Repository: `ecommerce-cart-service`
   - Images successfully pushed with commit SHA tags

### Terraform Configuration

**Files Modified**:
- `infrastructure/environments/dev/terraform.tfvars` - CI/CD enabled
- `infrastructure/environments/dev/main.tf` - CodeStar connection + CI/CD module
- `infrastructure/environments/dev/variables.tf` - CI/CD variables added
- `infrastructure/environments/dev/outputs.tf` - CodeStar connection outputs
- `infrastructure/modules/cicd/main.tf` - IAM permissions updated
- `infrastructure/modules/ecs/main.tf` - Health check made optional
- `infrastructure/modules/ecs/variables.tf` - `enable_container_health_check` variable
- `services/cart-service/buildspec-build.yml` - Fixed paths, ECR repo name
- `services/cart-service/Dockerfile` - Added healthcheck binary

## Key Problems Identified

### 1. Health Check Binary May Not Be Working

**Hypothesis**: The `/healthcheck` binary might not be executing correctly in distroless environment, or the HTTP request is failing for some reason.

**Evidence**:
- Tasks start and application runs (logs confirm)
- Health checks fail consistently
- No error logs from healthcheck binary (can't see healthcheck execution logs)

**Possible Issues**:
- Binary might not be statically linked correctly for distroless
- HTTP client might not work in distroless environment
- Binary might not have proper permissions despite chmod
- Network connectivity issue within container

### 2. Alternative Approaches Not Tried

**Option A**: Use ALB health checks only (disable container health check)
- Pros: ALB health checks are working (tasks register successfully)
- Cons: Less granular, user wants container health check

**Option B**: Switch to Alpine base image instead of distroless
- Pros: Has shell and utilities (wget/curl)
- Cons: Larger image, defeats purpose of distroless

**Option C**: Use application binary's built-in health check
- Check if `/cart-service` binary supports health check flag
- Dockerfile line 52 shows: `CMD ["/cart-service", "-health-check"]`
- This suggests the binary might support health checks natively

**Option D**: Create separate healthcheck.go file instead of inline build
- More maintainable
- Easier to debug

## Recommended Next Steps

### Immediate Actions

1. **Verify healthcheck binary works**:
   - Test the binary locally in a distroless container
   - Check if it can actually make HTTP requests
   - Verify it's properly statically linked

2. **Check application health endpoint**:
   - Verify `/health` endpoint returns 200 OK
   - Test from within container if possible

3. **Consider using application's native health check**:
   - The Dockerfile suggests `/cart-service -health-check` might work
   - This would be simpler than custom binary

4. **Alternative: Use ALB health checks**:
   - Disable container health check temporarily
   - Rely on ALB target group health checks
   - Tasks are registering successfully with ALB

### Investigation Needed

1. **Check ECS task logs** for healthcheck binary execution errors
2. **Verify healthcheck binary exists** in the image (inspect image layers)
3. **Test healthcheck binary** in a local distroless container
4. **Check if application binary** supports `-health-check` flag natively

## Files Reference

### Key Configuration Files

- **CI/CD Module**: `infrastructure/modules/cicd/main.tf`
- **ECS Module**: `infrastructure/modules/ecs/main.tf`
- **Dev Environment**: `infrastructure/environments/dev/main.tf`
- **Buildspec**: `services/cart-service/buildspec-build.yml`
- **Dockerfile**: `services/cart-service/Dockerfile`

### Current Branch

- **Branch**: `develop`
- **Repository**: `SinaVosooghi/ecommerce`
- **Latest Commit**: Healthcheck binary fixes

## Environment Details

- **AWS Region**: eu-central-1
- **ECS Cluster**: ecommerce-dev
- **ECS Service**: cart-service-dev
- **ECR Repository**: ecommerce-cart-service
- **ALB**: ecommerce-dev-alb
- **Target Group**: ecommerce-cart-service-dev

## Success Metrics

✅ **Working**:
- CodePipeline triggers on push to develop branch
- CodeBuild successfully builds Go application
- Docker images built and pushed to ECR
- ECS tasks start and application runs
- ALB target registration works

❌ **Not Working**:
- Container health checks fail
- ECS deployments fail due to unhealthy tasks
- Pipeline deploy stage fails

## Conclusion

The CI/CD pipeline is **90% functional**. The build process works perfectly, but the deployment fails due to container health check issues. The root cause appears to be the custom healthcheck binary not working correctly in the distroless environment, despite multiple attempts to fix it.

**Recommendation**: Investigate why the healthcheck binary fails, or consider using the application's native health check capability (if it exists) or temporarily disable container health checks and rely on ALB health checks only.
