# Browser Surfaces & Deployment Recommendation

This document describes the recommended deployment model for Railzway browser-facing surfaces when frontend assets are hosted separately from compute.

The examples use placeholder domains such as `*.yourdomain.com` so they can be adapted to any environment.

## 1. Recommended Public Surface Split

Railzway has three browser-facing surfaces with different user journeys and trust boundaries:

- `manage.yourdomain.com` for the admin console
- `checkout.yourdomain.com` for the hosted checkout experience
- `customer.yourdomain.com` for the customer portal

Each surface should keep a stable public origin even if the backend compute moves between Nomad, Compute Engine, or Kubernetes.

## 2. Recommended Deployment Model

The recommended model is:

- frontend assets are hosted at the public surface origin
- backend compute runs on internal or separately managed infrastructure
- an edge proxy or rewrite layer forwards browser API requests to the correct backend service

Example:

- `manage.yourdomain.com` serves the admin UI bundle
- `manage.yourdomain.com/admin/*` is proxied to the `cmd/admin` service
- `checkout.yourdomain.com/checkout/*` is proxied to the `cmd/checkout` service
- `customer.yourdomain.com/customer/*` is proxied to the `cmd/customer` service

This keeps the browser view same-origin per product surface while allowing compute portability behind the proxy.

## 3. Alternative: Shared API Domain

An alternative public shape is to keep the browser surfaces separate while centralizing API traffic under a single public API domain.

Example:

- `manage.yourdomain.com` serves the admin UI bundle
- `checkout.yourdomain.com` serves the checkout UI bundle
- `customer.yourdomain.com` serves the customer portal bundle
- `api.yourdomain.com/admin/*` routes to `cmd/admin`
- `api.yourdomain.com/checkout/*` routes to `cmd/checkout`
- `api.yourdomain.com/customer/*` routes to `cmd/customer`
- `api.yourdomain.com/api/*` can remain available for public or shared API traffic when required

This model is valid, but it should be treated as an intentional cross-origin architecture.

Tradeoffs:

- browser-visible backend origin becomes `api.yourdomain.com`
- CORS becomes a required production dependency
- cookie and CSRF policy become more sensitive, especially for the admin surface
- API routing and observability become centralized under one public host
- ingress and WAF policy can be simpler to manage because one API edge handles all browser API traffic

This model is a reasonable fit when:

- API operations must be managed under one public gateway
- centralized rate limiting or API edge controls are more important than same-origin browser simplicity
- the team is comfortable operating cross-origin cookies, CSRF, and CORS as first-class production concerns

## 4. Why Same-Origin Proxy Is Preferred

Compared with exposing browser traffic directly to `admin-api.yourdomain.com` or similar subdomains, same-origin proxying has better operational and security properties for the current Railzway architecture:

- backend origin is not exposed as a separate public browser host
- session cookies stay simpler because browser requests remain same-origin
- CSRF handling remains aligned with cookie-based admin auth
- there is no cross-origin CORS dependency for the steady-state production path
- Nomad-to-Kubernetes migration only changes backend routing, not public contracts

This model fits the existing BEFE approach used by the admin, checkout, and customer surfaces.

## 5. Cross-Origin Mode

Cross-origin mode is still possible when needed for development, previews, or temporary transition phases:

- UI origin: `https://preview-manage.yourdomain.com`
- backend origin: `https://admin-api.yourdomain.com`

When using this mode:

- `browser.allowed_origins` must explicitly list the UI origin
- admin session cookies may require `session.cookie_same_site: none`
- production deployments must still use secure cookies and HTTPS

Cross-origin mode should be treated as an exception path, not the long-term default.

## 6. Configuration Guidance

### 6.1. Recommended Production Shape

For the recommended same-origin setup:

- keep `VITE_API_BASE_URL` empty in the admin frontend so requests stay relative
- expose the UI through the public surface origin such as `manage.yourdomain.com`
- proxy `/admin/v1/*` to the admin backend
- leave `browser.allowed_origins` empty unless direct cross-origin browser access is intentionally required

### 6.2. Shared API Domain Shape

For the shared public API domain model:

- point the browser-facing frontend to `https://api.yourdomain.com`
- keep backend routes separated by path such as `/admin/*`, `/checkout/*`, and `/customer/*`
- explicitly list the frontend origins in `browser.allowed_origins`
- review admin cookie settings before rollout, especially `cookie_same_site`

Example:

```yaml
browser:
  allowed_origins:
    - https://manage.yourdomain.com
    - https://checkout.yourdomain.com
    - https://customer.yourdomain.com

session:
  cookie_name: rz_admin_session
  cookie_domain: ""
  cookie_same_site: none
```

This should only be used with HTTPS and explicit CORS policy.

### 6.3. Optional Browser-Origin Allow List

Railzway now supports an explicit browser origin allow list:

```yaml
browser:
  allowed_origins:
    - https://preview-manage.yourdomain.com
    - https://manage.yourdomain.com
```

This is useful when:

- running the frontend from a Vite dev server
- validating Vercel previews against a shared backend
- temporarily separating the UI origin from the API origin

### 6.4. Session Cookie Controls

The admin surface now supports additional session cookie controls:

```yaml
session:
  cookie_name: rz_admin_session
  cookie_domain: ""
  cookie_same_site: lax
```

Guidance:

- use the default host-only cookie when the browser should stay bound to a single surface origin
- only set `cookie_domain` if multiple subdomains intentionally need the same cookie scope
- keep `cookie_same_site: lax` for same-origin production
- use `cookie_same_site: none` only when direct cross-origin browser requests are intentional and HTTPS is enforced

## 7. Infrastructure Portability

Public contracts should stay stable even if compute changes:

- today: proxy -> Nomad job
- later: proxy -> Kubernetes service

The browser should not need to know or care whether the upstream is:

- a Nomad allocation
- a Compute Engine instance group
- a Kubernetes ingress or service mesh

That portability is the main reason to define the public origin and public API paths first, then let infrastructure evolve behind them.

## 8. Summary

Recommended steady-state architecture:

- `manage.yourdomain.com` -> admin UI + `/admin/v1/*` proxy to `cmd/admin`
- `checkout.yourdomain.com` -> checkout UI + `/checkout/v1/*` proxy to `cmd/checkout`
- `customer.yourdomain.com` -> customer portal UI + `/customer/v1/*` proxy to `cmd/customer`

Alternative supported architecture:

- `manage.yourdomain.com` -> admin UI
- `checkout.yourdomain.com` -> checkout UI
- `customer.yourdomain.com` -> customer portal UI
- `api.yourdomain.com/admin/*` -> `cmd/admin`
- `api.yourdomain.com/checkout/*` -> `cmd/checkout`
- `api.yourdomain.com/customer/*` -> `cmd/customer`

Recommended default stance:

- same-origin in production
- cross-origin only when explicitly needed
- keep backend compute portable behind a stable public contract
