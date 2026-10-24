import {
	expect,
	type APIRequestContext,
	type Page,
	type PlaywrightWorkerArgs
} from '@playwright/test';

// Shared steps for the frontend-only seam. The real SvelteKit build runs
// against the stub Connect server, and every case starts from an empty store.
export const STUB_URL = 'http://127.0.0.1:4174';
export const PASSWORD = 'password123';

const CSRF = { 'X-Requested-With': 'XMLHttpRequest' };

/** Opens a request context against the stub Connect server. */
export async function connectContext(
	playwright: PlaywrightWorkerArgs['playwright']
): Promise<APIRequestContext> {
	return playwright.request.newContext({ baseURL: STUB_URL, extraHTTPHeaders: CSRF });
}

/**
 * Registers through the UI and waits for the teams screen. Pass
 * `expectSuccess = false` when the registration is expected to stay on the
 * form (for example a duplicate email).
 */
export async function registerThroughUi(
	page: Page,
	name: string,
	email: string,
	expectSuccess = true
): Promise<void> {
	await page.goto('/register');
	await page.getByLabel('Name', { exact: true }).fill(name);
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
	await page.getByRole('button', { name: 'Register', exact: true }).click();
	if (expectSuccess) {
		await expect(page).toHaveURL(/\/teams$/);
	}
}

/** Logs in through the UI and waits for the teams screen. */
export async function loginThroughUi(page: Page, email: string): Promise<void> {
	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
	await page.getByRole('button', { name: 'Log in', exact: true }).click();
	await expect(page).toHaveURL(/\/teams$/);
}

/** Creates a team from the teams screen and lands on its detail screen. */
export async function createTeamThroughUi(page: Page, name: string): Promise<void> {
	await page.goto('/teams');
	await page.getByLabel('Team name', { exact: true }).fill(name);
	await page.getByRole('button', { name: 'Create team', exact: true }).click();
	await expect(page).toHaveURL(/\/teams\/[a-z0-9-]+$/);
}

/** Creates a season from the current team detail screen. */
export async function createSeasonThroughUi(page: Page, name: string): Promise<void> {
	await page.getByLabel('Season name', { exact: true }).fill(name);
	await page.getByRole('button', { name: 'Create season', exact: true }).click();
	await expect(page.getByRole('link', { name, exact: true })).toBeVisible();
}

/** Accepts the next confirmation dialog triggered by the page. */
export function acceptNextDialog(page: Page): void {
	page.once('dialog', (dialog) => {
		void dialog.accept();
	});
}
