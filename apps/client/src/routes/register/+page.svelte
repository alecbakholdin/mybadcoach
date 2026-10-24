<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { auth } from '#lib/api';
	import { machineCodeOf } from '#lib/api/errors';
	import { formErrorOf } from '#lib/forms';
	import { session } from '#lib/session.svelte';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let fieldErrors = $state<Record<string, string>>({});
	let errorMessage = $state('');
	let submitting = $state(false);

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		errorMessage = '';
		fieldErrors = {};
		submitting = true;
		try {
			const response = await auth.register({ email, password, name });
			if (response.user !== undefined) {
				session.setUser(response.user);
			}
			await goto(resolve('/teams'));
		} catch (error) {
			if (machineCodeOf(error) === 'email_taken') {
				errorMessage = 'That email is already registered.';
			} else {
				const outcome = formErrorOf(error, 'Could not create your account. Please try again.');
				fieldErrors = outcome.fieldErrors;
				errorMessage = outcome.message;
			}
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Register · My Bad, Coach</title>
</svelte:head>

<main class="screen">
	<div class="card">
		<h1>Register</h1>
		<form onsubmit={handleSubmit}>
			<div class="field">
				<label for="register-name">Name</label>
				<input id="register-name" type="text" autocomplete="name" required bind:value={name} />
				{#if fieldErrors.name}
					<p class="field-error">{fieldErrors.name}</p>
				{/if}
			</div>
			<div class="field">
				<label for="register-email">Email</label>
				<input id="register-email" type="email" autocomplete="email" required bind:value={email} />
				{#if fieldErrors.email}
					<p class="field-error">{fieldErrors.email}</p>
				{/if}
			</div>
			<div class="field">
				<label for="register-password">Password</label>
				<input
					id="register-password"
					type="password"
					autocomplete="new-password"
					required
					bind:value={password}
				/>
				{#if fieldErrors.password}
					<p class="field-error">{fieldErrors.password}</p>
				{/if}
			</div>
			{#if errorMessage !== ''}
				<p class="form-error" role="alert">{errorMessage}</p>
			{/if}
			<button class="btn btn-primary submit" type="submit" disabled={submitting}>
				{submitting ? 'Creating account…' : 'Register'}
			</button>
		</form>
		<p class="switch">Already have an account? <a href={resolve('/login')}>Log in</a></p>
	</div>
</main>
