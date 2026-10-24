<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { seasons as seasonClient } from '#lib/api';
	import { fieldErrorsOf } from '#lib/api/errors';
	import type { SeasonDetailData } from '#lib/structure-data';

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
						<li class="item-row">{player.names[0] ?? player.slug}</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</main>
