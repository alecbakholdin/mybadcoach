<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { events as eventClient, matches as matchClient } from '#lib/api';
	import { formErrorOf } from '#lib/forms';
	import { displayName } from '#lib/players';
	import type { EventDetailData } from '#lib/structure-data';

	let { data }: { data: EventDetailData } = $props();

	let editName = $derived(data.event?.name ?? '');
	let editStart = $derived(data.event?.startDate ?? '');
	let editEnd = $derived(data.event?.endDate ?? '');
	let editLocation = $derived(data.event?.location ?? '');
	let fieldErrors = $state<Record<string, string>>({});
	let saveError = $state('');
	let saving = $state(false);

	let deleting = $state(false);
	let deleteError = $state('');

	let opponentName = $state('');
	let scheduledAt = $state('');
	let pointCap = $state('');
	let matchFieldErrors = $state<Record<string, string>>({});
	let matchError = $state('');
	let creatingMatch = $state(false);

	let selectedPlayerSlug = $state('');
	let rosterError = $state('');
	let rosterBusy = $state(false);

	/** Team players not yet registered on this event's roster. */
	const availablePlayers = $derived(
		data.teamPlayers.filter(
			(player) => !data.roster.some((rostered) => rostered.slug === player.slug),
		),
	);

	async function handleAddToRoster(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		if (selectedPlayerSlug === '') {
			return;
		}
		rosterError = '';
		rosterBusy = true;
		try {
			await eventClient.addEventRosterEntry({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
				playerSlug: selectedPlayerSlug,
			});
			selectedPlayerSlug = '';
			await refreshAll();
		} catch {
			rosterError = 'Could not add the player to the event roster.';
		} finally {
			rosterBusy = false;
		}
	}

	async function handleRemoveFromRoster(playerSlug: string): Promise<void> {
		rosterError = '';
		rosterBusy = true;
		try {
			await eventClient.removeEventRosterEntry({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
				playerSlug,
			});
			await refreshAll();
		} catch {
			rosterError = 'Could not remove the player from the event roster.';
		} finally {
			rosterBusy = false;
		}
	}

	/** Empty stays absent; the number input keeps the rest integral. */
	function parsedPointCap(value: string): bigint | undefined {
		const trimmed = value.trim();
		if (trimmed === '' || !/^\d+$/.test(trimmed)) {
			return undefined;
		}
		return BigInt(trimmed);
	}

	async function handleSave(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		saveError = '';
		fieldErrors = {};
		saving = true;
		try {
			await eventClient.updateEvent({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
				name: editName,
				startDate: editStart,
				endDate: editEnd,
				location: editLocation,
			});
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not save the event.');
			fieldErrors = outcome.fieldErrors;
			saveError = outcome.message;
		} finally {
			saving = false;
		}
	}

	async function handleDelete(): Promise<void> {
		if (
			!confirm(`Delete ${data.event?.name ?? 'this event'}? This permanently deletes its matches.`)
		) {
			return;
		}
		deleteError = '';
		deleting = true;
		try {
			await eventClient.deleteEvent({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
			});
			await goto(
				resolve('/teams/[team_slug]/seasons/[season_slug]', {
					team_slug: data.teamSlug,
					season_slug: data.seasonSlug,
				}),
			);
		} catch {
			deleteError = 'Could not delete the event.';
		} finally {
			deleting = false;
		}
	}

	async function handleCreateMatch(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		matchError = '';
		matchFieldErrors = {};
		creatingMatch = true;
		try {
			await matchClient.createMatch({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
				opponentName,
				scheduledAt: scheduledAt === '' ? undefined : scheduledAt,
				pointCap: parsedPointCap(pointCap),
			});
			opponentName = '';
			scheduledAt = '';
			pointCap = '';
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not create the match.');
			matchFieldErrors = outcome.fieldErrors;
			matchError = outcome.message;
		} finally {
			creatingMatch = false;
		}
	}
</script>

<svelte:head>
	<title>{data.event?.name ?? 'Event'} · My Bad, Coach</title>
</svelte:head>

