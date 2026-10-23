import type { PlayerStats, PointDerivedState } from './state';
import { emptyPlayerStats } from './state';

/**
 * A player's complete stat line: the seven base stats credited by the reducer
 * plus the three stats derived from them (ADR-0002). Derived fields are
 * computed on every aggregation and never stored (ADR-0004).
 */
export interface PlayerStatLine extends PlayerStats {
  receptions: number;
  throwAttempts: number;
  plusMinus: number;
}

/** Per-player stat lines keyed by player id. */
export type PlayerStatLines = Record<string, PlayerStatLine>;

/**
 * Aggregate a match's reduced points into per-player stat totals.
 *
 * Input is the output of `reducePoint` for each point in the match. The
 * returned map is keyed by player id and is empty (never null) when the match
 * has no credited players.
 */
export function aggregateMatchStats(
  points: readonly PointDerivedState[],
): PlayerStatLines {
  const totals: Record<string, PlayerStats> = {};

  for (const point of points) {
    for (const [playerId, stats] of Object.entries(point.playerStats)) {
      // Start every player from a complete zero-filled line: a player credited
      // in one point but absent from another contributes zeros for that point.
      const total = totals[playerId] ?? emptyPlayerStats();
      accumulateBaseStats(total, stats);
      totals[playerId] = total;
    }
  }

  return toStatLines(totals);
}

// aggregateEventStats/aggregateCareerStats are intentional per-level entry
// points over the shared fold (ticket 05 requires both names).
/**
 * Roll per-match stat lines up into totals for an event.
 *
 * Each input is a match's aggregate (`aggregateMatchStats` output). The
 * result has the same shape as a match aggregate, so it can be rendered by
 * the same stat table; it is empty (never null) when no matches are supplied.
 */
export function aggregateEventStats(
  matches: readonly PlayerStatLines[],
): PlayerStatLines {
  return sumStatLines(matches);
}

/**
 * Roll per-match stat lines up into career totals across every match a player
 * has featured in. Takes the same per-match aggregates as `aggregateEventStats`;
 * only the scope of the caller's match list differs.
 */
export function aggregateCareerStats(
  matches: readonly PlayerStatLines[],
): PlayerStatLines {
  return sumStatLines(matches);
}

/** Sum per-match stat lines, recomputing all derived stats from base totals. */
function sumStatLines(matches: readonly PlayerStatLines[]): PlayerStatLines {
  const totals: Record<string, PlayerStats> = {};

  for (const match of matches) {
    for (const [playerId, line] of Object.entries(match)) {
      const total = totals[playerId] ?? emptyPlayerStats();
      // Only the seven base stats are summed. Any derived fields carried by
      // the input are ignored and recomputed by `toStatLines`, so a rollup
      // can never materialize a stale derived value (ADR-0004).
      accumulateBaseStats(total, line);
      totals[playerId] = total;
    }
  }

  return toStatLines(totals);
}

/** Add one player's base stats into a running total, in place. */
function accumulateBaseStats(target: PlayerStats, source: PlayerStats): void {
  target.touches += source.touches;
  target.completions += source.completions;
  target.drops += source.drops;
  target.turnovers += source.turnovers;
  target.blocks += source.blocks;
  target.goals += source.goals;
  target.assists += source.assists;
}

/** Turn base totals into complete stat lines, deriving the three extra stats. */
function toStatLines(totals: Record<string, PlayerStats>): PlayerStatLines {
  const lines: PlayerStatLines = {};
  for (const [playerId, base] of Object.entries(totals)) {
    lines[playerId] = deriveStatLine(base);
  }
  return lines;
}

/** Compute the three derived stats from a base-stat total (ADR-0002). */
function deriveStatLine(base: PlayerStats): PlayerStatLine {
  return {
    ...base,
    // receptions = touches. GLOSSARY defines Touch as "a player catching or
    // possessing the disc" — the action a reception names — and ADR-0002 makes
    // receptions a derived view of the touches the reducer credits. A callahan
    // is logged as DEFENSIVE_BLOCK then unassisted SCORE with no TOUCH, so it
    // is goals=1, touches=0; the spec-gap defaults likewise credit no touch for
    // a SCORE recorded alone. This formula therefore reports 0 receptions for a
    // callahan, while folding goals in (e.g. touches + goals) would not.
    receptions: base.touches,
    // Formula pinned by GLOSSARY: touches + completions − goals − turnovers.
    // Kept literal (never clamped), so a callahan's algebraically negative
    // result is preserved rather than invented into a zero.
    throwAttempts: base.touches + base.completions - base.goals - base.turnovers,
    // Formula pinned by GLOSSARY: goals + assists + blocks − turnovers − drops.
    plusMinus: base.goals + base.assists + base.blocks - base.turnovers - base.drops,
  };
}
