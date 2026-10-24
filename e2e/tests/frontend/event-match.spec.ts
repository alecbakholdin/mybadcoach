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

// Frontend-only event + match CRUD seam (ticket 07): the real SvelteKit build
// runs against the stub Connect server, whose EventService/MatchService mirror
// the server contract (nested slugs, member-gated deletes, hard deletes).
// Every case starts from an empty fake store.
const AUTH = 'mybadcoach.auth.v1.AuthService';
const TEAMS = 'mybadcoach.teams.v1.TeamService';
const SEASONS = 'mybadcoach.seasons.v1.SeasonService';
const EVENTS = 'mybadcoach.events.v1.EventService';
const MATCHES = 'mybadcoach.matches.v1.MatchService';

test.beforeEach(async ({ request }) => {
	const reset = await request.post(`${STUB_URL}/__test/reset`);
	expect(reset.ok()).toBe(true);
});

test('creates, edits, and deletes an event and its matches', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	await createSeasonThroughUi(page, 'Spring 2026');
	await page.getByRole('link', { name: 'Spring 2026', exact: true }).click();
	await expect(page).toHaveURL(/\/teams\/fleet-foxes\/seasons\/spring-2026$/);

	// Create the event and list it on the season screen.
	await page.getByLabel('Event name', { exact: true }).fill('Summer Classic');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(
		page.getByRole('link', { name: 'Summer Classic', exact: true })
	).toBeVisible();

	await page.getByRole('link', { name: 'Summer Classic', exact: true }).click();
	await expect(page).toHaveURL(
		/\/teams\/fleet-foxes\/seasons\/spring-2026\/events\/summer-classic$/
	);
	await expect(
		page.getByRole('heading', { name: 'Summer Classic', level: 1 })
	).toBeVisible();

	// Edit the event.
	await page.getByLabel('Event name', { exact: true }).fill('Summer Classic 2026');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Summer Classic 2026', level: 1 })
	).toBeVisible();

	// Create a match and list it on the event screen.
	await page.getByLabel('Opponent name', { exact: true }).fill('Rival Ultimate');
	await page.getByRole('button', { name: 'Create match', exact: true }).click();
	await expect(
		page.getByRole('link', { name: 'Rival Ultimate', exact: true })
	).toBeVisible();

	await page.getByRole('link', { name: 'Rival Ultimate', exact: true }).click();
	await expect(page).toHaveURL(
		/\/teams\/fleet-foxes\/seasons\/spring-2026\/events\/summer-classic\/matches\/rival-ultimate$/
	);
	await expect(
		page.getByRole('heading', { name: 'Rival Ultimate', level: 1 })
	).toBeVisible();

	// Edit the match.
	await page.getByLabel('Opponent name', { exact: true }).fill('Rival Ultimate Club');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Rival Ultimate Club', level: 1 })
	).toBeVisible();

	// Delete the match, returning to the event (whose slug is stable across the
	// event rename).
	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete match', exact: true }).click();
	await expect(page).toHaveURL(
		/\/teams\/fleet-foxes\/seasons\/spring-2026\/events\/summer-classic$/
	);
	await expect(
		page.getByRole('link', { name: 'Rival Ultimate Club', exact: true })
	).toHaveCount(0);

	// Delete the event, returning to the season.
	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete event', exact: true }).click();
	await expect(page).toHaveURL(/\/teams\/fleet-foxes\/seasons\/spring-2026$/);
	await expect(
		page.getByRole('link', { name: 'Summer Classic 2026', exact: true })
	).toHaveCount(0);
});

interface SeededStructure {
	teamSlug: string;
	seasonSlug: string;
	eventSlug: string;
	matchSlug: string;
}

