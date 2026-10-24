<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { seasons as seasonClient } from '#lib/api';
	import { formErrorOf } from '#lib/forms';
	import Icon from '#lib/Icon.svelte';
	import type { TeamRole } from '#lib/structure-data';

	let {
		teamSlug,
		seasonSlug,
		name,
		startDate,
		endDate,
		role,
	}: {
		teamSlug: string;
		seasonSlug: string;
		name: string;
		startDate: string;
		endDate: string;
		role: TeamRole;
	} = $props();

	const isOwner = $derived(role === 'owner');

	let editName = $derived(name);
	let editStart = $derived(startDate);
	let editEnd = $derived(endDate);
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
				teamSlug,
				seasonSlug,
				name: editName,
				// An empty string is present-on-the-wire and clears the column;
				// `undefined` would mean "leave unchanged".
				startDate: editStart,
				endDate: editEnd,
			});
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not save the season.');
			fieldErrors = outcome.fieldErrors;
			saveError = outcome.message;
		} finally {
			saving = false;
		}
	}

	async function handleDelete(): Promise<void> {
		if (!confirm(`Delete ${name}? This permanently deletes its events and matches.`)) {
			return;
		}
		deleteError = '';
		deleting = true;
		try {
			await seasonClient.deleteSeason({ teamSlug, seasonSlug });
			await goto(resolve('/teams/[team_slug]', { team_slug: teamSlug }));
		} catch {
			deleteError = 'Could not delete the season.';
		} finally {
			deleting = false;
		}
	}
</script>

<section class="settings-group" aria-labelledby="season-settings-heading">
	<div class="settings-group-head">
		<h2 class="settings-group-title" id="season-settings-heading">
			<Icon name="calendar" />
			Season settings
		</h2>
		<p class="settings-group-note">Only affects {name}.</p>
	</div>

	<form class="settings-form" onsubmit={handleSave}>
		<div class="form-grid">
			<div class="field field-full">
				<label for="edit-season-name">Season name</label>
				<input id="edit-season-name" type="text" required bind:value={editName} />
				{#if fieldErrors.name}
					<p class="field-error">{fieldErrors.name}</p>
				{/if}
			</div>
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
			<span class="danger-label">
				<Icon name="trash" />
				Danger zone
			</span>
			{#if deleteError !== ''}
				<p class="form-error" role="alert">{deleteError}</p>
			{/if}
			<button class="btn btn-danger" type="button" disabled={deleting} onclick={handleDelete}>
				<Icon name="trash" />
				{deleting ? 'Deleting…' : 'Delete season'}
			</button>
		</div>
	{/if}
</section>
