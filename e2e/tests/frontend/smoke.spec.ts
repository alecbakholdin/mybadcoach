import { expect, test } from '@playwright/test';

// Frontend-only smoke: proves the real client build renders against the stub
// Connect server and that the stub is a real Connect boundary (real framing,
// real HTTP status codes, real cookie sessions).
const STUB_URL = 'http://127.0.0.1:4174';
const AUTH = 'mybadcoach.auth.v1.AuthService';

test.beforeEach(async ({ request }) => {
	const reset = await request.post(`${STUB_URL}/__test/reset`);
	expect(reset.ok()).toBe(true);
});

test('renders the client app shell', async ({ page }) => {
	const response = await page.goto('/');

	expect(response?.status()).toBe(200);
	await expect(
		page.getByRole('button', { name: /Switch to (light|dark) theme/ })
	).toBeVisible();
	await expect(page.getByRole('heading', { name: 'My Bad, Coach' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Log in' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Register' })).toBeVisible();
});

test('serves Connect unauthenticated for Me without a session', async ({ playwright }) => {
	const api = await playwright.request.newContext({
		baseURL: STUB_URL,
		extraHTTPHeaders: { 'X-Requested-With': 'XMLHttpRequest' }
	});

	try {
		const response = await api.post(`/${AUTH}/Me`, { data: {} });

		expect(response.status()).toBe(401);
		expect(await response.json()).toMatchObject({ code: 'unauthenticated' });
	} finally {
		await api.dispose();
	}
});

test('mints a real session cookie that Me honors', async ({ playwright }) => {
	const api = await playwright.request.newContext({
		baseURL: STUB_URL,
		extraHTTPHeaders: { 'X-Requested-With': 'XMLHttpRequest' }
	});

	try {
		const email = `smoke-${String(Date.now())}@mybadcoach.dev`;
		const registered = await api.post(`/${AUTH}/Register`, {
			data: { email, password: 'password123', name: 'Smoke Tester' }
		});

		expect(registered.status()).toBe(200);
		const state = await api.storageState();
		expect(state.cookies.find((cookie) => cookie.name === 'mbc_session')).toBeDefined();

		const me = await api.post(`/${AUTH}/Me`, { data: {} });

		expect(me.status()).toBe(200);
		expect(await me.json()).toMatchObject({ user: { email } });
	} finally {
		await api.dispose();
	}
});
