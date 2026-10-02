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
# Option 1: Using .env file (recommended for local development)
# Copy .env.example to .env and customize as needed:
# cp .env.example .env
# Then edit .env with your local settings
go run cmd/cart-service/main.go

# Option 2: Using environment variables
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
| POST | `/v1/cart/{userID}/merge` | Merge a guest cart (`{"guest_id": "..."}`) into the user's cart |

### Authentication and authorization

All `/v1` routes require `Authorization: Bearer <JWT>`:

- Tokens must be HS256, signed with `JWT_SECRET_KEY`, and carry an `exp` claim. If `JWT_ISSUER` or `JWT_AUDIENCE` is set, the token's `iss` and `aud` must match.
- The token's `sub` claim must equal `{userID}` in the path. A token for one user gets `403` on another user's cart.
- `AUTH_ENABLED=false` turns auth off. The service refuses to start that way unless `ENV_NAME=dev`.

### Concurrency and retries

- Every write is a conditional DynamoDB put on the cart `version`, so concurrent requests can't overwrite each other. The server retries a few times on contention, then returns `409 CONFLICT`. Clients should re-read the cart and retry.
- `PATCH .../items/{itemID}` accepts an optional `version`. If the cart has changed since that version, the request fails with `409` instead of overwriting the newer state.
- Send an `Idempotency-Key` header on `POST` and `PATCH` to make retries safe. Keys are scoped per user and stored in DynamoDB for `IDEMPOTENCY_TTL`.
  - A repeat request replays the original response, marked with `X-Idempotent-Replayed: true`.
  - Reusing a key with a different body returns `422`.
  - Reusing a key while the original request is still in flight returns `409`.

> **Pricing:** no catalog service exists yet, so the client-supplied `unit_price` is stored as-is (a warning is logged at startup). Plug a `cart.PriceValidator` into `cart.ServiceConfig.Prices` to use server-side prices.

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_PORT` | HTTP server port | 8080 |
| `ENV_NAME` | Environment (dev/staging/prod) | dev |
| `AUTH_ENABLED` | Require JWT auth on `/v1` (only `dev` may disable it) | true |
| `JWT_SECRET_KEY` | HS256 signing key, at least 32 bytes. In AWS, ECS injects it from Secrets Manager | - |
| `JWT_ISSUER` / `JWT_AUDIENCE` | Expected `iss` / `aud` claims (optional) | - |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins. `*` is only allowed in dev; empty disables CORS | `*` in dev, empty otherwise |
| `IDEMPOTENCY_ENABLED` / `IDEMPOTENCY_TTL` | Idempotency-Key support and retention | true / 24h |
| `METRICS_ENABLED` | Emit request metrics as CloudWatch EMF on stdout | false |
| `LOG_LEVEL` | Logging level | info |
| `AWS_REGION` | AWS region | us-east-1 |
| `DYNAMODB_TABLE` | DynamoDB table name | cart-service-carts |
| `DYNAMODB_ENDPOINT` | DynamoDB endpoint (for local) | - |
| `AWS_XRAY_ENABLED` | Enable X-Ray tracing | false |
| `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | Per-user (or per-peer-IP when unauthenticated) rate limit | 100 / 200 |
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

# Run integration tests (in-memory; they exercise the real router and middleware)
go test -race ./tests/integration/...

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

- **Liveness** (`/health`): Always returns 200 OK.
- **Readiness** (`/ready`): Returns 503 when DynamoDB is unreachable.
- **Container health check**: the runtime image is distroless (no shell or `wget`), so Docker and ECS run `/cart-service -health-check`, which probes `http://127.0.0.1:$APP_PORT/health`.

### JWT signing key

Terraform creates the `<project>/<env>/cart-service/jwt-secret-key` secret but never its value, so the key stays out of Terraform state. Set the value once per environment, before the first deploy:

```bash
aws secretsmanager put-secret-value \
  --secret-id "$(terraform -chdir=infrastructure/environments/dev output -raw jwt_secret_name)" \
  --secret-string "$(openssl rand -base64 48)"
```

ECS injects it as `JWT_SECRET_KEY`. Tasks fail to start until the secret has a value.

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
