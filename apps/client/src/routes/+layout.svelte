<script lang="ts">
	import type { Snippet } from 'svelte';

	import { page } from '$app/state';

	import AccountActions from '#lib/AccountActions.svelte';
	import Brand from '#lib/Brand.svelte';

	import '../app.css';

	let { children }: { children: Snippet } = $props();

	// The teams shell owns its own chrome (wordmark in the rail, account controls
	// in the rail footer), so the top mast only appears outside /teams.
	const onTeams = $derived(page.url.pathname.startsWith('/teams'));
</script>

<svelte:head>
	<title>My Bad, Coach</title>
</svelte:head>

<div class="app-shell">
	{#if !onTeams}
		<header class="app-mast">
			<Brand />
			<AccountActions variant="mast" />
		</header>
	{/if}

	<div class="app-body">
		{@render children()}
	</div>
</div>
