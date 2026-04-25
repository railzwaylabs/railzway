# Repository Structure Philosophy

Railzway is intentionally built as a pragmatic monorepo.

This repository is not a textbook implementation of Clean Architecture, Hexagonal Architecture, or full Domain-Driven Design. Some parts of the codebase are organized by domain, while other parts are organized by product surface, delivery context, or deployment topology.

That is an intentional tradeoff, not an accident.

## What This Repository Optimizes For

Railzway prioritizes:

- clear module and surface boundaries
- code that is easy to find and follow
- fast product iteration
- deployment and operational clarity
- practical separation of frontend apps, backend binaries, shared packages, config, and infrastructure assets

The project does not optimize for architectural purity as a goal by itself.

## What This Means In Practice

You will see a mix of structures such as:

- domain-oriented backend modules like `internal/customer`, `internal/invoice`, `internal/subscription`, and `internal/ledger`
- surface-oriented modules like `internal/admin`, `internal/public`, and `internal/customerportal`
- separate entrypoints in `cmd/` for different runtime hosts and topologies
- frontend apps in `apps/` and shared UI packages in `packages/`

This means the codebase is modular, but not rigidly layered according to one architecture doctrine.

## Why

Railzway serves multiple browser-facing and backend-facing surfaces:

- admin operations
- public/API-key access
- customer-facing experiences
- migration and background job binaries

For a codebase like this, forcing every part into one strict architecture style can reduce clarity instead of improving it.

Examples of tradeoffs this repository intentionally accepts:

- a package may be shaped by product surface instead of pure domain modeling
- an entrypoint may compose multiple modules because that reflects a real runtime surface
- deployment topology may influence artifact naming and packaging

## Contributor Guidance

Contributors should optimize for:

- explicit boundaries
- consistent naming
- predictable module ownership
- minimal surprise in folder layout
- correctness and maintainability

Contributors should not assume that every change must be reshaped into strict Clean Architecture or strict DDD terminology to be acceptable.

If a pattern helps clarity, correctness, or maintainability, use it.
If a pattern only adds abstraction without helping the problem at hand, do not force it.

## Rule Of Thumb

Railzway prefers:

- clear over clever
- proportionate structure over dogmatic structure
- practical modularity over architectural performance art

The standard for contributions is not ideological purity.
The standard is whether the code makes the system easier to understand, safer to change, and more correct in production.
