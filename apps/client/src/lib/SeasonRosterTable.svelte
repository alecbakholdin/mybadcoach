<script lang="ts">
	import Icon from '#lib/Icon.svelte';
	import { displayName } from '#lib/players';
	import {
		compareRosterRows,
		NUMERIC_STAT_COLUMNS,
		type RosterRow,
		type RosterSortKey,
		type SortDirection,
	} from '#lib/roster';
	import type { Player } from '$shared/gen/mybadcoach/players/v1/players_pb';

	let { players, selectedSlugs = $bindable([]) }: { players: Player[]; selectedSlugs?: string[] } =
		$props();

	let sortKey = $state<RosterSortKey>('name');
	let sortDir = $state<SortDirection>('asc');

	const rows = $derived<RosterRow[]>(
		players.map((player) => ({ slug: player.slug, name: displayName(player) })),
	);
	const sortedRows = $derived([...rows].sort((a, b) => compareRosterRows(a, b, sortKey, sortDir)));

	const allSelected = $derived(players.length > 0 && selectedSlugs.length === players.length);

	function toggleSort(key: RosterSortKey): void {
		if (sortKey !== key) {
			sortKey = key;
			sortDir = 'asc';
		} else {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		}
	}

	function ariaSort(key: RosterSortKey): 'ascending' | 'descending' | 'none' {
		if (sortKey !== key) {
			return 'none';
		}
		return sortDir === 'asc' ? 'ascending' : 'descending';
	}

	function toggleSelectAll(checked: boolean): void {
		selectedSlugs = checked ? players.map((player) => player.slug) : [];
	}

	function toggleSelect(slug: string, checked: boolean): void {
		selectedSlugs = checked
			? [...new Set([...selectedSlugs, slug])]
			: selectedSlugs.filter((selected) => selected !== slug);
	}
</script>

<div class="table-shell">
	<table class="stats-table">
		<caption class="sr-only">Player season statistics</caption>
		<thead>
			<tr>
				<th class="col-check" scope="col">
					<input
						class="check-all"
						type="checkbox"
						aria-label="Select all players"
						checked={allSelected}
						onchange={(event) => {
							toggleSelectAll(event.currentTarget.checked);
						}}
					/>
				</th>
				<th class="col-player" scope="col" aria-sort={ariaSort('name')}>
					<button
						class="sort-btn"
						type="button"
						onclick={() => {
							toggleSort('name');
						}}
					>
						<span>Player</span>
						<Icon name="sort" class="sort-arrow" />
					</button>
				</th>
				{#each NUMERIC_STAT_COLUMNS as column (column.key)}
					<th
						class="col-num"
						class:col-wide={column.wide}
						class:col-delta={column.delta}
						scope="col"
						aria-sort={ariaSort(column.key)}
					>
						<button
							class="sort-btn"
							type="button"
							aria-label={`Sort by ${column.title.toLowerCase()}`}
							title={column.title}
							onclick={() => {
								toggleSort(column.key);
							}}
						>
							<span>{column.label}</span>
							<Icon name="sort" class="sort-arrow" />
						</button>
					</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each sortedRows as row (row.slug)}
				<tr class:is-selected={selectedSlugs.includes(row.slug)}>
					<td class="col-check">
						<input
							class="row-check"
							type="checkbox"
							aria-label={`Select ${row.name}`}
							checked={selectedSlugs.includes(row.slug)}
							onchange={(event) => {
								toggleSelect(row.slug, event.currentTarget.checked);
							}}
						/>
					</td>
					<th scope="row" class="col-player">
						<span class="player-name">{row.name}</span>
					</th>
					{#each NUMERIC_STAT_COLUMNS as column (column.key)}
						<td class="col-num" class:col-wide={column.wide} class:col-delta={column.delta}> — </td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>
