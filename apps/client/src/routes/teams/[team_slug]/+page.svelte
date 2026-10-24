<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { seasons as seasonClient, teams } from '#lib/api';
	import { fieldErrorsOf } from '#lib/api/errors';
	import type { TeamDetailData } from '#lib/structure-data';

	let { data }: { data: TeamDetailData } = $props();

	const isOwner = $derived(data.role === 'owner');

	let renameName = $derived(data.teamName);
	let renameError = $state('');
	let renaming = $state(false);

	let seasonName = $state('');
	let startDate = $state('');
	let endDate = $state('');
	let seasonFieldErrors = $state<Record<string, string>>({});
	let seasonError = $state('');
	let creatingSeason = $state(false);

	let deleting = $state(false);
	let deleteError = $state('');

	async function handleRename(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		renameError = '';
		renaming = true;
		try {
			await teams.updateTeam({ teamSlug: data.teamSlug, name: renameName });
			await refreshAll();
		} catch {
			renameError = 'Could not rename the team.';
		} finally {
			renaming = false;
		}
	}

	async function handleDelete(): Promise<void> {
		if (
			!confirm(
				`Delete ${data.teamName}? This permanently deletes the team and all of its seasons.`
			)
		) {
			return;
		}
		deleteError = '';
		deleting = true;
		try {
			await teams.deleteTeam({ teamSlug: data.teamSlug });
			await goto(resolve('/teams'));
		} catch {
			deleteError = 'Could not delete the team.';
		} finally {
			deleting = false;
		}
	}

	async function handleCreateSeason(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		seasonError = '';
		seasonFieldErrors = {};
		creatingSeason = true;
		try {
			await seasonClient.createSeason({
				teamSlug: data.teamSlug,
				name: seasonName,
				startDate: startDate === '' ? undefined : startDate,
				endDate: endDate === '' ? undefined : endDate
			});
			seasonName = '';
			startDate = '';
			endDate = '';
			await refreshAll();
		} catch (error) {
			const violations = fieldErrorsOf(error);
			if (Object.keys(violations).length > 0) {
				seasonFieldErrors = violations;
			} else {
				seasonError = 'Could not create the season.';
			}
		} finally {
			creatingSeason = false;
		}
	}
</script>

<svelte:head>
	<title>{data.teamName || 'Team'} · My Bad, Coach</title>
</svelte:head>

<main class="page">
	<a class="back-link" href={resolve('/teams')}>← Your teams</a>

	{#if data.teamName === ''}
		<div class="card">
			<h1>Team not found</h1>
			<p class="muted">This team does not exist or you are not a member of it.</p>
		</div>
	{:else}
		<div class="page-head">
			<h1>{data.teamName}</h1>
			<div class="btn-row">
				<a
					class="btn btn-secondary"
					href={resolve('/teams/[team_slug]/players', { team_slug: data.teamSlug })}
				>
					Players
				</a>
				<span class="badge" class:badge-owner={isOwner}>{isOwner ? 'Owner' : 'Member'}</span>
			</div>
		</div>

		<section class="section-card">
			<h2>Seasons</h2>
			{#if data.seasons.length === 0}
				<p class="muted">No seasons yet.</p>
			{:else}
				<ul class="item-list" aria-label="Seasons">
					{#each data.seasons as season (season.slug)}
						<li class="item-row">
							<a
								href={resolve('/teams/[team_slug]/seasons/[season_slug]', {
									team_slug: data.teamSlug,
									season_slug: season.slug
								})}
							>
								{season.name}
							</a>
							{#if season.startDate !== '' || season.endDate !== ''}
								<span class="muted season-dates">
									{season.startDate}{season.endDate === ''
										? ''
										: ` – ${season.endDate}`}
								</span>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}

			<h3>Create season</h3>
			<form onsubmit={handleCreateSeason}>
				<div class="field">
					<label for="create-season-name">Season name</label>
					<input id="create-season-name" type="text" required bind:value={seasonName} />
					{#if seasonFieldErrors.name}
						<p class="field-error">{seasonFieldErrors.name}</p>
					{/if}
				</div>
				<div class="date-range">
					<div class="field">
						<label for="create-season-start">Start date</label>
						<input id="create-season-start" type="date" bind:value={startDate} />
					</div>
					<div class="field">
						<label for="create-season-end">End date</label>
						<input id="create-season-end" type="date" bind:value={endDate} />
					</div>
				</div>
				{#if seasonError !== ''}
					<p class="form-error" role="alert">{seasonError}</p>
				{/if}
				<button class="btn btn-primary" type="submit" disabled={creatingSeason}>
					{creatingSeason ? 'Creating season…' : 'Create season'}
				</button>
			</form>
		</section>

		{#if isOwner}
			<section class="card section-card">
				<h2>Team settings</h2>
				<form onsubmit={handleRename}>
					<div class="field">
						<label for="rename-team-name">Team name</label>
						<input id="rename-team-name" type="text" required bind:value={renameName} />
					</div>
					{#if renameError !== ''}
						<p class="form-error" role="alert">{renameError}</p>
					{/if}
					<button class="btn btn-secondary" type="submit" disabled={renaming}>
						{renaming ? 'Renaming…' : 'Rename team'}
					</button>
				</form>
				<div class="danger-zone">
					{#if deleteError !== ''}
						<p class="form-error" role="alert">{deleteError}</p>
					{/if}
					<button class="btn btn-danger" type="button" disabled={deleting} onclick={handleDelete}>
						{deleting ? 'Deleting…' : 'Delete team'}
					</button>
				</div>
			</section>
		{/if}
	{/if}
</main>
