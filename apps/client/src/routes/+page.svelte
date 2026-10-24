<script lang="ts">
	// Public marketing/hero home. Login + Register are the only actions for now;
	// demo content hydrates here in a later ticket.
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>My Bad, Coach</title>
</svelte:head>

<main class="hero">
	<p class="eyebrow">Ultimate Frisbee, quantified</p>
	<h1>My Bad, Coach</h1>
	<p class="lede">
		Track every touch, turn, and goal — live, together, and even when the network drops.
	</p>
	<div class="btn-row hero-actions">
		<a class="btn btn-primary" href={resolve('/login')}>Log in</a>
		<a class="btn btn-secondary" href={resolve('/register')}>Register</a>
	</div>
</main>
