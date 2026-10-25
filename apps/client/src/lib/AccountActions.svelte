<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { session } from '#lib/session.svelte';
	import { theme } from '#lib/theme.svelte';

	let { variant = 'mast' }: { variant?: 'mast' | 'rail' } = $props();

	const toggleLabel = $derived(
		theme.current === 'dark' ? 'Switch to light theme' : 'Switch to dark theme',
	);

	async function handleLogout(): Promise<void> {
		await session.logout();
		await goto(resolve('/'));
	}
</script>

<div class={variant === 'rail' ? 'rail-account' : 'mast-actions'}>
	{#if session.currentUser}
		<span class="who">{session.currentUser.name}</span>
		<button class="btn btn-secondary" type="button" onclick={handleLogout}>Log out</button>
	{/if}
	<button class="theme-toggle" type="button" aria-label={toggleLabel} onclick={theme.toggle}>
		{theme.current === 'dark' ? 'Light' : 'Dark'}
	</button>
</div>
