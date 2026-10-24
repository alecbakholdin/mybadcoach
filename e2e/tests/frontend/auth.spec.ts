import { expect, test, type Page } from '@playwright/test';

// Frontend-only auth seam: the real SvelteKit build runs against the stub
// Connect server through Vite preview's same-origin `/mybadcoach` proxy
// (ticket 04). Every case starts from an empty fake store.
const STUB_URL = 'http://127.0.0.1:4174';
const AUTH = 'mybadcoach.auth.v1.AuthService';

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
}

test('visiting a protected route unauthenticated redirects to /login', async ({ page }) => {
	await page.goto('/teams');

	await expect(page).toHaveURL(/\/login$/);
	await expect(page.getByRole('heading', { name: 'Log in' })).toBeVisible();
});

test('register signs the user in and redirects to /teams', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');

	await expect(page).toHaveURL(/\/teams$/);
	await expect(page.getByRole('heading', { name: 'Your teams' })).toBeVisible();
	await expect(page.getByText('Signed in as Casey Coach')).toBeVisible();
});

test('logout clears the session and returns home', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');
	await expect(page).toHaveURL(/\/teams$/);

	await page.getByRole('button', { name: 'Log out' }).click();

	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { name: 'My Bad, Coach' })).toBeVisible();

	// The session is gone, so the protected route gates again.
	await page.goto('/teams');
	await expect(page).toHaveURL(/\/login$/);
});

test('login accepts a registered account', async ({ page, playwright }) => {
	const email = 'robin@example.com';
	const api = await playwright.request.newContext({
		baseURL: STUB_URL,
		extraHTTPHeaders: { 'X-Requested-With': 'XMLHttpRequest' }
	});
	try {
		const registered = await api.post(`/${AUTH}/Register`, {
			data: { email, password: PASSWORD, name: 'Robin Handler' }
		});
		expect(registered.status()).toBe(200);
	} finally {
		await api.dispose();
	}

	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill(email);
	await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
	await page.getByRole('button', { name: 'Log in', exact: true }).click();

	await expect(page).toHaveURL(/\/teams$/);
	await expect(page.getByText('Signed in as Robin Handler')).toBeVisible();
});

test('login surfaces invalid credentials on the form', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Email', { exact: true }).fill('nobody@example.com');
	await page.getByLabel('Password', { exact: true }).fill('wrong-password');
	await page.getByRole('button', { name: 'Log in', exact: true }).click();

	await expect(page.getByRole('alert')).toHaveText('Invalid email or password.');
	await expect(page).toHaveURL(/\/login$/);
});

test('register surfaces a duplicate email on the form', async ({ page, playwright }) => {
	const email = 'taken@example.com';
	const api = await playwright.request.newContext({
		baseURL: STUB_URL,
		extraHTTPHeaders: { 'X-Requested-With': 'XMLHttpRequest' }
	});
	try {
		const registered = await api.post(`/${AUTH}/Register`, {
			data: { email, password: PASSWORD, name: 'Already Here' }
		});
		expect(registered.status()).toBe(200);
	} finally {
		await api.dispose();
	}

	await registerThroughUi(page, 'Later Arrival', email);

	await expect(page.getByRole('alert')).toHaveText('That email is already registered.');
	await expect(page).toHaveURL(/\/register$/);
});
