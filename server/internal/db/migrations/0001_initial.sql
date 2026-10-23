-- 0001-initial-schema: domain model + auth + migration bookkeeping.
--
-- Draft for wayfinder ticket 02 (backend-api map), amended by ticket 05 (hard deletes).
-- Mirrors the planned schema in docs/architecture.md plus the auth additions settled in
-- grilling: auth_identities, password_credentials, sessions, schema_migrations.
--
-- Conventions: STRICT tables; TEXT ids (client-generated for match_events so offline
-- replays are idempotent, server-generated elsewhere); hard deletes with FK cascades — no
-- storage tombstones (match_events.deleted is the event-log tombstone of the parity
-- contract, not delete semantics); slugs per ADR-0006; no stats tables per ADR-0004.
-- JSON lives in TEXT columns with engine-level CHECK (json_valid) validation.

CREATE TABLE schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
) STRICT;

CREATE TABLE users (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE auth_identities (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    email            TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    UNIQUE (provider, provider_subject)
) STRICT;

CREATE INDEX auth_identities_user ON auth_identities (user_id);

CREATE TABLE password_credentials (
    user_id       TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    updated_at    TEXT NOT NULL
) STRICT;

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX sessions_user ON sessions (user_id);

CREATE TABLE teams (
    id         TEXT PRIMARY KEY,
    slug       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE TABLE team_memberships (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id    TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at TEXT NOT NULL,
    UNIQUE (user_id, team_id)
) STRICT;

CREATE INDEX team_memberships_team ON team_memberships (team_id);

CREATE TABLE seasons (
    id         TEXT PRIMARY KEY,
    team_id    TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    start_date TEXT,
    end_date   TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (team_id, slug)
) STRICT;

CREATE TABLE events (
    id         TEXT PRIMARY KEY,
    season_id  TEXT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    start_date TEXT,
    end_date   TEXT,
    location   TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (season_id, slug)
) STRICT;

CREATE TABLE matches (
    id            TEXT PRIMARY KEY,
    event_id      TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    our_team_id   TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    opponent_name TEXT NOT NULL,
    slug          TEXT NOT NULL,
    scheduled_at  TEXT,
    point_cap     INTEGER,
    created_at    TEXT NOT NULL,
    UNIQUE (event_id, slug)
) STRICT;

CREATE INDEX matches_event ON matches (event_id);

CREATE TABLE players (
    id         TEXT PRIMARY KEY,
    team_id    TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    names      TEXT NOT NULL CHECK (json_valid(names)),
    slug       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (team_id, slug)
) STRICT;

CREATE TABLE season_roster_entries (
    id         TEXT PRIMARY KEY,
    season_id  TEXT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    player_id  TEXT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    UNIQUE (season_id, player_id)
) STRICT;

CREATE TABLE event_roster_entries (
    id         TEXT PRIMARY KEY,
    event_id   TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    player_id  TEXT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    UNIQUE (event_id, player_id)
) STRICT;

CREATE TABLE match_events (
    id           TEXT PRIMARY KEY,
    match_id     TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    point_number INTEGER NOT NULL,
    effective_ts INTEGER NOT NULL,
    type         TEXT NOT NULL,
    payload      TEXT NOT NULL CHECK (json_valid(payload)),
    deleted      INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL
) STRICT;

CREATE INDEX match_events_match_time ON match_events (match_id, effective_ts);
