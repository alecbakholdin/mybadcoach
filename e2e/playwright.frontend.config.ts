import { fileURLToPath } from 'node:url';

import { defineConfig } from '@playwright/test';

// Frontend-only integration seam (docs/testing.md): the real SvelteKit build is
// driven in a browser, with the only test double being the stub Connect server
// at the network boundary. The full-stack suite keeps its own config.
const repoRoot = fileURLToPath(new URL('..', import.meta.url));

const CLIENT_PORT = 4173;
const STUB_PORT = 4174;

export default defineConfig({
	testDir: './tests/frontend',
	fullyParallel: false,
	workers: 1,
	retries: 0,
	timeout: 60_000,
	reporter: [['list']],
	use: {
		baseURL: `http://127.0.0.1:${CLIENT_PORT}`,
		trace: 'retain-on-failure'
	},
	webServer: [
		{
			// Native Node type stripping: no build step and no test-runner
			// dependency for the fake API.
			command: 'node --experimental-strip-types apps/client/test-server/main.ts',
			cwd: repoRoot,
			url: `http://127.0.0.1:${STUB_PORT}/__test/health`,
			env: { MBC_STUB_PORT: String(STUB_PORT) },
			reuseExistingServer: !process.env.CI,
			timeout: 30_000
		},
		{
			// Build then serve the production SPA with Vite preview, whose
			// same-origin `/mybadcoach` proxy targets the stub (see
			// apps/client/vite.config.ts).
			command: `npm run build:client && npm run preview -w apps/client -- --host 127.0.0.1 --port ${String(CLIENT_PORT)} --strictPort`,
			cwd: repoRoot,
			url: `http://127.0.0.1:${CLIENT_PORT}`,
			reuseExistingServer: !process.env.CI,
			timeout: 180_000
		}
	]
});
