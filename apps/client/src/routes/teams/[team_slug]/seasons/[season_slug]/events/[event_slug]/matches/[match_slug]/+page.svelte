<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { matches as matchClient } from '#lib/api';
	import { formErrorOf } from '#lib/forms';
	import type { MatchDetailData } from '#lib/structure-data';

	let { data }: { data: MatchDetailData } = $props();

	let editOpponent = $derived(data.match?.opponentName ?? '');
	let editScheduledAt = $derived(data.match?.scheduledAt ?? '');
	let editPointCap = $derived.by(() => {
		if (data.match === undefined || data.match.pointCap === 0n) {
			return '';
		}
		return String(data.match.pointCap);
	});
	let fieldErrors = $state<Record<string, string>>({});
	let saveError = $state('');
	let saving = $state(false);

	let deleting = $state(false);
	let deleteError = $state('');

	/** Empty stays absent; the number input keeps the rest integral. */
	function parsedPointCap(value: string): bigint | undefined {
		const trimmed = value.trim();
		if (trimmed === '' || !/^\d+$/.test(trimmed)) {
			return undefined;
		}
		return BigInt(trimmed);
	}

	async function handleSave(submit: SubmitEvent): Promise<void> {
		submit.preventDefault();
		saveError = '';
		fieldErrors = {};
		saving = true;
		try {
			await matchClient.updateMatch({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
				matchSlug: data.matchSlug,
				opponentName: editOpponent,
				scheduledAt: editScheduledAt,
				pointCap: parsedPointCap(editPointCap)
			});
			await refreshAll();
		} catch (error) {
			const outcome = formErrorOf(error, 'Could not save the match.');
			fieldErrors = outcome.fieldErrors;
			saveError = outcome.message;
		} finally {
			saving = false;
		}
	}

	async function handleDelete(): Promise<void> {
		if (
			!confirm(
				`Delete the match against ${data.match?.opponentName ?? 'this opponent'}?`
			)
		) {
			return;
		}
		deleteError = '';
		deleting = true;
		try {
			await matchClient.deleteMatch({
				teamSlug: data.teamSlug,
				seasonSlug: data.seasonSlug,
				eventSlug: data.eventSlug,
				matchSlug: data.matchSlug
			});
			await goto(
				resolve('/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]', {
					team_slug: data.teamSlug,
					season_slug: data.seasonSlug,
					event_slug: data.eventSlug
				})
			);
		} catch {
			deleteError = 'Could not delete the match.';
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>{data.match?.opponentName ?? 'Match'} · My Bad, Coach</title>
</svelte:head>

<main class="page">
	<a
		class="back-link"
		href={resolve('/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]', {
			team_slug: data.teamSlug,
			season_slug: data.seasonSlug,
			event_slug: data.eventSlug
		})}
	>
		← {data.eventName || 'Event'}
	</a>

	{#if data.match === undefined}
		<div class="card">
			<h1>Match not found</h1>
			<p class="muted">This match does not exist or you are not a member of its team.</p>
		</div>
	{:else}
		<div class="page-head">
			<h1>{data.match.opponentName}</h1>
		</div>

		<section class="card section-card">
			<h2>Match details</h2>
			<form onsubmit={handleSave}>
				<div class="field">
					<label for="edit-match-opponent">Opponent name</label>
					<input id="edit-match-opponent" type="text" required bind:value={editOpponent} />
					{#if fieldErrors.opponent_name}
						<p class="field-error">{fieldErrors.opponent_name}</p>
					{/if}
				</div>
				<div class="date-range">
					<div class="field">
						<label for="edit-match-scheduled">Scheduled at</label>
						<input
							id="edit-match-scheduled"
							type="datetime-local"
							bind:value={editScheduledAt}
						/>
					</div>
					<div class="field">
						<label for="edit-match-point-cap">Point cap</label>
						<input
							id="edit-match-point-cap"
							type="number"
							min="0"
							step="1"
							bind:value={editPointCap}
						/>
					</div>
				</div>
				{#if saveError !== ''}
					<p class="form-error" role="alert">{saveError}</p>
				{/if}
				<button class="btn btn-primary" type="submit" disabled={saving}>
					{saving ? 'Saving…' : 'Save changes'}
				</button>
			</form>
			<div class="danger-zone">
				{#if deleteError !== ''}
					<p class="form-error" role="alert">{deleteError}</p>
				{/if}
				<button class="btn btn-danger" type="button" disabled={deleting} onclick={handleDelete}>
					{deleting ? 'Deleting…' : 'Delete match'}
				</button>
			</div>
		</section>
	{/if}
</main>
