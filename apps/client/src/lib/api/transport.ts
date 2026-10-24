/**
 * The client's single Connect transport (settled in ticket 01).
 *
 * - Same-origin `baseUrl: '/'`: dev and the frontend test seam both proxy
 *   `/mybadcoach` to a server, so the URL shape matches production.
 * - Every request carries `X-Requested-With: XMLHttpRequest`. Mutating RPCs
 *   require it (plus a same-origin `Origin`); it is harmless elsewhere.
 * - `credentials: 'include'` makes the `mbc_session` cookie explicit. Fetch's
 *   `same-origin` default happens to suffice today, but the transport must keep
 *   working if the client is ever served from another origin.
 * - Unauthenticated failures other than a failed login/register flow into one
 *   shared handler (registered by `session.svelte.ts`) that clears the session
 *   and redirects to `/login` with a "session expired" notice.
 */
import { Code, ConnectError, type Interceptor } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';

import { machineCodeOf } from '#lib/api/errors';

/** CSRF header required by the Go auth middleware on mutating RPCs. */
export const CSRF_HEADER = 'X-Requested-With';

/**
 * Auth RPCs whose `unauthenticated` response is a form outcome, not an expired
 * session: `Login`/`Register` carry `invalid_credentials`, and the startup
 * `Me` hydration is handled by the route guard.
 */
const FORM_OUTCOME_METHODS = new Set(['Login', 'Register', 'Me']);

let unauthenticatedHandler: ((error: ConnectError) => void) | undefined;

/** Registers the one handler shared by every service client. */
export function onUnauthenticated(handler: (error: ConnectError) => void): void {
	unauthenticatedHandler = handler;
}

const interceptors: Interceptor[] = [
	(next) => (request) => {
		request.header.set(CSRF_HEADER, 'XMLHttpRequest');
		return next(request);
	},
	(next) => async (request) => {
		try {
			return await next(request);
		} catch (error) {
			if (
				error instanceof ConnectError &&
				error.code === Code.Unauthenticated &&
				machineCodeOf(error) !== 'invalid_credentials' &&
				!FORM_OUTCOME_METHODS.has(request.method.name)
			) {
				unauthenticatedHandler?.(error);
			}
			throw error;
		}
	},
];

export const transport = createConnectTransport({
	baseUrl: '/',
	interceptors,
	fetch: (input, init) => globalThis.fetch(input, { ...init, credentials: 'include' }),
});
