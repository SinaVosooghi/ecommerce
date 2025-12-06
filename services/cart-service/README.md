# Cart Service

A production-ready shopping cart microservice built in Go, designed for AWS deployment with ECS Fargate.

## Features

- **RESTful API** with versioned endpoints (v1)
- **DynamoDB** persistence with single-table design
- **EventBridge** integration for async event publishing
- **Optimistic locking** for concurrency control
- **Circuit breaker** and retry patterns for resilience
- **JWT authentication** support
- **Rate limiting** and request validation
- **Comprehensive observability** (structured logging, metrics, X-Ray tracing)
- **Feature flags** support
- **Idempotency** for safe retries

## Quick Start

### Prerequisites

- Go 1.22+
- AWS CLI configured
- Docker (for local DynamoDB)

### Local Development

```bash
# Clone and navigate to the service
cd services/cart-service

# Install dependencies
go mod download

# Run with local configuration
export ENV_NAME=dev
export DYNAMODB_ENDPOINT=http://localhost:8000
export DYNAMODB_TABLE=cart-service-carts
go run cmd/cart-service/main.go
```

### Running with Docker Compose

```bash
# Start dependencies (DynamoDB Local, Redis)
docker-compose up -d

# Run the service
go run cmd/cart-service/main.go
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe |
| GET | `/v1/cart/{userID}` | Get cart |
| POST | `/v1/cart/{userID}/items` | Add item to cart |
| PATCH | `/v1/cart/{userID}/items/{itemID}` | Update item quantity |
| DELETE | `/v1/cart/{userID}/items/{itemID}` | Remove item from cart |
| DELETE | `/v1/cart/{userID}` | Clear cart |

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_PORT` | HTTP server port | 8080 |
| `ENV_NAME` | Environment (dev/staging/prod) | dev |
| `LOG_LEVEL` | Logging level | info |
| `AWS_REGION` | AWS region | us-east-1 |
| `DYNAMODB_TABLE` | DynamoDB table name | cart-service-carts |
| `DYNAMODB_ENDPOINT` | DynamoDB endpoint (for local) | - |
| `AWS_XRAY_ENABLED` | Enable X-Ray tracing | false |
| `RATE_LIMIT_RPS` | Rate limit per second | 100 |
| `CIRCUIT_BREAKER_ENABLED` | Enable circuit breaker | true |
| `EVENTBRIDGE_ENABLED` | Enable EventBridge events | true |
| `EVENTBRIDGE_BUS_NAME` | EventBridge bus name | default |

## Project Structure

```
cart-service/
├── cmd/
│   └── cart-service/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── v1/handlers/         # HTTP handlers
│   │   └── middleware/          # HTTP middleware
│   ├── core/cart/               # Domain logic
│   ├── config/                  # Configuration
│   ├── logging/                 # Structured logging
│   ├── server/                  # HTTP server
│   ├── health/                  # Health endpoints
│   ├── persistence/
│   │   ├── dynamodb/            # DynamoDB implementation
│   │   └── inmemory/            # In-memory for testing
│   ├── events/
│   │   ├── eventbridge/         # EventBridge implementation
│   │   └── models/              # Event definitions
│   ├── metrics/                 # Observability
│   ├── features/                # Feature flags
│   ├── secrets/                 # Secrets management
│   ├── resilience/              # Circuit breaker, retry
│   ├── errors/                  # Error handling
│   └── app/                     # Dependency injection
├── migrations/                  # Database migrations
├── docs/
│   ├── swagger.yaml             # OpenAPI spec
│   ├── postman.json             # Postman collection
│   ├── runbook.md               # Operations runbook
│   └── adr/                     # Architecture decisions
├── tests/
│   ├── integration/             # Integration tests
│   ├── e2e/                     # End-to-end tests
│   ├── contract/                # Contract tests
│   └── load/                    # Load tests (k6)
├── Dockerfile
├── go.mod
└── README.md
```

## Testing

```bash
# Run unit tests
go test ./internal/...

# Run with coverage
go test -cover ./internal/...

# Run integration tests (requires Docker)
go test ./tests/integration/... -tags=integration

# Run load tests
k6 run tests/load/scenarios/baseline.js
```

## Building

```bash
# Build binary
go build -o bin/cart-service ./cmd/cart-service

# Build Docker image
docker build -t cart-service:latest .
```

## Deployment

The service is designed for deployment on AWS ECS Fargate. See the infrastructure documentation for Terraform modules.

### Health Checks

- **Liveness** (`/health`): Always returns 200 OK
- **Readiness** (`/ready`): Checks DynamoDB connectivity

### IAM Permissions Required

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/cart-service-*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "events:PutEvents"
      ],
      "Resource": "arn:aws:events:*:*:event-bus/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ],
      "Resource": "*"
    }
  ]
}
```

## Architecture Decisions

See [docs/adr/](docs/adr/) for architecture decision records.

## License

Internal use only.
