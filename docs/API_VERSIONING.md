# Railzway API Versioning Strategy

[← Back to Documentation Index](index.md)

This document outlines the strategy for managing API versions in the Railzway OSS project.

## 1. Versioning Strategy

We use **URL Path Versioning** to manage API lifecycle.
- **Current Stable**: `/api/v1/*`
- **Future Major**: `/api/v2/*`

This approach is chosen for:
- **Clarity**: Explicit version in URL (easy to debug/log).
- **Cacheability**: Versions are distinct resources.
- **Browser/Tooling Support**: Easier to use with curl/Postman than header-based versioning.

## 2. Implementing a New Version (v2)

When introducing breaking changes (e.g., changing response structure, removing fields), create a new version `v2`.

### Step 1: Create v2 Group in Server
In `internal/server/server.go`:

```go
func (s *Server) RegisterAPIRoutes() {
    rootAPI := s.engine.Group("/api")

    // Maintain V1 (Deprecated but active)
    v1 := rootAPI.Group("/v1")
    s.registerV1Routes(v1)

    // Introduce V2 (Active development)
    v2 := rootAPI.Group("/v2")
    s.registerV2Routes(v2)
}
```

### Step 2: Register Routes
Create separate methods or files for route registration to keep `server.go` clean.

```go
func (s *Server) registerV1Routes(g *gin.RouterGroup) {
    g.GET("/customers", s.ListCustomersV1)
}

func (s *Server) registerV2Routes(g *gin.RouterGroup) {
    // New handler with different signature/response
    g.GET("/customers", s.ListCustomersV2)
}
```

## 3. Code Organization Recommendations

Currently, handlers are methods on the `Server` struct. For better scalability with multiple versions, we recommend refactoring handlers into dedicated packages:

**Recommended Structure:**
```
internal/
  api/
    v1/
      handlers/
        customer.go  // func ListCustomers(c *gin.Context)
        invoice.go
      router.go      // RegisterRoutes(r *gin.RouterGroup)
    v2/
      handlers/
        customer.go  // struct CustomerHandler ...
      router.go
```

**Transitioning from current state:**
1.  Keep V1 handlers as `Server` methods (legacy).
2.  Implement V2 handlers in new `internal/api/v2` package.
3.  Inject dependencies (Services) into V2 handlers.

## 4. Deprecation Policy (OSS)

For OSS projects, clear communication is key:
1.  **Mark Deprecated**: Add `Deprecation: true` header or field in Swagger V1.
2.  **Sunset Period**: Support V1 for at least 6-12 months after V2 release.
3.  **Communication**: Use Release Notes and `CHANGELOG.md`.

## 5. Breaking Changes Checklist
Before creating `v2`:
- [ ] Are fields renamed? (e.g. `user_id` -> `customerId`)
- [ ] Is pagination logic changed?
- [ ] Are error codes changed?
- [ ] Is the resource structure fundamentally different?

If "Yes", bump version.
If "No" (additive changes), stick to `v1`.
