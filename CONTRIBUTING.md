# Contributing

Thanks for your interest in Railzway. The project is evolving quickly, so lightweight coordination helps keep changes aligned.

## Before You Start

1. Check existing issues or open a new one describing:
   - the problem you want to solve
   - expected behavior
   - any API or data model impact
2. Keep changes scoped and incremental. If the scope is large, propose it first.

## Development Setup

Follow the Quick Start in `README.md` to run locally.

## Style & Expectations

- Prefer small, focused PRs.
- Maintain org scoping and idempotency behavior.
- Add tests when changing core billing logic.
- Avoid breaking public routes without discussion.

## PR Checklist

- [ ] Local build passes (Go + admin UI)
- [ ] Database migrations updated if schema changes
- [ ] Docs updated if behavior changes

## Running Tests

```bash
go test ./...
```

## Discussion

If you are unsure about the direction, open an issue first. It is easier to align before big code changes.
