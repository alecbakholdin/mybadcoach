import { expect, test, type PlaywrightWorkerArgs } from '@playwright/test';

import {
	acceptNextDialog,
	connectContext,
	createSeasonThroughUi,
	createTeamThroughUi,
	loginThroughUi,
	PASSWORD,
	registerThroughUi,
	STUB_URL
} from './helpers';

// Frontend-only player + roster seam (ticket 08): the real SvelteKit build runs
// against the stub Connect server, whose PlayerService and roster RPCs mirror
// the server contract (team-scoped slugs, idempotent roster adds, hard deletes,
// owner-only player deletion, member-gated roster mutations). Every case starts
// from an empty fake store.
const AUTH = 'mybadcoach.auth.v1.AuthService';
const TEAMS = 'mybadcoach.teams.v1.TeamService';
const PLAYERS = 'mybadcoach.players.v1.PlayerService';
const SEASONS = 'mybadcoach.seasons.v1.SeasonService';
const EVENTS = 'mybadcoach.events.v1.EventService';

test.beforeEach(async ({ request }) => {
	const reset = await request.post(`${STUB_URL}/__test/reset`);
	expect(reset.ok()).toBe(true);
});

test('creates, edits, rosters, and deletes a player', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	// The team screen links to player management.
	await page.getByRole('link', { name: 'Players', exact: true }).click();
	await expect(page).toHaveURL(/\/teams\/fleet-foxes\/players$/);
	await expect(page.getByRole('heading', { name: 'Players', level: 1 })).toBeVisible();

	// Create: display name first, aliases after.
	await page.getByLabel('Display name', { exact: true }).fill('Ada Lovelace');
	await page.getByLabel('Aliases (comma-separated)', { exact: true }).fill('Ada, Lovelace');
	await page.getByRole('button', { name: 'Add player', exact: true }).click();

	const playersList = page.getByRole('list', { name: 'Team players' });
	const adaRow = playersList.getByRole('listitem').filter({ hasText: 'Ada Lovelace' });
	await expect(adaRow).toBeVisible();
	await expect(adaRow.getByText('Ada, Lovelace', { exact: true })).toBeVisible();

	// Edit replaces the whole name list; the slug stays stable. While editing,
	// the row shows inputs rather than the old text, so target the list itself.
	await adaRow.getByRole('button', { name: 'Edit', exact: true }).click();
	await playersList.getByLabel('Display name', { exact: true }).fill('Ada King');
	await playersList
		.getByLabel('Aliases (comma-separated)', { exact: true })
		.fill('Ada, Lovelace');
	await playersList.getByRole('button', { name: 'Save player', exact: true }).click();
	await expect(
		playersList.getByRole('listitem').filter({ hasText: 'Ada King' })
	).toBeVisible();

	// Create a season and event to roster the player at both independent levels.
	await page.goto('/teams/fleet-foxes');
	await createSeasonThroughUi(page, 'Spring 2026');
	await page.getByRole('link', { name: 'Spring 2026', exact: true }).click();
	await expect(page).toHaveURL(/\/teams\/fleet-foxes\/seasons\/spring-2026$/);

	const seasonRoster = page.getByRole('list', { name: 'Season roster' });
	await page.getByLabel('Player to add', { exact: true }).selectOption({ label: 'Ada King' });
	await page.getByRole('button', { name: 'Add to season roster', exact: true }).click();
	await expect(
		seasonRoster.getByRole('listitem').filter({ hasText: 'Ada King' })
	).toBeVisible();

	await page.getByLabel('Event name', { exact: true }).fill('Summer Classic');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await page.getByRole('link', { name: 'Summer Classic', exact: true }).click();
	await expect(page).toHaveURL(
		/\/teams\/fleet-foxes\/seasons\/spring-2026\/events\/summer-classic$/
	);

	const eventRoster = page.getByRole('list', { name: 'Event roster' });
	await page.getByLabel('Player to add', { exact: true }).selectOption({ label: 'Ada King' });
	await page.getByRole('button', { name: 'Add to event roster', exact: true }).click();
	await expect(
		eventRoster.getByRole('listitem').filter({ hasText: 'Ada King' })
	).toBeVisible();

	// Removing from the event roster leaves the season roster untouched.
	await eventRoster.getByRole('button', { name: 'Remove', exact: true }).click();
	await expect(page.getByText("No players on this event's roster yet.")).toBeVisible();

	await page.goto('/teams/fleet-foxes/seasons/spring-2026');
	await expect(
		page
			.getByRole('list', { name: 'Season roster' })
			.getByRole('listitem')
			.filter({ hasText: 'Ada King' })
	).toBeVisible();
	await page.getByRole('list', { name: 'Season roster' })
		.getByRole('button', { name: 'Remove', exact: true })
		.click();
	await expect(page.getByText("No players on this season's roster yet.")).toBeVisible();

	// Owner-only hard delete from the players screen.
	await page.goto('/teams/fleet-foxes/players');
	acceptNextDialog(page);
	await page
		.getByRole('list', { name: 'Team players' })
		.getByRole('button', { name: 'Delete', exact: true })
		.click();
	await expect(page.getByText('No players yet.')).toBeVisible();
});

interface SeededPlayer {
	teamSlug: string;
	playerSlug: string;
	seasonSlug: string;
	eventSlug: string;
}