<main class="page">
	<a
		class="back-link"
		href={resolve('/teams/[team_slug]/seasons/[season_slug]', {
			team_slug: data.teamSlug,
			season_slug: data.seasonSlug,
		})}
	>
		← {data.seasonName || 'Season'}
	</a>

	{#if data.event === undefined}
		<div class="card">
			<h1>Event not found</h1>
			<p class="muted">This event does not exist or you are not a member of its team.</p>
		</div>
	{:else}
		<div class="page-head">
			<h1>{data.event.name}</h1>
		</div>

		<section class="card section-card">
			<h2>Event details</h2>
			<form onsubmit={handleSave}>
				<div class="field">
					<label for="edit-event-name">Event name</label>
					<input id="edit-event-name" type="text" required bind:value={editName} />
					{#if fieldErrors.name}
						<p class="field-error">{fieldErrors.name}</p>
					{/if}
				</div>
				<div class="date-range">
					<div class="field">
						<label for="edit-event-start">Event start date</label>
						<input id="edit-event-start" type="date" bind:value={editStart} />
					</div>
					<div class="field">
						<label for="edit-event-end">Event end date</label>
						<input id="edit-event-end" type="date" bind:value={editEnd} />
					</div>
				</div>
				<div class="field">
					<label for="edit-event-location">Location</label>
					<input id="edit-event-location" type="text" bind:value={editLocation} />
				</div>
				{#if saveError !== ''}
					<p class="form-error" role="alert">{saveError}</p>
				{/if}
				<button class="btn btn-primary" type="submit" disabled={saving}>
					{saving ? 'Saving…' : 'Save changes'}
				</button>
			</form>
			<div class="danger-zone">
				{#if deleteError !== ''}
					<p class="form-error" role="alert">{deleteError}</p>
				{/if}
				<button class="btn btn-danger" type="button" disabled={deleting} onclick={handleDelete}>
					{deleting ? 'Deleting…' : 'Delete event'}
				</button>
			</div>
		</section>

		<section class="section-card">
			<h2>Matches</h2>
			{#if data.matches.length === 0}
				<p class="muted">No matches yet.</p>
			{:else}
				<ul class="item-list" aria-label="Matches">
					{#each data.matches as match (match.slug)}
						<li class="item-row">
							<a
								href={resolve(
									'/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]/matches/[match_slug]',
									{
										team_slug: data.teamSlug,
										season_slug: data.seasonSlug,
										event_slug: data.eventSlug,
										match_slug: match.slug,
									},
								)}
							>
								{match.opponentName}
							</a>
							{#if match.scheduledAt !== ''}
								<span class="muted season-dates">{match.scheduledAt}</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			<h3>Create match</h3>
			<form onsubmit={handleCreateMatch}>
				<div class="field">
					<label for="create-match-opponent">Opponent name</label>
					<input id="create-match-opponent" type="text" required bind:value={opponentName} />
					{#if matchFieldErrors.opponent_name}
						<p class="field-error">{matchFieldErrors.opponent_name}</p>
					{/if}
				</div>
				<div class="date-range">
					<div class="field">
						<label for="create-match-scheduled">Scheduled at</label>
						<input id="create-match-scheduled" type="datetime-local" bind:value={scheduledAt} />
					</div>
					<div class="field">
						<label for="create-match-point-cap">Point cap</label>
						<input
							id="create-match-point-cap"
							type="number"
							min="0"
							step="1"
							bind:value={pointCap}
						/>
					</div>
				</div>
				{#if matchError !== ''}
					<p class="form-error" role="alert">{matchError}</p>
				{/if}
				<button class="btn btn-primary" type="submit" disabled={creatingMatch}>
					{creatingMatch ? 'Creating match…' : 'Create match'}
				</button>
			</form>
		</section>

		<section class="section-card">
			<h2>Event roster</h2>
			<p class="muted">
				{data.rosterTotal}
				{data.rosterTotal === 1 ? 'player' : 'players'} on this event's roster. The event roster is independent
				of the season roster.
			</p>
			{#if data.roster.length === 0}
				<p class="muted">No players on this event's roster yet.</p>
			{:else}
				<ul class="item-list" aria-label="Event roster">
					{#each data.roster as player (player.slug)}
						<li class="item-row">
							<span>{displayName(player)}</span>
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
						Every team player is already on this event's roster.
					{/if}
				</p>
			{:else}
				<form class="roster-form" onsubmit={handleAddToRoster}>
					<div class="field">
						<label for="add-event-roster-player">Player to add</label>
						<select id="add-event-roster-player" bind:value={selectedPlayerSlug}>
							<option value="">Select a player</option>
							{#each availablePlayers as player (player.slug)}
								<option value={player.slug}>{displayName(player)}</option>
							{/each}
						</select>
					</div>
					<button
						class="btn btn-primary"
						type="submit"
						disabled={rosterBusy || selectedPlayerSlug === ''}
					>
						Add to event roster
					</button>
				</form>
			{/if}
		</section>
	{/if}
</main>
