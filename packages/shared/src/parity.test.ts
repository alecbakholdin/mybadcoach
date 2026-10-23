import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

import type { MatchEvent } from './events';
import { reducePoint } from './index';
import type { PlayerStats, PointDerivedState, StartingRole } from './state';

/**
 * One shared fixtures directory at the repo root, executed by both this suite
 * and `server/internal/reduce/parity_test.go`. See fixtures/README.md for the
 * format and equality rules.
 */
const FIXTURES_DIR = fileURLToPath(new URL('../../../fixtures', import.meta.url));

const REQUIRED_FIXTURES = [
  '01-empty-point.json',
  '02-out-of-order-sort.json',
  '03-duplicate-id-dedupe.json',
];

/**
 * The full parity suite: every MatchEvent type plus the documented edge cases.
 * Both ports run this exact count.
 */
const EXPECTED_FIXTURE_COUNT = 24;

const PLAYER_STAT_KEYS = [
  'assists',
  'blocks',
  'completions',
  'drops',
  'goals',
  'touches',
  'turnovers',
] as const;

interface Fixture {
  file: string;
  name: string;
  initialRole: StartingRole;
  events: MatchEvent[];
  expected: PointDerivedState;
}

function readFixtures(): Fixture[] {
  if (!existsSync(FIXTURES_DIR)) {
    return [];
  }

  return readdirSync(FIXTURES_DIR)
    .filter((file) => file.endsWith('.json'))
    .sort()
    .map((file) => {
      const parsed = JSON.parse(readFileSync(join(FIXTURES_DIR, file), 'utf8')) as Omit<Fixture, 'file'>;
      return { file, ...parsed };
    });
}

/**
 * Canonical JSON: object keys sorted by code unit, arrays in order, numbers as
 * integers, no whitespace. Undefined is a hard error, so an accidental
 * undefined field can never be silently dropped.
 */
function canonicalize(value: unknown): string {
  if (value === null) {
    return 'null';
  }
  if (value === undefined) {
    throw new Error('derived state contains an undefined value');
  }
  if (Array.isArray(value)) {
    return `[${value.map(canonicalize).join(',')}]`;
  }
  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
      a < b ? -1 : a > b ? 1 : 0,
    );
    return `{${entries.map(([key, entry]) => `${JSON.stringify(key)}:${canonicalize(entry)}`).join(',')}}`;
  }
  if (typeof value === 'number') {
    if (!Number.isInteger(value)) {
      throw new Error(`derived state contains a non-integer number: ${value}`);
    }
    return String(value);
  }
  if (typeof value === 'string' || typeof value === 'boolean') {
    return JSON.stringify(value);
  }
  throw new Error(`derived state contains an unsupported value: ${typeof value}`);
}

/** The never-null invariant: no null or undefined anywhere in derived state. */
function assertNoNullish(value: unknown, path = '$'): void {
  if (value === null || value === undefined) {
    throw new Error(`${path} must never be null or undefined`);
  }
  if (Array.isArray(value)) {
    value.forEach((entry, index) => assertNoNullish(entry, `${path}[${index}]`));
  } else if (typeof value === 'object') {
    for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
      assertNoNullish(entry, `${path}.${key}`);
    }
  }
}

/** Every player-stats entry is complete, zero-filled, and non-negative. */
function assertZeroFilledStats(playerStats: Record<string, PlayerStats>): void {
  for (const [playerId, stats] of Object.entries(playerStats)) {
    expect(Object.keys(stats).sort(), `stats keys for ${playerId}`).toEqual([...PLAYER_STAT_KEYS]);
    for (const [stat, value] of Object.entries(stats)) {
      expect(Number.isInteger(value) && value >= 0, `${playerId}.${stat} = ${value}`).toBe(true);
    }
  }
}

const fixtures = readFixtures();

describe('shared parity fixtures', () => {
  it('ships all 24 parity fixtures, including the three tracer bullets', () => {
    expect(fixtures).toHaveLength(EXPECTED_FIXTURE_COUNT);
    expect(fixtures.map((fixture) => fixture.file)).toEqual(expect.arrayContaining(REQUIRED_FIXTURES));
  });

  for (const fixture of fixtures) {
    it(`${fixture.file}: ${fixture.name}`, () => {
      const actual = reducePoint(fixture.events, fixture.initialRole);

      assertNoNullish(actual);
      assertZeroFilledStats(actual.playerStats);
      expect(canonicalize(actual)).toBe(canonicalize(fixture.expected));
    });
  }
});
