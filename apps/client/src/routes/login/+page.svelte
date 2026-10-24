<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { auth } from '#lib/api';
	import { isInvalidCredentials } from '#lib/api/errors';
	import { session } from '#lib/session.svelte';

	let email = $state('');
	let password = $state('');
	let errorMessage = $state('');
	let submitting = $state(false);

	const expired = $derived(page.url.searchParams.get('notice') === 'session-expired');

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		errorMessage = '';
		submitting = true;
		try {
			const response = await auth.login({ email, password });
			if (response.user !== undefined) {
				session.setUser(response.user);
			}
			await goto(resolve('/teams'));
		} catch (error) {
			errorMessage = isInvalidCredentials(error)
				? 'Invalid email or password.'
				: 'Could not log in. Please try again.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Log in · My Bad, Coach</title>
</svelte:head>

<main class="screen">
	<div class="card">
		<h1>Log in</h1>
		{#if expired}
			<p class="form-notice" role="status">Your session expired. Please log in again.</p>
		{/if}
		<form onsubmit={handleSubmit}>
			<div class="field">
				<label for="login-email">Email</label>
				<input id="login-email" type="email" autocomplete="email" required bind:value={email} />
			</div>
			<div class="field">
				<label for="login-password">Password</label>
				<input
					id="login-password"
					type="password"
					autocomplete="current-password"
					required
					bind:value={password}
				/>
			</div>
			{#if errorMessage !== ''}
				<p class="form-error" role="alert">{errorMessage}</p>
			{/if}
			<button class="btn btn-primary submit" type="submit" disabled={submitting}>
				{submitting ? 'Logging in…' : 'Log in'}
			</button>
		</form>
		<p class="switch">No account? <a href={resolve('/register')}>Register</a></p>
	</div>
</main>
