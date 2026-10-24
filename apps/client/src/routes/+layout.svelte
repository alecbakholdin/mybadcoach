<script lang="ts">
	import type { Snippet } from 'svelte';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { session } from '#lib/session.svelte';
	import { theme } from '#lib/theme.svelte';

	import '../app.css';

	let { children }: { children: Snippet } = $props();

	const toggleLabel = $derived(
		theme.current === 'dark' ? 'Switch to light theme' : 'Switch to dark theme',
	);

	async function handleLogout(): Promise<void> {
		await session.logout();
		await goto(resolve('/'));
	}
</script>

<svelte:head>
	<title>My Bad, Coach</title>
</svelte:head>

<div class="app-shell">
	<header class="app-mast">
		<a class="brand" href={resolve('/')}>My Bad, Coach</a>
		<div class="mast-actions">
			{#if session.currentUser}
				<span class="who">{session.currentUser.name}</span>
				<button class="btn btn-secondary" type="button" onclick={handleLogout}>Log out</button>
			{/if}
			<button class="theme-toggle" type="button" aria-label={toggleLabel} onclick={theme.toggle}>
				{theme.current === 'dark' ? 'Light' : 'Dark'}
			</button>
		</div>
	</header>

	<div class="app-body">
		{@render children()}
	</div>
</div>
