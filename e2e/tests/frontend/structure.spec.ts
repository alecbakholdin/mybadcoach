import { expect, test, type Page, type PlaywrightWorkerArgs } from '@playwright/test';

// Frontend-only structure CRUD seam (ticket 06): the real SvelteKit build runs
// against the stub Connect server, whose TeamService/SeasonService mirror the
// server contract (nested slugs, owner-gated destructive operations, hard
// deletes). Every case starts from an empty fake store.
const STUB_URL = 'http://127.0.0.1:4174';
const AUTH = 'mybadcoach.auth.v1.AuthService';
const TEAMS = 'mybadcoach.teams.v1.TeamService';

const PASSWORD = 'password123';

test.beforeEach(async ({ request }) => {
	const reset = await request.post(`${STUB_URL}/__test/reset`);
	expect(reset.ok()).toBe(true);
});

async function registerThroughUi(page: Page, name: string, email: string): Promise<void> {
	await page.goto('/register');
	await page.getByLabel('Name', { exact: true }).fill(name);
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
	await page.getByRole('button', { name: 'Register', exact: true }).click();
	await expect(page).toHaveURL(/\/teams$/);
}

async function createTeamThroughUi(page: Page, name: string): Promise<void> {
	await page.goto('/teams');
	await page.getByLabel('Team name', { exact: true }).fill(name);
	await page.getByRole('button', { name: 'Create team', exact: true }).click();
	await expect(page).toHaveURL(/\/teams\/[a-z0-9-]+$/);
}

async function createSeasonThroughUi(page: Page, name: string): Promise<void> {
	await page.getByLabel('Season name', { exact: true }).fill(name);
	await page.getByRole('button', { name: 'Create season', exact: true }).click();
	await expect(page.getByRole('link', { name, exact: true })).toBeVisible();
}

function acceptNextDialog(page: Page): void {
	page.once('dialog', (dialog) => {
		void dialog.accept();
	});
}

test('creates a team and lists it with the owner role', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	await expect(page.getByRole('heading', { name: 'Fleet Foxes', level: 1 })).toBeVisible();

	await page.goto('/teams');
	const row = page.getByRole('listitem').filter({ hasText: 'Fleet Foxes' });
	await expect(row.getByRole('link', { name: 'Fleet Foxes', exact: true })).toBeVisible();
	await expect(row.getByText('Owner', { exact: true })).toBeVisible();
});

test('gives duplicate team names distinct slugs', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	await expect(page).toHaveURL(/\/teams\/fleet-foxes$/);

	await createTeamThroughUi(page, 'Fleet Foxes');
	await expect(page).toHaveURL(/\/teams\/fleet-foxes-2$/);

	await page.goto('/teams');
	await expect(page.getByRole('listitem')).toHaveCount(2);
});

test('renames a team without changing its slug', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	const detailUrl = page.url();

	await page.getByLabel('Team name', { exact: true }).fill('Fleet Foxes FC');
	await page.getByRole('button', { name: 'Rename team', exact: true }).click();

	await expect(page.getByRole('heading', { name: 'Fleet Foxes FC', level: 1 })).toBeVisible();
	expect(page.url()).toBe(detailUrl);
});

test('creates, edits, and shows the season roster', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');

	await createSeasonThroughUi(page, 'Spring 2026');
	await page.getByRole('link', { name: 'Spring 2026', exact: true }).click();

	await expect(page).toHaveURL(/\/teams\/fleet-foxes\/seasons\/spring-2026$/);
	await expect(page.getByRole('heading', { name: 'Spring 2026', level: 1 })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Season roster' })).toBeVisible();
	await expect(page.getByText("No players on this season's roster yet.")).toBeVisible();

	await page.getByLabel('Season name', { exact: true }).fill('Spring 2026 (updated)');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();

	await expect(
		page.getByRole('heading', { name: 'Spring 2026 (updated)', level: 1 })
	).toBeVisible();
});

test('lets the owner delete a season and then the team', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await createTeamThroughUi(page, 'Fleet Foxes');
	await createSeasonThroughUi(page, 'Spring 2026');
	await page.getByRole('link', { name: 'Spring 2026', exact: true }).click();
	await expect(page).toHaveURL(/\/seasons\/spring-2026$/);

	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete season', exact: true }).click();

	await expect(page).toHaveURL(/\/teams\/fleet-foxes$/);
	await expect(page.getByRole('link', { name: 'Spring 2026', exact: true })).toHaveCount(0);

	acceptNextDialog(page);
	await page.getByRole('button', { name: 'Delete team', exact: true }).click();

	await expect(page).toHaveURL(/\/teams$/);
	await expect(page.getByRole('listitem')).toHaveCount(0);
});

/** Seeds an owner + a non-owner member through the Connect boundary. */
async function seedTeamWithMember(
	playwright: PlaywrightWorkerArgs['playwright']
): Promise<string> {
	const ownerApi = await playwright.request.newContext({
		baseURL: STUB_URL,
		extraHTTPHeaders: { 'X-Requested-With': 'XMLHttpRequest' }
	});
	try {
		const registered = await ownerApi.post(`/${AUTH}/Register`, {
			data: { email: 'owner@example.com', password: PASSWORD, name: 'Owner One' }
		});
		expect(registered.ok()).toBe(true);

		const created = await ownerApi.post(`/${TEAMS}/CreateTeam`, {
			data: { name: 'Fleet Foxes' }
		});
		expect(created.ok()).toBe(true);
		const body = (await created.json()) as { team?: { slug?: string } };
		const teamSlug = body.team?.slug ?? '';
		expect(teamSlug).toBe('fleet-foxes');

		const memberApi = await playwright.request.newContext({
			baseURL: STUB_URL,
			extraHTTPHeaders: { 'X-Requested-With': 'XMLHttpRequest' }
		});
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
		return teamSlug;
	} finally {
		await ownerApi.dispose();
	}
}

test('hides destructive team actions from a non-owner member', async ({ page, playwright }) => {
	const teamSlug = await seedTeamWithMember(playwright);

	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill('member@example.com');
	await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
	await page.getByRole('button', { name: 'Log in', exact: true }).click();
	await expect(page).toHaveURL(/\/teams$/);

	const row = page.getByRole('listitem').filter({ hasText: 'Fleet Foxes' });
	await expect(row.getByText('Member', { exact: true })).toBeVisible();

	await page.goto(`/teams/${teamSlug}`);
	await expect(page.getByRole('heading', { name: 'Fleet Foxes', level: 1 })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Create season', exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Rename team', exact: true })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Delete team', exact: true })).toHaveCount(0);

	await createSeasonThroughUi(page, 'Spring 2026');
	await page.getByRole('link', { name: 'Spring 2026', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Save changes', exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Delete season', exact: true })).toHaveCount(0);
});
