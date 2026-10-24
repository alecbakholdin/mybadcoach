import { expect, test } from '@playwright/test';

import { connectContext, PASSWORD, registerThroughUi, STUB_URL } from './helpers';

// Frontend-only auth seam: the real SvelteKit build runs against the stub
// Connect server through Vite preview's same-origin `/mybadcoach` proxy
// (ticket 04). Every case starts from an empty fake store.
const AUTH = 'mybadcoach.auth.v1.AuthService';

test.beforeEach(async ({ request }) => {
	const reset = await request.post(`${STUB_URL}/__test/reset`);
	expect(reset.ok()).toBe(true);
});

test('visiting a protected route unauthenticated redirects to /login', async ({ page }) => {
	await page.goto('/teams');

	await expect(page).toHaveURL(/\/login$/);
	await expect(page.getByRole('heading', { name: 'Log in' })).toBeVisible();
});

test('register signs the user in and redirects to /teams', async ({ page }) => {
	await registerThroughUi(page, 'Casey Coach', 'casey@example.com');

	await expect(page).toHaveURL(/\/teams$/);
	await expect(page.getByRole('heading', { name: 'Your teams' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Create team' })).toBeVisible();
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
	const api = await connectContext(playwright);
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
	await expect(page.getByRole('heading', { name: 'Your teams' })).toBeVisible();
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
	const api = await connectContext(playwright);
	try {
		const registered = await api.post(`/${AUTH}/Register`, {
			data: { email, password: PASSWORD, name: 'Already Here' }
		});
		expect(registered.status()).toBe(200);
	} finally {
		await api.dispose();
	}

	await registerThroughUi(page, 'Later Arrival', email, false);

	await expect(page.getByRole('alert')).toHaveText('That email is already registered.');
	await expect(page).toHaveURL(/\/register$/);
});
