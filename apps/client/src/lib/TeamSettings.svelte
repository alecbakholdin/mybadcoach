<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { teams } from '#lib/api';
	import Icon from '#lib/Icon.svelte';
	import type { TeamRole } from '#lib/structure-data';

	let { teamSlug, teamName, role }: { teamSlug: string; teamName: string; role: TeamRole } =
		$props();

	const isOwner = $derived(role === 'owner');

	let renameName = $derived(teamName);
	let renameError = $state('');
	let renaming = $state(false);
	let deleting = $state(false);
	let deleteError = $state('');

	async function handleRename(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		renameError = '';
		renaming = true;
		try {
			await teams.updateTeam({ teamSlug, name: renameName });
			await refreshAll();
		} catch {
			renameError = 'Could not rename the team.';
		} finally {
			renaming = false;
		}
	}

	async function handleDelete(): Promise<void> {
		if (!confirm(`Delete ${teamName}? This permanently deletes the team and all of its seasons.`)) {
			return;
		}
		deleteError = '';
		deleting = true;
		try {
			await teams.deleteTeam({ teamSlug });
			await goto(resolve('/teams'));
		} catch {
			deleteError = 'Could not delete the team.';
		} finally {
			deleting = false;
		}
	}
</script>

<section class="settings-group" aria-labelledby="team-settings-heading">
	<div class="settings-group-head">
		<h2 class="settings-group-title" id="team-settings-heading">
			<Icon name="team" />
			Team settings
		</h2>
		<p class="settings-group-note">Shared by every season on this team.</p>
	</div>

	{#if isOwner}
		<form class="settings-form" onsubmit={handleRename}>
			<div class="field">
				<label for="rename-team-name">Team name</label>
				<input id="rename-team-name" type="text" required bind:value={renameName} />
			</div>
			{#if renameError !== ''}
				<p class="form-error" role="alert">{renameError}</p>
			{/if}
			<button class="btn btn-primary" type="submit" disabled={renaming}>
				<Icon name="pencil" />
				{renaming ? 'Renaming…' : 'Rename team'}
			</button>
		</form>

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
				{deleting ? 'Deleting…' : 'Delete team'}
			</button>
		</div>
	{:else}
		<p class="muted">Only the team owner can rename or delete this team.</p>
	{/if}
</section>
