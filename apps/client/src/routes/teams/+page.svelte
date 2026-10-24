<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { teams } from '#lib/api';
	import { fieldErrorsOf } from '#lib/api/errors';
	import type { TeamListData } from '#lib/structure-data';

	let { data }: { data: TeamListData } = $props();

	let name = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let errorMessage = $state('');
	let submitting = $state(false);

	async function handleCreate(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		errorMessage = '';
		fieldErrors = {};
		submitting = true;
		try {
			const response = await teams.createTeam({ name });
			if (response.team === undefined) {
				errorMessage = 'Could not create the team. Please try again.';
				return;
			}
			await goto(resolve('/teams/[team_slug]', { team_slug: response.team.slug }));
		} catch (error) {
			const violations = fieldErrorsOf(error);
			if (Object.keys(violations).length > 0) {
				fieldErrors = violations;
			} else {
				errorMessage = 'Could not create the team. Please try again.';
			}
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Your teams · My Bad, Coach</title>
</svelte:head>

<main class="page">
	<div class="page-head">
		<h1>Your teams</h1>
	</div>

	{#if data.memberships.length === 0}
		<p class="muted">You don't belong to any teams yet.</p>
	{:else}
		<ul class="item-list" aria-label="Your teams">
			{#each data.memberships as membership (membership.team?.slug ?? membership.role)}
				{#if membership.team}
					<li class="item-row">
						<a href={resolve('/teams/[team_slug]', { team_slug: membership.team.slug })}>
							{membership.team.name}
						</a>
						<span class="badge" class:badge-owner={membership.role === 'owner'}>
							{membership.role === 'owner' ? 'Owner' : 'Member'}
						</span>
					</li>
				{/if}
			{/each}
		</ul>
	{/if}

	<section class="card section-card">
		<h2>Create team</h2>
		<form onsubmit={handleCreate}>
			<div class="field">
				<label for="create-team-name">Team name</label>
				<input id="create-team-name" type="text" required bind:value={name} />
				{#if fieldErrors.name}
					<p class="field-error">{fieldErrors.name}</p>
				{/if}
			</div>
			{#if errorMessage !== ''}
				<p class="form-error" role="alert">{errorMessage}</p>
			{/if}
			<button class="btn btn-primary" type="submit" disabled={submitting}>
				{submitting ? 'Creating team…' : 'Create team'}
			</button>
		</form>
	</section>
</main>
