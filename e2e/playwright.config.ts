import { defineConfig } from '@playwright/test';

// The container stack is orchestrated by docker compose: the server and client
// are addressed by their compose service names. Override the URLs when running
// the suite outside the composed network (e.g. against a local server).
export default defineConfig({
	testDir: './tests',
	// The frontend-only seam runs against the stub Connect server under its own
	// config (`playwright.frontend.config.ts`); the full-stack suite excludes it.
	testIgnore: ['**/frontend/**'],
	fullyParallel: false,
	workers: 1,
	retries: 0,
	timeout: 60_000,
	reporter: [['list']],
	use: {
		trace: 'retain-on-failure',
	},
});
