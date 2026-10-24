import { redirect } from '@sveltejs/kit';

import { session } from '#lib/session.svelte';

// Disable server-side rendering: the client is a static SPA served by the Go
// server, and all rendering happens in the browser.
export const ssr = false;

/** Routes that do not require an authenticated session. */
const PUBLIC_ROUTES = new Set(['/', '/login', '/register']);

/**
 * Hydrates the session once per app start, then gates every non-public route.
 * The splash of `Me` runs inside this load, so no protected screen renders
 * before the session is known.
 */
export async function load(event: { url: URL }): Promise<void> {
	const user = await session.load();
	if (user === null && !PUBLIC_ROUTES.has(event.url.pathname)) {
		redirect(303, '/login');
	}
}
