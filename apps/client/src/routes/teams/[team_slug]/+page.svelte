<script lang="ts">
	import { resolve } from '$app/paths';

	import Icon from '#lib/Icon.svelte';
	import RoleBadge from '#lib/RoleBadge.svelte';
	import type { TeamDetailData } from '#lib/structure-data';
	import TeamSettings from '#lib/TeamSettings.svelte';

	let { data }: { data: TeamDetailData } = $props();

	let tab = $state<'overview' | 'settings'>('overview');
</script>

<svelte:head>
	<title>{data.teamName || 'Team'} · My Bad, Coach</title>
</svelte:head>

<main class="structure-page">
	{#if data.teamName === ''}
		<div class="card">
			<h1>Team not found</h1>
			<p class="muted">This team does not exist or you are not a member of it.</p>
		</div>
	{:else}
		<div class="route" aria-label="Breadcrumb">
			<span class="route-seg">/teams</span>
			<span class="route-seg">/{data.teamSlug}</span>
		</div>

		<header class="structure-header">
			<div class="title-row">
				<h1>{data.teamName}</h1>
				<RoleBadge role={data.role} />
			</div>
			<div class="head-sub">
				<span class="summary">
					{data.seasons.length}
					{data.seasons.length === 1 ? 'season' : 'seasons'}
				</span>
			</div>
		</header>

		<div class="tabbed">
			<div class="tabs" role="tablist" aria-label="Team">
				<button
					class="tab"
					type="button"
					role="tab"
					aria-selected={tab === 'overview'}
					onclick={() => {
						tab = 'overview';
					}}
				>
					<Icon name="grid" />
					Overview
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

			{#if tab === 'overview'}
				<div class="panel" role="tabpanel" aria-label="Overview">
					<div class="overview-strip">
						<p class="overview-note">
							{data.seasons.length}
							{data.seasons.length === 1 ? 'season' : 'seasons'}
						</p>
						<a
							class="btn btn-secondary btn-compact"
							href={resolve('/teams/[team_slug]/players', { team_slug: data.teamSlug })}
						>
							<Icon name="team" />
							Players
						</a>
					</div>
				</div>
			{:else}
				<div class="panel" role="tabpanel" aria-label="Settings">
					<div class="settings-stack">
						<TeamSettings teamSlug={data.teamSlug} teamName={data.teamName} role={data.role} />
					</div>
				</div>
			{/if}
		</div>
	{/if}
</main>
