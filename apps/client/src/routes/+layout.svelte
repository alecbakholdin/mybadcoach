<script lang="ts">
	import '../app.css';
	import { theme } from '#lib/theme.svelte';

	let { children } = $props();

	const toggleLabel = $derived(
		theme.current === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'
	);
</script>

<svelte:head>
	<title>My Bad, Coach</title>
</svelte:head>

<header class="app-mast">
	<button class="theme-toggle" type="button" aria-label={toggleLabel} onclick={theme.toggle}>
		{theme.current === 'dark' ? 'Light' : 'Dark'}
	</button>
</header>

{@render children()}

<style>
	.app-mast {
		display: flex;
		justify-content: flex-end;
		padding: 16px 20px 0;
	}
</style>
