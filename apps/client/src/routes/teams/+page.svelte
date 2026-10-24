<script lang="ts">
	// Minimal protected placeholder (ticket 05). The real team list and CRUD are
	// ticket 06; this route proves the session gate and the header logout action.
	import { session } from '#lib/session.svelte';
</script>

<svelte:head>
	<title>Your teams · My Bad, Coach</title>
</svelte:head>

<main class="screen">
	<div class="card">
		<h1>Your teams</h1>
		{#if session.currentUser}
			<p class="muted">
				Signed in as {session.currentUser.name} ({session.currentUser.email}).
			</p>
		{/if}
		<p>Team management lands next. The header's Log out action ends this session.</p>
	</div>
</main>
