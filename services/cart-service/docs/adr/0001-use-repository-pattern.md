# ADR-0001: Use Repository Pattern for Persistence

## Status
Accepted

## Context
The cart service needs to persist cart data. We need a clean abstraction that:
- Separates business logic from persistence details
- Allows easy testing with mock implementations
- Enables switching between persistence backends (DynamoDB, Redis, etc.)

## Decision
We will use the Repository Pattern to abstract data access.

The `CartRepository` interface defines operations:
- `GetCart(ctx, userID)` - Retrieve a cart
- `SaveCart(ctx, cart)` - Save a cart
- `SaveCartWithVersion(ctx, cart, version)` - Save with optimistic locking
- `DeleteCart(ctx, userID)` - Delete a cart

Implementations:
- `dynamodb.Repository` - Production DynamoDB backend
- `inmemory.Repository` - Testing in-memory backend

## Consequences

### Positive
- Business logic is decoupled from storage technology
- Unit tests can use in-memory implementation
- Easy to add new persistence backends
- Consistent interface across implementations

### Negative
- Additional abstraction layer
- Need to maintain interface across implementations
- Some DynamoDB-specific optimizations may be harder to expose

## Alternatives Considered

### Direct DynamoDB Access
Using DynamoDB SDK directly in business logic.
- Rejected: Tight coupling, hard to test, no flexibility

### ORM/ODM
Using an ORM like GORM or ODM.
- Rejected: Added complexity, less control over queries, DynamoDB doesn't fit ORM model well
