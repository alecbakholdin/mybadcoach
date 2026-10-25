<script lang="ts">
	import type { TeamListData } from '#lib/structure-data';

	let { data }: { data: TeamListData } = $props();
</script>

<svelte:head>
	<title>Your teams · My Bad, Coach</title>
</svelte:head>

<main class="structure-page">
	<div class="structure-header">
		<div class="title-row">
			<h1>Your teams</h1>
		</div>
	</div>

	{#if data.memberships.length === 0}
		<p class="muted">
			You don't belong to any teams yet. Use “New team” in the sidebar to create one.
		</p>
	{:else}
		<p class="muted">Select a team from the sidebar to see its seasons and settings.</p>
	{/if}
</main>
