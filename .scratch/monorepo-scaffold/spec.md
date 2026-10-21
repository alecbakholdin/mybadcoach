# Spec: Scaffold monorepo + shared core

## Goal

Establish the repo architecture and the shared event-sourcing core: a monorepo with a dependency-free
`@mybadcoach/shared` package and a Go server, with `reducePoint` proven identical across TypeScript and
Go via a shared parity fixture suite.

## Scope

- Monorepo layout (npm workspaces: `packages/*`, `apps/*`) + Go module at `server/`.
- Shared domain types + `MatchEvent` schema + pure `reducePoint` + stat aggregation.
- Parity fixtures (24) run by Vitest and `go test`.
- SvelteKit client skeleton + Go server stub.
- `docs/architecture.md`, `README.md`, and a grep-able USAU rules reference.

Out of scope: WebSocket `MatchActor`, full UI, SQLite DDL/persistence, auth/permissions, match-level
point lifecycle (time cap, forfeit).

## Settled decisions (see `GLOSSARY.md` + `docs/adr/`)

- Event-sourced gameplay; pure reducer `reducePoint(events, initialRole) -> PointDerivedState` (ADR-0001).
- 7 base stats credited by the reducer; 3 derived stats computed upward (ADR-0002, ADR-0004).
- Faceless opponent (ADR-0003).
- Roster is season- and event-scoped; on-field lineup is derived, not stored (ADR-0005).
- Stable, scoped slugs (ADR-0006).
- Hierarchy Team → Season → Event → Match; season required (ADR-0007).
