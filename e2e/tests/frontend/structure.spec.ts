import { expect, test, type PlaywrightWorkerArgs } from '@playwright/test';

import {
	acceptNextDialog,
	connectContext,
	createSeasonThroughUi,
	createTeamThroughUi,
	loginThroughUi,
	PASSWORD,
	registerThroughUi,
	STUB_URL,
} from './helpers';

// Frontend-only structure CRUD seam (ticket 06): the real SvelteKit build runs
// against the stub Connect server, whose TeamService/SeasonService mirror the
// server contract (nested slugs, owner-gated destructive operations, hard
// deletes). Every case starts from an empty fake store. The redesign moved the
// team list into a sidebar rail and uncommon/destructive actions into Settings
// tabs, but the underlying create → rename → delete and gating flows are the
// same.
const AUTH = 'mybadcoach.auth.v1.AuthService';
const TEAMS = 'mybadcoach.teams.v1.TeamService';

test.beforeEach(async ({ request }) => {
	const reset = await request.post(`${STUB_URL}/__test/reset`);
	expect(reset.ok()).toBe(true);
});

test('creates a team and lists it with the owner role', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	await expect(page.getByRole('heading', { name: 'Fleet Foxes', level: 1 })).toBeVisible();

	await page.goto('/teams');
	const row = page.getByRole('link', { name: 'Fleet Foxes' });
	await expect(row).toBeVisible();
	await expect(row.getByText('Owner', { exact: true })).toBeVisible();
});

test('gives duplicate team names distinct slugs', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	await expect(page).toHaveURL(/\/teams\/fleet-foxes$/);

	await createTeamThroughUi(page, 'Fleet Foxes');
	await expect(page).toHaveURL(/\/teams\/fleet-foxes-2$/);

	await page.goto('/teams');
	await expect(page.getByRole('link', { name: 'Fleet Foxes' })).toHaveCount(2);
});

test('renames a team without changing its slug', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	const detailUrl = page.url();

	await page.getByRole('tab', { name: 'Settings' }).click();
	await page.getByLabel('Team name', { exact: true }).fill('Fleet Foxes FC');
	await page.getByRole('button', { name: 'Rename team', exact: true }).click();

	await expect(page.getByRole('heading', { name: 'Fleet Foxes FC', level: 1 })).toBeVisible();
	expect(page.url()).toBe(detailUrl);
});

test('creates a season, shows the empty roster, and edits its name', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	await createSeasonThroughUi(page, 'Spring 2026');

	await expect(page).toHaveURL(/\/teams\/fleet-foxes\/seasons\/spring-2026$/);
	await expect(page.getByRole('heading', { name: 'Spring 2026', level: 1 })).toBeVisible();

	await page.getByRole('tab', { name: 'Roster' }).click();
	await expect(page.getByText("No players on this season's roster yet.")).toBeVisible();

	await page.getByRole('tab', { name: 'Settings' }).click();
	await page.getByLabel('Season name', { exact: true }).fill('Spring 2026 (updated)');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();

	await expect(
		page.getByRole('heading', { name: 'Spring 2026 (updated)', level: 1 }),
	).toBeVisible();
});

test('lets the owner delete a season and then the team', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	await createSeasonThroughUi(page, 'Spring 2026');
	await expect(page).toHaveURL(/\/seasons\/spring-2026$/);

	await page.getByRole('tab', { name: 'Settings' }).click();
	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete season', exact: true }).click();

	await expect(page).toHaveURL(/\/teams\/fleet-foxes$/);
	await expect(page.getByRole('link', { name: 'Spring 2026' })).toHaveCount(0);

	await page.getByRole('tab', { name: 'Settings' }).click();
	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete team', exact: true }).click();

	await expect(page).toHaveURL(/\/teams$/);
	await expect(page.getByRole('link', { name: 'Fleet Foxes' })).toHaveCount(0);
});

