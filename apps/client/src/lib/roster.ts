/**
 * Roster statistics-table model: the sortable column keys, a per-row stat line,
 * and the sort comparator the season roster table uses.
 *
 * Stat values are `undefined` until the gameplay effort supplies real match
 * events to aggregate; the table renders those cells as em-dashes. The
 * comparator therefore treats `undefined` as "sorts last" so the current
 * placeholder data degrades to a stable name-sorted list.
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

/** A single roster row: identity plus the (currently placeholder) stat line. */
export interface RosterRow {
	slug: string;
	name: string;
	stats?: {
		goals?: number;
		assists?: number;
		blocks?: number;
		turnovers?: number;
		drops?: number;
		touches?: number;
		completions?: number;
		plusMinus?: number;
	};
}

type NumericRosterSortKey = Exclude<RosterSortKey, 'name'>;

/** `a` and `b` are equal when both values are absent; absent sorts after present. */
function compareOptional(a: number | undefined, b: number | undefined): number {
	if (a === undefined && b === undefined) {
		return 0;
	}
	if (a === undefined) {
		return 1;
	}
	if (b === undefined) {
		return -1;
	}
	return a - b;
}

/**
 * Compares two roster rows by a sort key. A name sort is by display name; a
 * numeric sort compares the stat value (undefined sorts last). Ties fall back
 * to the display name so every order is deterministic.
 */
export function compareRosterRows(
	a: RosterRow,
	b: RosterRow,
	key: RosterSortKey,
	direction: SortDirection,
): number {
	const sign = direction === 'asc' ? 1 : -1;
	if (key === 'name') {
		const compared = a.name.localeCompare(b.name);
		return compared !== 0 ? sign * compared : a.slug.localeCompare(b.slug);
	}
	const numericKey: NumericRosterSortKey = key;
	const compared = compareOptional(a.stats?.[numericKey], b.stats?.[numericKey]);
	return compared !== 0 ? sign * compared : a.name.localeCompare(b.name);
}
