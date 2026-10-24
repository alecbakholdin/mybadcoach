<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { events as eventClient, seasons as seasonClient } from '#lib/api';
	import { formErrorOf } from '#lib/forms';
	import Icon from '#lib/Icon.svelte';
	import { displayName } from '#lib/players';
	import RoleBadge from '#lib/RoleBadge.svelte';
	import SeasonRosterTable from '#lib/SeasonRosterTable.svelte';
	import SeasonSettings from '#lib/SeasonSettings.svelte';
	import type { SeasonDetailData } from '#lib/structure-data';
	import TeamSettings from '#lib/TeamSettings.svelte';
	import type { Event } from '$shared/gen/mybadcoach/events/v1/events_pb';

	let { data }: { data: SeasonDetailData } = $props();

	let tab = $state<'events' | 'roster' | 'settings'>('events');

	// --- Events tab: compact create form + ledger ---
	let showCreateEvent = $state(false);
	let eventName = $state('');
	let eventStart = $state('');
	let eventEnd = $state('');
	let eventLocation = $state('');
	let eventFieldErrors = $state<Record<string, string>>({});
	let eventError = $state('');
	let creatingEvent = $state(false);

	// --- Roster tab: stats table, selection, and multi-add ---
	let selectedSlugs = $state<string[]>([]);
	let rosterError = $state('');
	let removingSelected = $state(false);

	let searchQuery = $state('');
	let pickedSlugs = $state<string[]>([]);
	let addError = $state('');
	let addingPlayers = $state(false);

	/** Team players not yet registered on this season's roster. */
	const availablePlayers = $derived(
		data.teamPlayers.filter(
			(player) => !data.roster.some((rostered) => rostered.slug === player.slug),
		),
	);

	const pickedPlayers = $derived(
		pickedSlugs.flatMap((slug) => {
			const player = data.teamPlayers.find((entry) => entry.slug === slug);
			return player === undefined ? [] : [player];
		}),
	);

	const suggestions = $derived.by(() => {
		const query = searchQuery.trim().toLowerCase();
		if (query === '') {
			return [];
		}
		return availablePlayers.filter(
			(player) =>
				!pickedSlugs.includes(player.slug) && displayName(player).toLowerCase().includes(query),
		);
	});

	/** One-line summary of an event's dates and location, omitting blanks. */
	function eventMeta(event: Event): string {
		let dates = event.startDate;
		if (event.endDate !== '' && event.endDate !== event.startDate) {
			dates = dates === '' ? event.endDate : `${dates} – ${event.endDate}`;
		}
		return [dates, event.location].filter((part) => part !== '').join(' · ');
	}

	function seasonRange(start: string, end: string): string {
		if (start !== '' && end !== '') {
			return `${start} – ${end}`;
		}
		return start !== '' ? start : end;
	}

	async function handleCreateEvent(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		eventError = '';
		eventFieldErrors = {};
		creatingEvent = true;
		try {
			await eventClient.createEvent({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				name: eventName,
				startDate: eventStart === '' ? undefined : eventStart,
				endDate: eventEnd === '' ? undefined : eventEnd,
				location: eventLocation === '' ? undefined : eventLocation,
			});
			eventName = '';
			eventStart = '';
			eventEnd = '';
			eventLocation = '';
			showCreateEvent = false;
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not create the event.');
			eventFieldErrors = outcome.fieldErrors;
			eventError = outcome.message;
		} finally {
			creatingEvent = false;
		}
	}

	function pickPlayer(slug: string): void {
		pickedSlugs = [...new Set([...pickedSlugs, slug])];
	}

	function unpickPlayer(slug: string): void {
		pickedSlugs = pickedSlugs.filter((entry) => entry !== slug);
	}

	async function handleAddPlayers(): Promise<void> {
		if (pickedSlugs.length === 0) {
			return;
		}
		addError = '';
		addingPlayers = true;
		try {
			for (const slug of pickedSlugs) {
				await seasonClient.addSeasonRosterEntry({
					teamSlug: data.teamSlug,
					seasonSlug: data.seasonSlug,
					playerSlug: slug,
				});
			}
			pickedSlugs = [];
			searchQuery = '';
			await refreshAll();
		} catch {
			addError = 'Could not add the selected players.';
		} finally {
			addingPlayers = false;
		}
	}

	async function handleRemoveSelected(): Promise<void> {
		if (selectedSlugs.length === 0) {
			return;
		}
		rosterError = '';
		removingSelected = true;
		try {
			for (const slug of selectedSlugs) {
				await seasonClient.removeSeasonRosterEntry({
					teamSlug: data.teamSlug,
					seasonSlug: data.seasonSlug,
					playerSlug: slug,
				});
			}
			selectedSlugs = [];
			await refreshAll();
		} catch {
			rosterError = 'Could not remove the selected players.';
		} finally {
			removingSelected = false;
		}
	}
