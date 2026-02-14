# Contributing to Railzway

Thank you for your interest in contributing to **Railzway**! We want to build the best Open Source Billing Engine, and your help is vital.

## 1. Getting Started

### Prerequisites
- **Go**: Version 1.21+
- **PostgreSQL**: Version 14+
- **Node.js**: Version 18+ (for frontend apps)

### Setup
1.  Clone the repository.
2.  Copy `.env.example` to `.env`.
3.  Run `go run cmd/railzway/main.go migrate` to setup the database.
4.  Run `go run cmd/railzway/main.go serve` to start the backend.

## 2. Coding Standards

We strive for code that is simple, explicit, and easy to read.

### 2.1 Go Style
- **Formatting**: Always run `go fmt` before committing.
- **Errors**:
    - Use `errors.Join` for compiling multiple errors.
    - Export sentinel errors with `Err` prefix (e.g., `ErrNotFound`).
    - Don't just return `err`, wrap it with context: `fmt.Errorf("failed to create user: %w", err)`.
- **Interfaces**:
    - Define interfaces where they are *used* (consumer-driven), not where they are implemented, unless it's a domain port.
    - Keep interfaces small (Single Responsibility Principle).

### 2.2 Database (Gorm)
- Use `scope` functions for reusable query logic.
- **Transactions**: Use `db.Transaction(func(tx *gorm.DB) error { ... })` for atomicity.
- **Migrations**: Always create a new `.sql` migration file for schema changes. Do not modify existing migrations.

## 3. Architecture Guidelines

- **Depend on abstractions**: `service` packages should depend on `repository` interfaces, not structs.
- **No Globals**: Avoid global state. Use dependency injection (via `fx` or constructors).
- **Separation of Concerns**:
    - `service/`: Business logic only.
    - `server/`: HTTP transport (JSON decoding, status codes).
    - `repository/`: SQL queries.

## 4. Submitting a Pull Request

1.  Create a new branch: `feature/my-new-feature` or `fix/issue-123`.
2.  Write tests for your changes.
3.  Ensure `go test ./...` passes.
4.  Open a PR with a clear description of the problem and solution.

## 5. Security

- **Secrets**: Never commit API keys or secrets.
- **Input Validation**: Validate all inputs at the HTTP layer (request DTOs).
- **SQL Injection**: Use parameterized queries (Gorm does this by default).

Thank you for helping us build Railzway!
