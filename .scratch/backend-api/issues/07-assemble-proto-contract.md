# Assemble the proto contract

Type: task
Status: resolved
Blocked by: 04, 05, 06

## Question

Assemble the buf-managed `.proto` tree from the resolutions of tickets 04, 05, 06: services,
messages, error details. Wire buf lint (strict) + codegen into the Go and TypeScript workspaces —
generated types replace the hand-written `events.ts`/`state.go` and their mirrors. Prove the
generated types preserve the parity contract: **fixtures stay hand-written, human-readable JSON**,
and the canonical-JSON equality rules hold unchanged. Deliverable: `proto/` tree + buf config +
codegen wiring + parity proof.

## Answer

Resolved 2026-10-22. Approved by Alec as drafted. Deliverables in `assets/proto/`:

- `buf.yaml` — STANDARD lint, FILE breaking.
- `buf.gen.yaml` — Go + connect-go + es plugins (out paths finalized at implementation).
- `mybadcoach/errors/v1` — `ErrorDetail { code, field_violations }`.
- `mybadcoach/auth/v1` — `AuthService` + `User`.
- `mybadcoach/teams/v1` — `TeamService`: CRUD + `DeleteTeam` + membership ops (last_owner
  guard) + `ListMyTeams` + `GetTeamRecord`.
- `mybadcoach/seasons/v1` — `SeasonService`: CRUD + season rosters + `CopySeasonRoster`.
- `mybadcoach/events/v1` — `EventService`: CRUD + event rosters.
- `mybadcoach/players/v1` — `PlayerService`: CRUD.
- `mybadcoach/matches/v1` — `MatchService`: match CRUD + `AppendMatchEvents` +
  `ListMatchEvents` + `GetMatchState` + `GetMatchStats`; `MatchEvent` (oneof payloads),
  `PointDerivedState`, `PlayerStats`/`DerivedStats`, `View`, `StartingRole`.

Micro-decisions accepted: gameplay on `MatchService`; no `ListTeams`; `created_at` not
exposed on resource messages; ISO-8601 string timestamps; buf-lint enum names with the
parity harness mapping fixture short names; parity proof = the 24-fixture suite green
against generated-type reducers.
