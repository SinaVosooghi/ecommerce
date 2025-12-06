# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Cart Service.

## What is an ADR?

An Architecture Decision Record captures an important architectural decision made along with its context and consequences.

## ADR Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-use-repository-pattern.md) | Use Repository Pattern for Persistence | Accepted | 2024-12 |
| [0002](0002-dynamodb-single-table-design.md) | DynamoDB Single-Table Design | Accepted | 2024-12 |
| [0003](0003-optimistic-locking.md) | Optimistic Locking for Concurrency | Accepted | 2024-12 |
| [0004](0004-eventbridge-for-events.md) | EventBridge for Event Publishing | Accepted | 2024-12 |
| [0005](0005-circuit-breaker-pattern.md) | Circuit Breaker for Resilience | Accepted | 2024-12 |

## ADR Template

```markdown
# ADR-XXXX: [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-XXXX]

## Context
[What is the issue or decision that needs to be made?]

## Decision
[What is the proposed solution?]

## Consequences
### Positive
- [Benefit 1]
- [Benefit 2]

### Negative
- [Drawback 1]
- [Drawback 2]

## Alternatives Considered
### [Alternative 1]
[Description and why it was not chosen]

### [Alternative 2]
[Description and why it was not chosen]
```

## Creating a New ADR

1. Copy the template above
2. Create a new file: `NNNN-short-title.md`
3. Fill in all sections
4. Update this README index
5. Submit for review