/** Seeds an owner + a non-owner member through the Connect boundary. */
async function seedTeamWithMember(playwright: PlaywrightWorkerArgs['playwright']): Promise<string> {
	const ownerApi = await connectContext(playwright);
	try {
		const registered = await ownerApi.post(`/${AUTH}/Register`, {
			data: { email: 'owner@example.com', password: PASSWORD, name: 'Owner One' },
		});
		expect(registered.ok()).toBe(true);

		const created = await ownerApi.post(`/${TEAMS}/CreateTeam`, {
			data: { name: 'Fleet Foxes' },
		});
		expect(created.ok()).toBe(true);
		const body = (await created.json()) as { team?: { slug?: string } };
		const teamSlug = body.team?.slug ?? '';
		expect(teamSlug).toBe('fleet-foxes');

		const memberApi = await connectContext(playwright);
		try {
			const memberRegistered = await memberApi.post(`/${AUTH}/Register`, {
				data: { email: 'member@example.com', password: PASSWORD, name: 'Member Two' },
			});
			expect(memberRegistered.ok()).toBe(true);
		} finally {
			await memberApi.dispose();
		}

		const added = await ownerApi.post(`/${TEAMS}/AddMembership`, {
			data: { teamSlug, email: 'member@example.com' },
		});
		expect(added.ok()).toBe(true);
		return teamSlug;
	} finally {
		await ownerApi.dispose();
	}
}

test('hides destructive team and season actions from a non-owner member', async ({
	page,
	playwright,
}) => {
	const teamSlug = await seedTeamWithMember(playwright);

	await loginThroughUi(page, 'member@example.com');

	const row = page.getByRole('link', { name: 'Fleet Foxes' });
	await expect(row.getByText('Member', { exact: true })).toBeVisible();

	await page.goto(`/teams/${teamSlug}`);
	await expect(page.getByRole('heading', { name: 'Fleet Foxes', level: 1 })).toBeVisible();
	await expect(page.getByRole('button', { name: 'New season' })).toBeVisible();

	await page.getByRole('tab', { name: 'Settings' }).click();
	await expect(page.getByRole('button', { name: 'Rename team', exact: true })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Delete team', exact: true })).toHaveCount(0);

	await createSeasonThroughUi(page, 'Spring 2026');
	await expect(page.getByRole('heading', { name: 'Spring 2026', level: 1 })).toBeVisible();
	await page.getByRole('tab', { name: 'Settings' }).click();
	await expect(page.getByRole('button', { name: 'Save changes', exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Delete season', exact: true })).toHaveCount(0);
});

test('clears season dates when the edit fields are emptied', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	await page.getByRole('button', { name: 'New season' }).click();
	await page.getByLabel('Season name', { exact: true }).fill('Spring 2026');
	await page.getByLabel('Start date', { exact: true }).fill('2026-01-15');
	await page.getByLabel('End date', { exact: true }).fill('2026-03-01');
	await page.getByRole('button', { name: 'Create season', exact: true }).click();

	await expect(page.getByRole('heading', { name: 'Spring 2026', level: 1 })).toBeVisible();
	await page.getByRole('tab', { name: 'Settings' }).click();

	await expect(page.getByLabel('Start date', { exact: true })).toHaveValue('2026-01-15');
	await page.getByLabel('Start date', { exact: true }).fill('');
	await page.getByLabel('End date', { exact: true }).fill('');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();

	// A blank date is sent as an explicit empty value that clears the column;
	// the refreshed form therefore stays empty.
	await expect(page.getByLabel('Start date', { exact: true })).toHaveValue('');
	await expect(page.getByLabel('End date', { exact: true })).toHaveValue('');
});

test('shows the season not-found state for a stale link', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	await page.goto('/teams/fleet-foxes/seasons/spring-2026');

	await expect(page.getByRole('heading', { name: 'Season not found' })).toBeVisible();
});
