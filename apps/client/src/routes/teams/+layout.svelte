<script lang="ts">
	import type { Snippet } from 'svelte';

	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { seasons as seasonClient, teams } from '#lib/api';
	import { dateRange } from '#lib/dates';
	import { formErrorOf } from '#lib/forms';
	import Icon from '#lib/Icon.svelte';
	import RoleBadge from '#lib/RoleBadge.svelte';
	import { teamRoleOf, type TeamsShellData } from '#lib/structure-data';

	let { data, children }: { data: TeamsShellData; children: Snippet } = $props();

	// Off-canvas drawer state (mobile only; on desktop the rail is always open).
	let railOpen = $state(false);

	// Compact create forms, expanded from the rail's "+" affordances.
	let showCreateTeam = $state(false);
	let newTeamName = $state('');
	let teamFieldErrors = $state<Record<string, string>>({});
	let teamError = $state('');
	let creatingTeam = $state(false);

	let showCreateSeason = $state(false);
	let newSeasonName = $state('');
	let newSeasonStart = $state('');
	let newSeasonEnd = $state('');
	let seasonFieldErrors = $state<Record<string, string>>({});
	let seasonError = $state('');
	let creatingSeason = $state(false);

	const isActiveTeam = (slug: string): boolean => slug === data.activeTeamSlug;
	const isActiveSeason = (slug: string): boolean => slug === data.activeSeasonSlug;

	const activeTeamName = $derived(
		data.memberships.find((entry) => entry.team?.slug === data.activeTeamSlug)?.team?.name ?? '',
	);
	const mobileTitle = $derived(activeTeamName === '' ? 'Teams' : activeTeamName);

	function openRail(): void {
		railOpen = true;
	}

	function closeRail(): void {
		railOpen = false;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && railOpen) {
			closeRail();
		}
	}

	function cancelCreateTeam(): void {
		showCreateTeam = false;
		newTeamName = '';
		teamFieldErrors = {};
		teamError = '';
	}

	function cancelCreateSeason(): void {
		showCreateSeason = false;
		newSeasonName = '';
		newSeasonStart = '';
		newSeasonEnd = '';
		seasonFieldErrors = {};
		seasonError = '';
	}

	async function handleCreateTeam(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		teamError = '';
		teamFieldErrors = {};
		creatingTeam = true;
		try {
			const response = await teams.createTeam({ name: newTeamName });
			if (response.team === undefined) {
				teamError = 'Could not create the team. Please try again.';
				return;
			}
			newTeamName = '';
			showCreateTeam = false;
			await goto(resolve('/teams/[team_slug]', { team_slug: response.team.slug }));
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not create the team. Please try again.');
			teamFieldErrors = outcome.fieldErrors;
			teamError = outcome.message;
		} finally {
			creatingTeam = false;
		}
	}

	async function handleCreateSeason(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (data.activeTeamSlug === undefined) {
			return;
		}
		seasonError = '';
		seasonFieldErrors = {};
		creatingSeason = true;
		try {
			const response = await seasonClient.createSeason({
				teamSlug: data.activeTeamSlug,
				name: newSeasonName,
				startDate: newSeasonStart === '' ? undefined : newSeasonStart,
				endDate: newSeasonEnd === '' ? undefined : newSeasonEnd,
			});
			newSeasonName = '';
			newSeasonStart = '';
			newSeasonEnd = '';
			showCreateSeason = false;
			if (response.season !== undefined) {
				await goto(
					resolve('/teams/[team_slug]/seasons/[season_slug]', {
						team_slug: data.activeTeamSlug,
						season_slug: response.season.slug,
					}),
				);
			} else {
				await refreshAll();
			}
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not create the season.');
			seasonFieldErrors = outcome.fieldErrors;
			seasonError = outcome.message;
		} finally {
			creatingSeason = false;
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="structure-shell" class:rail-open={railOpen}>
	<div class="scrim" aria-hidden="true" onclick={closeRail}></div>

	<aside class="rail" id="team-navigation" aria-label="Team navigation">
		<div class="rail-inner">
			<button
				class="icon-btn rail-close"
				type="button"
				aria-label="Close navigation"
				onclick={closeRail}
			>
				<Icon name="close" />
			</button>

			<nav class="rail-nav" aria-label="Your teams">
				<div class="rail-kicker">
					<span>Your teams</span>
					<span class="rail-count">{data.memberships.length}</span>
				</div>

				<div class="team-list">
					{#each data.memberships as membership (membership.team?.slug ?? membership.role)}
						{#if membership.team}
							<a
								class="team-row"
								class:is-active={isActiveTeam(membership.team.slug)}
								href={resolve('/teams/[team_slug]', { team_slug: membership.team.slug })}
								aria-current={isActiveTeam(membership.team.slug) ? 'true' : undefined}
								onclick={closeRail}
							>
								<Icon name="team" class="team-glyph" />
								<span class="team-name">{membership.team.name}</span>
								<RoleBadge role={teamRoleOf(membership.role)} />
							</a>

							{#if isActiveTeam(membership.team.slug)}
								<div class="seasons" role="group" aria-label={`${membership.team.name} seasons`}>
									{#if data.activeTeamSeasons.length > 0}
										<div class="seasons-kicker">Seasons</div>
										{#each data.activeTeamSeasons as season (season.slug)}
											<a
												class="season-row"
												class:is-active={isActiveSeason(season.slug)}
												href={resolve('/teams/[team_slug]/seasons/[season_slug]', {
													team_slug: membership.team.slug,
													season_slug: season.slug,
												})}
												aria-current={isActiveSeason(season.slug) ? 'page' : undefined}
												onclick={closeRail}
											>
												<Icon name="calendar" class="season-glyph" />
												<span class="season-text">
													<span class="season-name">{season.name}</span>
													{#if dateRange(season.startDate, season.endDate) !== ''}
														<span class="season-dates">
															{dateRange(season.startDate, season.endDate)}
														</span>
													{/if}
												</span>
											</a>
										{/each}
									{/if}

									{#if showCreateSeason}
										<form class="rail-create" onsubmit={handleCreateSeason}>
											<label class="rail-create-label" for="new-season-name">Season name</label>
											<input id="new-season-name" type="text" required bind:value={newSeasonName} />
											{#if seasonFieldErrors.name}
												<p class="field-error">{seasonFieldErrors.name}</p>
											{/if}
											<div class="rail-create-dates">
												<input
													id="new-season-start"
													type="date"
													aria-label="Start date"
													bind:value={newSeasonStart}
												/>
												<input
													id="new-season-end"
													type="date"
													aria-label="End date"
													bind:value={newSeasonEnd}
												/>
											</div>
											{#if seasonError !== ''}
												<p class="form-error" role="alert">{seasonError}</p>
											{/if}
											<div class="rail-create-actions">
												<button
													class="btn btn-primary btn-compact"
													type="submit"
													disabled={creatingSeason}
												>
													{creatingSeason ? 'Creating season…' : 'Create season'}
												</button>
												<button
													class="btn btn-tertiary btn-compact"
													type="button"
													onclick={cancelCreateSeason}
												>
													Cancel
												</button>
											</div>
										</form>
									{:else}
										<button
											class="rail-add season-add"
											type="button"
											onclick={() => {
												showCreateSeason = true;
											}}
										>
											<Icon name="plus" />
											New season
										</button>
									{/if}
								</div>
							{/if}
						{/if}
					{/each}
				</div>
			</nav>

			<div class="rail-foot">
				{#if showCreateTeam}
					<form class="rail-create" onsubmit={handleCreateTeam}>
						<label class="rail-create-label" for="new-team-name">Team name</label>
						<input id="new-team-name" type="text" required bind:value={newTeamName} />
						{#if teamFieldErrors.name}
							<p class="field-error">{teamFieldErrors.name}</p>
						{/if}
						{#if teamError !== ''}
							<p class="form-error" role="alert">{teamError}</p>
						{/if}
						<div class="rail-create-actions">
							<button class="btn btn-primary btn-compact" type="submit" disabled={creatingTeam}>
								{creatingTeam ? 'Creating team…' : 'Create team'}
							</button>
							<button class="btn btn-tertiary btn-compact" type="button" onclick={cancelCreateTeam}>
								Cancel
							</button>
						</div>
					</form>
				{:else}
					<button
						class="rail-add"
						type="button"
						onclick={() => {
							showCreateTeam = true;
						}}
					>
						<Icon name="plus" />
						New team
					</button>
				{/if}
			</div>
		</div>
	</aside>

	<div class="structure-main">
		<header class="mobile-bar">
			<button
				class="icon-btn rail-toggle"
				type="button"
				aria-label="Open navigation"
				aria-controls="team-navigation"
				aria-expanded={railOpen}
				onclick={openRail}
			>
				<Icon name="menu" />
			</button>
			<span class="mobile-brand">{mobileTitle}</span>
		</header>

		<div class="main-scroll">
			{@render children()}
		</div>
	</div>
</div>
