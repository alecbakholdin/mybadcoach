/**
 * Roster statistics-table model: the sortable column keys and the sort
 * comparator the season roster table uses.
 *
 * Every stat column is a placeholder (the gameplay effort supplies real match
 * events later), so a numeric sort is a stable no-op for now — there is no
 * value to compare. The name column sorts the roster by display name. The
 * numeric comparison is reintroduced in `compareRosterRows` once real
 * per-player stats land.
 */

export type RosterSortKey =
	| 'name'
	| 'goals'
	| 'assists'
	| 'blocks'
	| 'turnovers'
	| 'drops'
	| 'touches'
	| 'completions'
	| 'plusMinus';

export type SortDirection = 'asc' | 'desc';

/** The numeric stat columns, in table order. */
export const NUMERIC_STAT_COLUMNS: ReadonlyArray<{
	key: RosterSortKey;
	label: string;
	title: string;
	wide?: boolean;
	delta?: boolean;
}> = [
	{ key: 'goals', label: 'G', title: 'Goals' },
	{ key: 'assists', label: 'A', title: 'Assists' },
	{ key: 'blocks', label: 'B', title: 'Blocks' },
	{ key: 'turnovers', label: 'T', title: 'Turnovers' },
	{ key: 'drops', label: 'D', title: 'Drops' },
	{ key: 'touches', label: 'Touches', title: 'Touches', wide: true },
	{ key: 'completions', label: 'Completions', title: 'Completions', wide: true },
	{ key: 'plusMinus', label: '+/−', title: 'Plus-minus', delta: true },
];

/** A single roster row: identity only — stats are placeholders for now. */
export interface RosterRow {
	slug: string;
	name: string;
}

/**
 * Compares two roster rows by a sort key. A name sort is by display name; a
 * numeric-column sort is a stable no-op until real stats exist.
 */
export function compareRosterRows(
	a: RosterRow,
	b: RosterRow,
	key: RosterSortKey,
	direction: SortDirection,
): number {
	if (key === 'name') {
		const sign = direction === 'asc' ? 1 : -1;
		const compared = a.name.localeCompare(b.name);
		return compared !== 0 ? sign * compared : a.slug.localeCompare(b.slug);
	}
	// Placeholder stats carry no values, so the sort preserves the current
	// (registration) order. A real numeric comparison lands here with the
	// gameplay data.
	return 0;
}