/**
 * Seeds an owner plus a non-owner member with a player, season, and event via
 * the Connect boundary. Also proves the server rejects a member's DeletePlayer.
 */
async function seedMemberWithPlayer(
	playwright: PlaywrightWorkerArgs['playwright']
): Promise<SeededPlayer> {
	const ownerApi = await connectContext(playwright);
	try {
		const registered = await ownerApi.post(`/${AUTH}/Register`, {
			data: { email: 'owner@example.com', password: PASSWORD, name: 'Owner One' }
		});
		expect(registered.ok()).toBe(true);

		const createdTeam = await ownerApi.post(`/${TEAMS}/CreateTeam`, {
			data: { name: 'Fleet Foxes' }
		});
		expect(createdTeam.ok()).toBe(true);
		const teamSlug =
			((await createdTeam.json()) as { team?: { slug?: string } }).team?.slug ?? '';
		expect(teamSlug).toBe('fleet-foxes');

		const createdPlayer = await ownerApi.post(`/${PLAYERS}/CreatePlayer`, {
			data: { teamSlug, names: ['Ada Lovelace', 'Ada'] }
		});
		expect(createdPlayer.ok()).toBe(true);
		const playerSlug =
			((await createdPlayer.json()) as { player?: { slug?: string } }).player?.slug ?? '';
		expect(playerSlug).toBe('ada-lovelace');

		const createdSeason = await ownerApi.post(`/${SEASONS}/CreateSeason`, {
			data: { teamSlug, name: 'Spring 2026' }
		});
		expect(createdSeason.ok()).toBe(true);
		const seasonSlug =
			((await createdSeason.json()) as { season?: { slug?: string } }).season?.slug ?? '';
		expect(seasonSlug).toBe('spring-2026');

		const createdEvent = await ownerApi.post(`/${EVENTS}/CreateEvent`, {
			data: { teamSlug, seasonSlug, name: 'Summer Classic' }
		});
		expect(createdEvent.ok()).toBe(true);
		const eventSlug =
			((await createdEvent.json()) as { event?: { slug?: string } }).event?.slug ?? '';
		expect(eventSlug).toBe('summer-classic');

		const memberApi = await connectContext(playwright);
		try {
			const memberRegistered = await memberApi.post(`/${AUTH}/Register`, {
				data: { email: 'member@example.com', password: PASSWORD, name: 'Member Two' }
			});
			expect(memberRegistered.ok()).toBe(true);

			const added = await ownerApi.post(`/${TEAMS}/AddMembership`, {
				data: { teamSlug, email: 'member@example.com' }
			});
			expect(added.ok()).toBe(true);

			// A member is allowed on rosters but not on player deletion.
			const denied = await memberApi.post(`/${PLAYERS}/DeletePlayer`, {
				data: { teamSlug, playerSlug }
			});
			expect(denied.ok()).toBe(false);
			expect(denied.status()).toBe(403);
		} finally {
			await memberApi.dispose();
		}

		return { teamSlug, playerSlug, seasonSlug, eventSlug };
	} finally {
		await ownerApi.dispose();
	}
}

test('lets a non-owner member manage rosters but not delete players', async ({
	page,
	playwright
}) => {
	const { teamSlug, seasonSlug, eventSlug } = await seedMemberWithPlayer(playwright);

	await loginThroughUi(page, 'member@example.com');

	// Members may create and edit players, but not delete them.
	await page.goto(`/teams/${teamSlug}/players`);
	await expect(page.getByRole('heading', { name: 'Players', level: 1 })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Add player', exact: true })).toBeVisible();
	const playerRow = page
		.getByRole('list', { name: 'Team players' })
		.getByRole('listitem')
		.filter({ hasText: 'Ada Lovelace' });
	await expect(playerRow).toBeVisible();
	await expect(playerRow.getByRole('button', { name: 'Edit', exact: true })).toBeVisible();
	await expect(playerRow.getByRole('button', { name: 'Delete', exact: true })).toHaveCount(0);

	// Season roster add/remove is member-gated, not owner-gated.
	await page.goto(`/teams/${teamSlug}/seasons/${seasonSlug}`);
	await page.getByLabel('Player to add', { exact: true }).selectOption({ label: 'Ada Lovelace' });
	await page.getByRole('button', { name: 'Add to season roster', exact: true }).click();
	const seasonRoster = page.getByRole('list', { name: 'Season roster' });
	await expect(
		seasonRoster.getByRole('listitem').filter({ hasText: 'Ada Lovelace' })
	).toBeVisible();
	await seasonRoster.getByRole('button', { name: 'Remove', exact: true }).click();
	await expect(page.getByText("No players on this season's roster yet.")).toBeVisible();

	// Event roster add/remove is independently member-gated.
	await page.goto(`/teams/${teamSlug}/seasons/${seasonSlug}/events/${eventSlug}`);
	await page.getByLabel('Player to add', { exact: true }).selectOption({ label: 'Ada Lovelace' });
	await page.getByRole('button', { name: 'Add to event roster', exact: true }).click();
	const eventRoster = page.getByRole('list', { name: 'Event roster' });
	await expect(
		eventRoster.getByRole('listitem').filter({ hasText: 'Ada Lovelace' })
	).toBeVisible();
	await eventRoster.getByRole('button', { name: 'Remove', exact: true }).click();
	await expect(page.getByText("No players on this event's roster yet.")).toBeVisible();
});
