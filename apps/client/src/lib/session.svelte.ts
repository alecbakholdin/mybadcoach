/**
 * Current-user session state as a Svelte 5 rune module (no store, no context).
 *
 * `currentUser` has three meaningful values:
 * - `undefined` — the initial `Me` hydration has not finished;
 * - `null` — the visitor is unauthenticated;
 * - a `User` — the visitor is signed in.
 *
 * `loadSession()` is memoized so root-layout load functions across navigations
 * perform at most one `Me` call. Login/register set the user directly and
 * logout clears it; a route guard can then make decisions synchronously.
 */
import { Code, ConnectError } from '@connectrpc/connect';

import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { page } from '$app/state';

import { auth, onUnauthenticated } from '#lib/api';
import type { User } from '$shared/gen/mybadcoach/auth/v1/auth_pb';

let currentUser = $state<User | null | undefined>(undefined);
let ready = $state(false);
let hydration: Promise<User | null> | undefined;
/** Guards against a late `Me` response overwriting an explicit clear/set. */
let generation = 0;

async function fetchCurrentUser(): Promise<User | null> {
	const requestGeneration = generation;
	try {
		const response = await auth.me({});
		const user = response.user ?? null;
		if (generation === requestGeneration) {
			currentUser = user;
		}
		return user;
	} catch (error) {
		if (error instanceof ConnectError && error.code === Code.Unauthenticated) {
			if (generation === requestGeneration) {
				currentUser = null;
			}
			return null;
		}
		throw error;
	} finally {
		if (generation === requestGeneration) {
			ready = true;
		}
	}
}

/** Hydrates the session once and returns the current user (or null). */
export function loadSession(): Promise<User | null> {
	if (currentUser !== undefined) {
		return Promise.resolve(currentUser);
	}
	hydration ??= fetchCurrentUser();
	return hydration;
}

/** Records a user after a successful login/register response. */
export function setUser(user: User): void {
	generation += 1;
	currentUser = user;
	ready = true;
	hydration = Promise.resolve(user);
}

/** Clears the local session (logout or expired-session handling). */
export function clear(): void {
	generation += 1;
	currentUser = null;
	ready = true;
	hydration = Promise.resolve(null);
}

/**
 * Best-effort server logout, always clearing local state: an unreachable
 * server must not strand the user in a signed-in shell.
 */
export async function logout(): Promise<void> {
	try {
		await auth.logout({});
	} catch {
		// Best effort: the local session is cleared below regardless.
	} finally {
		clear();
	}
}

// The one shared unauthenticated handler (ticket 01): a 401 from any service
// other than login/register/startup-Me clears the session and sends the user
// back to the login screen with a notice.
onUnauthenticated(() => {
	clear();
	if (page.url.pathname !== '/login') {
		void goto(resolve('/login?notice=session-expired'));
	}
});

export const session = {
	get currentUser(): User | null | undefined {
		return currentUser;
	},
	get ready(): boolean {
		return ready;
	},
	load: loadSession,
	setUser,
	clear,
	logout
};
