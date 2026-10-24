<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { players as playerClient } from '#lib/api';
	import { formErrorOf } from '#lib/forms';
	import { aliases, displayName } from '#lib/players';
	import RoleBadge from '#lib/RoleBadge.svelte';
	import type { PlayerListData } from '#lib/structure-data';
	import type { Player } from '$shared/gen/mybadcoach/players/v1/players_pb';

	let { data }: { data: PlayerListData } = $props();

	const isOwner = $derived(data.role === 'owner');

	let newDisplayName = $state('');
	let newAliases = $state('');
	let createFieldErrors = $state<Record<string, string>>({});
	let createError = $state('');
	let creating = $state(false);

	let editingSlug = $state<string | null>(null);
	let editDisplayName = $state('');
	let editAliases = $state('');
	let editFieldErrors = $state<Record<string, string>>({});
	let editError = $state('');
	let savingEdit = $state(false);

	let deletingSlug = $state<string | null>(null);
	let deleteError = $state('');

	/** Display name first, then trimmed non-empty aliases. */
	function buildNames(display: string, aliasList: string): string[] {
		const trimmedAliases = aliasList
			.split(',')
			.map((alias) => alias.trim())
			.filter((alias) => alias !== '');
		return [display.trim(), ...trimmedAliases];
	}

	async function handleCreate(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		createError = '';
		createFieldErrors = {};
		if (newDisplayName.trim() === '') {
			createFieldErrors = { names: 'Enter a display name.' };
			return;
		}
		creating = true;
		try {
			await playerClient.createPlayer({
				teamSlug: data.teamSlug,
				names: buildNames(newDisplayName, newAliases)
			});
			newDisplayName = '';
			newAliases = '';
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not create the player.');
			createFieldErrors = outcome.fieldErrors;
			createError = outcome.message;
		} finally {
			creating = false;
		}
	}

	function startEdit(player: Player): void {
		editingSlug = player.slug;
		editDisplayName = displayName(player);
		editAliases = aliases(player);
		editFieldErrors = {};
		editError = '';
	}

	function cancelEdit(): void {
		editingSlug = null;
		editFieldErrors = {};
		editError = '';
	}

	async function handleSaveEdit(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		if (editingSlug === null) {
			return;
		}
		editError = '';
		editFieldErrors = {};
		if (editDisplayName.trim() === '') {
			editFieldErrors = { names: 'Enter a display name.' };
			return;
		}
		savingEdit = true;
		try {
			// `names` replaces the whole list; the slug never changes.
			await playerClient.updatePlayer({
				teamSlug: data.teamSlug,
				playerSlug: editingSlug,
				names: buildNames(editDisplayName, editAliases)
			});
			editingSlug = null;
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not save the player.');
			editFieldErrors = outcome.fieldErrors;
			editError = outcome.message;
		} finally {
			savingEdit = false;
		}
	}

	async function handleDelete(player: Player): Promise<void> {
		if (
			!confirm(
				`Delete ${displayName(player)}? This removes them from every season and event roster.`
			)
		) {
			return;
		}
		deleteError = '';
		deletingSlug = player.slug;
		try {
			await playerClient.deletePlayer({
				teamSlug: data.teamSlug,
				playerSlug: player.slug
			});
			await refreshAll();
		} catch {
			deleteError = 'Could not delete the player.';
		} finally {
			deletingSlug = null;
		}
	}
</script>

<svelte:head>
	<title>Players · {data.teamName || 'Team'} · My Bad, Coach</title>
</svelte:head>

<main class="page">
	<a class="back-link" href={resolve('/teams/[team_slug]', { team_slug: data.teamSlug })}>
		← {data.teamName || 'Team'}
	</a>

	{#if data.teamName === ''}
		<div class="card">
			<h1>Team not found</h1>
			<p class="muted">This team does not exist or you are not a member of it.</p>
		</div>
	{:else}
		<div class="page-head">
			<h1>Players</h1>
			<RoleBadge role={data.role} />
		</div>

		<section class="section-card">
			<h2>Team players</h2>
			{#if data.players.length === 0}
				<p class="muted">No players yet.</p>
			{:else}
				<ul class="item-list" aria-label="Team players">
					{#each data.players as player (player.slug)}
						<li class="item-row player-row">
							{#if editingSlug === player.slug}
								<form class="player-edit" onsubmit={handleSaveEdit}>
									<div class="field">
										<label for="edit-player-display-name">Display name</label>
										<input
											id="edit-player-display-name"
											type="text"
											required
											bind:value={editDisplayName}
										/>
										{#if editFieldErrors.names}
											<p class="field-error">{editFieldErrors.names}</p>
										{/if}
									</div>
									<div class="field">
										<label for="edit-player-aliases">Aliases (comma-separated)</label>
										<input
											id="edit-player-aliases"
											type="text"
											bind:value={editAliases}
										/>
									</div>
									{#if editError !== ''}
										<p class="form-error" role="alert">{editError}</p>
									{/if}
									<div class="btn-row">
										<button class="btn btn-primary" type="submit" disabled={savingEdit}>
											{savingEdit ? 'Saving…' : 'Save player'}
										</button>
										<button class="btn btn-secondary" type="button" onclick={cancelEdit}>
											Cancel
										</button>
									</div>
								</form>
							{:else}
								<div class="player-names">
									<span class="player-name">{displayName(player)}</span>
									{#if aliases(player) !== ''}
										<span class="muted player-aliases">{aliases(player)}</span>
									{/if}
								</div>
								<div class="btn-row">
									<button
										class="btn btn-secondary"
										type="button"
										onclick={() => {
											startEdit(player);
										}}
									>
										Edit
									</button>
									{#if isOwner}
										<button
											class="btn btn-danger"
											type="button"
											disabled={deletingSlug === player.slug}
											onclick={() => {
												void handleDelete(player);
											}}
										>
											{deletingSlug === player.slug ? 'Deleting…' : 'Delete'}
										</button>
									{/if}
								</div>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
			{#if deleteError !== ''}
				<p class="form-error" role="alert">{deleteError}</p>
			{/if}
		</section>

		<section class="card section-card">
			<h2>Add player</h2>
			<form onsubmit={handleCreate}>
				<div class="field">
					<label for="create-player-display-name">Display name</label>
					<input id="create-player-display-name" type="text" required bind:value={newDisplayName} />
					{#if createFieldErrors.names}
						<p class="field-error">{createFieldErrors.names}</p>
					{/if}
				</div>
				<div class="field">
					<label for="create-player-aliases">Aliases (comma-separated)</label>
					<input id="create-player-aliases" type="text" bind:value={newAliases} />
				</div>
				{#if createError !== ''}
					<p class="form-error" role="alert">{createError}</p>
				{/if}
				<button class="btn btn-primary" type="submit" disabled={creating}>
					{creating ? 'Adding player…' : 'Add player'}
				</button>
			</form>
		</section>
	{/if}
</main>