/** Seeds an owner plus a non-owner member with an event and match, via Connect. */
async function seedStructureWithMatch(
	playwright: PlaywrightWorkerArgs['playwright']
): Promise<SeededStructure> {
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

		const createdSeason = await ownerApi.post(`/${SEASONS}/CreateSeason`, {
			data: { teamSlug, name: 'Spring 2026' }
		});
		expect(createdSeason.ok()).toBe(true);
		const seasonSlug =
			((await createdSeason.json()) as { season?: { slug?: string } }).season?.slug ??
			'';
		expect(seasonSlug).toBe('spring-2026');

		const createdEvent = await ownerApi.post(`/${EVENTS}/CreateEvent`, {
			data: { teamSlug, seasonSlug, name: 'Summer Classic' }
		});
		expect(createdEvent.ok()).toBe(true);
		const eventSlug =
			((await createdEvent.json()) as { event?: { slug?: string } }).event?.slug ?? '';
		expect(eventSlug).toBe('summer-classic');

		const createdMatch = await ownerApi.post(`/${MATCHES}/CreateMatch`, {
			data: { teamSlug, seasonSlug, eventSlug, opponentName: 'Rival Ultimate' }
		});
		expect(createdMatch.ok()).toBe(true);
		const matchSlug =
			((await createdMatch.json()) as { match?: { slug?: string } }).match?.slug ?? '';
		expect(matchSlug).toBe('rival-ultimate');

		const memberApi = await connectContext(playwright);
		try {
			const memberRegistered = await memberApi.post(`/${AUTH}/Register`, {
				data: { email: 'member@example.com', password: PASSWORD, name: 'Member Two' }
			});
			expect(memberRegistered.ok()).toBe(true);
		} finally {
			await memberApi.dispose();
		}

		const added = await ownerApi.post(`/${TEAMS}/AddMembership`, {
			data: { teamSlug, email: 'member@example.com' }
		});
		expect(added.ok()).toBe(true);
		return { teamSlug, seasonSlug, eventSlug, matchSlug };
	} finally {
		await ownerApi.dispose();
	}
}

test('lets a non-owner member create, edit, and delete events and matches', async ({
	page,
	playwright
}) => {
	const { teamSlug, seasonSlug, eventSlug, matchSlug } =
		await seedStructureWithMatch(playwright);

	await loginThroughUi(page, 'member@example.com');

	await page.goto(`/teams/${teamSlug}/seasons/${seasonSlug}`);
	await expect(page.getByRole('button', { name: 'Create event', exact: true })).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Summer Classic', exact: true })
	).toBeVisible();

	// Membership (not just ownership) grants structure mutations.
	await page.getByLabel('Event name', { exact: true }).fill('Fall League');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(page.getByRole('link', { name: 'Fall League', exact: true })).toBeVisible();
	await page.getByRole('link', { name: 'Fall League', exact: true }).click();
	await expect(page.getByRole('heading', { name: 'Fall League', level: 1 })).toBeVisible();
	await page.getByLabel('Event name', { exact: true }).fill('Fall League 2026');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Fall League 2026', level: 1 })
	).toBeVisible();

	await page.goto(`/teams/${teamSlug}/seasons/${seasonSlug}/events/${eventSlug}`);
	await expect(
		page.getByRole('heading', { name: 'Summer Classic', level: 1 })
	).toBeVisible();
	await expect(page.getByRole('button', { name: 'Create match', exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Delete event', exact: true })).toBeVisible();

	await page.getByLabel('Opponent name', { exact: true }).fill('Member Match');
	await page.getByRole('button', { name: 'Create match', exact: true }).click();
	await expect(
		page.getByRole('link', { name: 'Member Match', exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Member Match', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Member Match', level: 1 })
	).toBeVisible();
	await page.getByLabel('Opponent name', { exact: true }).fill('Member Match Club');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Member Match Club', level: 1 })
	).toBeVisible();

	await page.goto(
		`/teams/${teamSlug}/seasons/${seasonSlug}/events/${eventSlug}/matches/${matchSlug}`
	);
	await expect(
		page.getByRole('heading', { name: 'Rival Ultimate', level: 1 })
	).toBeVisible();
	await expect(page.getByRole('button', { name: 'Delete match', exact: true })).toBeVisible();

	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete match', exact: true }).click();
	await expect(page).toHaveURL(
		new RegExp(
			`/teams/${teamSlug}/seasons/${seasonSlug}/events/${eventSlug}$`
		)
	);
	await expect(
		page.getByRole('link', { name: 'Rival Ultimate', exact: true })
	).toHaveCount(0);

	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete event', exact: true }).click();
	await expect(page).toHaveURL(new RegExp(`/teams/${teamSlug}/seasons/${seasonSlug}$`));
	await expect(
		page.getByRole('link', { name: 'Summer Classic', exact: true })
	).toHaveCount(0);
});
