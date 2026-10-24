<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { events as eventClient, seasons as seasonClient } from '#lib/api';
	import { fieldErrorsOf } from '#lib/api/errors';
	import type { SeasonDetailData } from '#lib/structure-data';
	import type { Event } from '$shared/gen/mybadcoach/events/v1/events_pb';

	let { data }: { data: SeasonDetailData } = $props();

	const isOwner = $derived(data.role === 'owner');

	let editName = $derived(data.season?.name ?? '');
	let editStart = $derived(data.season?.startDate ?? '');
	let editEnd = $derived(data.season?.endDate ?? '');
	let fieldErrors = $state<Record<string, string>>({});
	let saveError = $state('');
	let saving = $state(false);

	let deleting = $state(false);
	let deleteError = $state('');

	let eventName = $state('');
	let eventStart = $state('');
	let eventEnd = $state('');
	let eventLocation = $state('');
	let eventFieldErrors = $state<Record<string, string>>({});
	let eventError = $state('');
	let creatingEvent = $state(false);

	let selectedPlayerSlug = $state('');
	let rosterError = $state('');
	let rosterBusy = $state(false);

	/** Team players not yet registered on this season's roster. */
	const availablePlayers = $derived(
		data.teamPlayers.filter(
			(player) => !data.roster.some((rostered) => rostered.slug === player.slug)
		)
	);

	/** One-line summary of an event's dates and location, omitting blanks. */
	function eventMeta(event: Event): string {
		let dates = event.startDate;
		if (event.endDate !== '' && event.endDate !== event.startDate) {
			dates = dates === '' ? event.endDate : `${dates} – ${event.endDate}`;
		}
		return [dates, event.location].filter((part) => part !== '').join(' · ');
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
				location: eventLocation === '' ? undefined : eventLocation
			});
			eventName = '';
			eventStart = '';
			eventEnd = '';
			eventLocation = '';
			await refreshAll();
		} catch (error) {
			const violations = fieldErrorsOf(error);
			if (Object.keys(violations).length > 0) {
				eventFieldErrors = violations;
			} else {
				eventError = 'Could not create the event.';
			}
		} finally {
			creatingEvent = false;
		}
	}

	async function handleSave(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		saveError = '';
		fieldErrors = {};
		saving = true;
		try {
			await seasonClient.updateSeason({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				name: editName,
				startDate: editStart === '' ? undefined : editStart,
				endDate: editEnd === '' ? undefined : editEnd
			});
			await refreshAll();
		} catch (error) {
			const violations = fieldErrorsOf(error);
			if (Object.keys(violations).length > 0) {
				fieldErrors = violations;
			} else {
				saveError = 'Could not save the season.';
			}
		} finally {
			saving = false;
		}
	}

	async function handleAddToRoster(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		if (selectedPlayerSlug === '') {
			return;
		}
		rosterError = '';
		rosterBusy = true;
		try {
			await seasonClient.addSeasonRosterEntry({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				playerSlug: selectedPlayerSlug
			});
			selectedPlayerSlug = '';
			await refreshAll();
		} catch {
			rosterError = 'Could not add the player to the season roster.';
		} finally {
			rosterBusy = false;
		}
	}

	async function handleRemoveFromRoster(playerSlug: string): Promise<void> {
		rosterError = '';
		rosterBusy = true;
		try {
			await seasonClient.removeSeasonRosterEntry({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				playerSlug
			});
			await refreshAll();
		} catch {
			rosterError = 'Could not remove the player from the season roster.';
		} finally {
			rosterBusy = false;
		}
	}

	async function handleDelete(): Promise<void> {
		if (
			!confirm(
				`Delete ${data.season?.name ?? 'this season'}? This permanently deletes its events and matches.`
			)
		) {
			return;
		}
		deleteError = '';
		deleting = true;
		try {
			await seasonClient.deleteSeason({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug
			});
			await goto(resolve('/teams/[team_slug]', { team_slug: data.teamSlug }));
		} catch {
			deleteError = 'Could not delete the season.';
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>{data.season?.name ?? 'Season'} · My Bad, Coach</title>
</svelte:head>

<main class="page">
	<a class="back-link" href={resolve('/teams/[team_slug]', { team_slug: data.teamSlug })}>
		← {data.teamName || 'Team'}
	</a>

	{#if data.season === undefined}
		<div class="card">
			<h1>Season not found</h1>
			<p class="muted">This season does not exist or you are not a member of its team.</p>
		</div>
	{:else}
		<div class="page-head">
			<h1>{data.season.name}</h1>
			<span class="badge" class:badge-owner={isOwner}>{isOwner ? 'Owner' : 'Member'}</span>
		</div>

		<section class="card section-card">
			<h2>Season details</h2>
			<form onsubmit={handleSave}>
				<div class="field">
					<label for="edit-season-name">Season name</label>
					<input id="edit-season-name" type="text" required bind:value={editName} />
					{#if fieldErrors.name}
						<p class="field-error">{fieldErrors.name}</p>
					{/if}
				</div>
				<div class="date-range">
					<div class="field">
						<label for="edit-season-start">Start date</label>
						<input id="edit-season-start" type="date" bind:value={editStart} />
					</div>
					<div class="field">
						<label for="edit-season-end">End date</label>
						<input id="edit-season-end" type="date" bind:value={editEnd} />
					</div>
				</div>
				{#if saveError !== ''}
					<p class="form-error" role="alert">{saveError}</p>
				{/if}
				<button class="btn btn-primary" type="submit" disabled={saving}>
					{saving ? 'Saving…' : 'Save changes'}
				</button>
			</form>
			{#if isOwner}
				<div class="danger-zone">
					{#if deleteError !== ''}
						<p class="form-error" role="alert">{deleteError}</p>
					{/if}
					<button class="btn btn-danger" type="button" disabled={deleting} onclick={handleDelete}>
						{deleting ? 'Deleting…' : 'Delete season'}
					</button>
				</div>
			{/if}
		</section>

		<section class="card section-card">
			<h2>Events</h2>
			{#if data.events.length === 0}
				<p class="muted">No events yet.</p>
			{:else}
				<ul class="item-list" aria-label="Events">
					{#each data.events as event (event.slug)}
						<li class="item-row">
							<a
								href={resolve(
									'/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]',
									{
										team_slug: data.teamSlug,
										season_slug: data.seasonSlug,
										event_slug: event.slug
									}
								)}
							>
								{event.name}
							</a>
							{#if eventMeta(event) !== ''}
								<span class="muted season-dates">{eventMeta(event)}</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			<h3>Create event</h3>
			<form onsubmit={handleCreateEvent}>
				<div class="field">
					<label for="create-event-name">Event name</label>
					<input id="create-event-name" type="text" required bind:value={eventName} />
					{#if eventFieldErrors.name}
						<p class="field-error">{eventFieldErrors.name}</p>
					{/if}
				</div>
				<div class="date-range">
					<div class="field">
						<label for="create-event-start">Event start date</label>
						<input id="create-event-start" type="date" bind:value={eventStart} />
					</div>
					<div class="field">
						<label for="create-event-end">Event end date</label>
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
				<button class="btn btn-primary" type="submit" disabled={creatingEvent}>
					{creatingEvent ? 'Creating event…' : 'Create event'}
				</button>
			</form>
		</section>

		<section class="section-card">
			<h2>Season roster</h2>
			<p class="muted">
				{data.rosterTotal}
				{data.rosterTotal === 1 ? 'player' : 'players'} on this season's roster.
			</p>
			{#if data.roster.length === 0}
				<p class="muted">No players on this season's roster yet.</p>
			{:else}
				<ul class="item-list" aria-label="Season roster">
					{#each data.roster as player (player.slug)}
						<li class="item-row">
							<span>{player.names[0] ?? player.slug}</span>
							<button
								class="btn btn-secondary roster-remove"
								type="button"
								disabled={rosterBusy}
								onclick={() => {
									void handleRemoveFromRoster(player.slug);
								}}
							>
								Remove
							</button>
						</li>
					{/each}
				</ul>
			{/if}
			{#if rosterError !== ''}
				<p class="form-error" role="alert">{rosterError}</p>
			{/if}
			{#if availablePlayers.length === 0}
				<p class="muted">
					{#if data.teamPlayers.length === 0}
						No team players yet. Add players from the team's Players screen.
					{:else}
						Every team player is already on this season's roster.
					{/if}
				</p>
			{:else}
				<form class="roster-form" onsubmit={handleAddToRoster}>
					<div class="field">
						<label for="add-season-roster-player">Player to add</label>
						<select id="add-season-roster-player" bind:value={selectedPlayerSlug}>
							<option value="">Select a player</option>
							{#each availablePlayers as player (player.slug)}
								<option value={player.slug}>{player.names[0] ?? player.slug}</option>
							{/each}
						</select>
					</div>
					<button
						class="btn btn-primary"
						type="submit"
						disabled={rosterBusy || selectedPlayerSlug === ''}
					>
						Add to season roster
					</button>
				</form>
			{/if}
		</section>
	{/if}
</main>