</script>

<svelte:head>
	<title>{data.season?.name ?? 'Season'} · My Bad, Coach</title>
</svelte:head>

<main class="structure-page">
	{#if data.season === undefined}
		<div class="card">
			<h1>Season not found</h1>
			<p class="muted">This season does not exist or you are not a member of its team.</p>
		</div>
	{:else}
		<div class="route" aria-label="Breadcrumb">
			<span class="route-seg">/teams</span>
			<span class="route-seg">/{data.teamSlug}</span>
			<span class="route-seg">/seasons</span>
			<span class="route-seg">/{data.seasonSlug}</span>
		</div>

		<header class="structure-header">
			<div class="title-row">
				<h1>{data.season.name}</h1>
				<RoleBadge role={data.role} />
			</div>
			{#if seasonRange(data.season.startDate, data.season.endDate) !== ''}
				<div class="head-sub">
					<span class="summary">
						<Icon name="calendar" />
						{seasonRange(data.season.startDate, data.season.endDate)}
					</span>
				</div>
			{/if}
		</header>

		<div class="tabbed">
			<div class="tabs" role="tablist" aria-label="Season">
				<button
					class="tab"
					type="button"
					role="tab"
					aria-selected={tab === 'events'}
					onclick={() => {
						tab = 'events';
					}}
				>
					<Icon name="calendar" />
					Events
				</button>
				<button
					class="tab"
					type="button"
					role="tab"
					aria-selected={tab === 'roster'}
					onclick={() => {
						tab = 'roster';
					}}
				>
					<Icon name="chart" />
					Roster
				</button>
				<button
					class="tab"
					type="button"
					role="tab"
					aria-selected={tab === 'settings'}
					onclick={() => {
						tab = 'settings';
					}}
				>
					<Icon name="gear" />
					Settings
				</button>
			</div>

			{#if tab === 'events'}
				<div class="panel" role="tabpanel" aria-label="Events">
					<div class="section-head">
						<h2 class="section-title">
							<Icon name="calendar" />
							Events
						</h2>
						<button
							class="btn btn-secondary btn-compact"
							type="button"
							onclick={() => {
								showCreateEvent = !showCreateEvent;
							}}
						>
							<Icon name="plus" />
							New event
						</button>
					</div>

					{#if showCreateEvent}
						<form class="create-event-form" onsubmit={handleCreateEvent}>
							<div class="field">
								<label for="create-event-name">Event name</label>
								<input id="create-event-name" type="text" required bind:value={eventName} />
								{#if eventFieldErrors.name}
									<p class="field-error">{eventFieldErrors.name}</p>
								{/if}
							</div>
							<div class="form-grid">
								<div class="field">
									<label for="create-event-start">Start date</label>
									<input id="create-event-start" type="date" bind:value={eventStart} />
								</div>
								<div class="field">
									<label for="create-event-end">End date</label>
									<input id="create-event-end" type="date" bind:value={eventEnd} />
								</div>
							</div>
							<div class="field">
								<label for="create-event-location">Location</label>
								<input id="create-event-location" type="text" bind:value={eventLocation} />
							</div>
							{#if eventError !== ''}
								<p class="form-error" role="alert">{eventError}</p>
							{/if}
							<div class="btn-row">
								<button class="btn btn-primary" type="submit" disabled={creatingEvent}>
									{creatingEvent ? 'Creating event…' : 'Create event'}
								</button>
								<button
									class="btn btn-tertiary"
									type="button"
									onclick={() => {
										showCreateEvent = false;
									}}
								>
									Cancel
								</button>
							</div>
						</form>
					{/if}

					{#if data.events.length === 0}
						<p class="muted">No events yet.</p>
					{:else}
						<ul class="ledger" aria-label="Events">
							{#each data.events as event (event.slug)}
								<li>
									<a
										class="ledger-row"
										href={resolve('/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]', {
											team_slug: data.teamSlug,
											season_slug: data.seasonSlug,
											event_slug: event.slug,
										})}
									>
										<span class="row-icon" aria-hidden="true">
											<Icon name="calendar" />
										</span>
										<span class="row-main">
											<span class="row-title">{event.name}</span>
											{#if eventMeta(event) !== ''}
												<span class="row-meta">{eventMeta(event)}</span>
											{/if}
										</span>
										<span class="row-open" aria-hidden="true">
											<Icon name="chevron-right" />
										</span>
									</a>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			{:else if tab === 'roster'}
				<div class="panel" role="tabpanel" aria-label="Roster">
					<div class="section-head">
						<h2 class="section-title">
							<Icon name="chart" />
							Player statistics
						</h2>
					</div>

					<p class="roster-summary">
						<strong>{data.rosterTotal}</strong>
						{data.rosterTotal === 1 ? 'player' : 'players'} on this season's roster.
					</p>

					{#if availablePlayers.length > 0}
						<div class="add-block">
							<h3 class="add-head">
								<Icon name="user-plus" />
								Add players
							</h3>

							<div class="autocomplete">
								<div class="search-wrap">
									<Icon name="search" />
									<label class="sr-only" for="player-search">Search team players</label>
									<input
										class="input"
										id="player-search"
										type="text"
										placeholder="Search team players"
										autocomplete="off"
										role="combobox"
										aria-expanded={suggestions.length > 0}
										aria-controls="player-suggestions"
										aria-autocomplete="list"
										bind:value={searchQuery}
									/>
								</div>

								{#if pickedPlayers.length > 0}
									<ul class="chips" aria-label="Selected players">
										{#each pickedPlayers as player (player.slug)}
											<li class="chip">
												<span>{displayName(player)}</span>
												<button
													class="chip-remove"
													type="button"
													aria-label={`Remove ${displayName(player)}`}
													onclick={() => {
														unpickPlayer(player.slug);
													}}
												>
													<Icon name="close" />
												</button>
											</li>
										{/each}
									</ul>
								{/if}

								{#if searchQuery.trim() !== ''}
									<div
										class="suggestions"
										id="player-suggestions"
										role="listbox"
										aria-label="Suggested players"
									>
										{#each suggestions as player (player.slug)}
											<button
												class="suggestion"
												type="button"
												role="option"
												aria-selected={false}
												onclick={() => {
													pickPlayer(player.slug);
												}}
											>
												<span class="suggestion-name">{displayName(player)}</span>
												<span class="suggestion-plus" aria-hidden="true">
													<Icon name="plus" />
												</span>
											</button>
										{:else}
											<p class="suggestion-empty">No matching players.</p>
										{/each}
									</div>
								{/if}

								{#if addError !== ''}
									<p class="form-error" role="alert">{addError}</p>
								{/if}

								<div class="add-actions">
									<button
										class="btn btn-primary"
										type="button"
										disabled={pickedSlugs.length === 0 || addingPlayers}
										onclick={handleAddPlayers}
									>
										{addingPlayers
											? 'Adding…'
											: `Add ${pickedSlugs.length} ${pickedSlugs.length === 1 ? 'player' : 'players'}`}
									</button>
								</div>
							</div>
						</div>
					{/if}

					{#if selectedSlugs.length > 0}
						<div class="bulk-bar">
							<span class="bulk-count" aria-live="polite">
								{selectedSlugs.length} selected
							</span>
							<button
								class="btn btn-danger btn-compact"
								type="button"
								disabled={removingSelected}
								onclick={handleRemoveSelected}
							>
								<Icon name="trash" />
								Remove selected
							</button>
						</div>
					{/if}

					{#if rosterError !== ''}
						<p class="form-error" role="alert">{rosterError}</p>
					{/if}

					{#if data.roster.length === 0}
						<p class="muted">No players on this season's roster yet.</p>
					{:else}
						<SeasonRosterTable players={data.roster} bind:selectedSlugs />
					{/if}
				</div>
			{:else}
				<div class="panel" role="tabpanel" aria-label="Settings">
					<div class="settings-stack">
						<TeamSettings teamSlug={data.teamSlug} teamName={data.teamName} role={data.role} />
						<SeasonSettings
							teamSlug={data.teamSlug}
							seasonSlug={data.seasonSlug}
							name={data.season.name}
							startDate={data.season.startDate}
							endDate={data.season.endDate}
							role={data.role}
						/>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</main>
