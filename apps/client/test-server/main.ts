/**
 * Stub Connect server for the frontend-only integration seam.
 *
 * `docs/testing.md` requires the only test double at this seam to be a fake
 * API at the network boundary. This process is that fake: real HTTP, real
 * Connect framing and codecs, real cookies/CORS, backed by the in-memory
 * `FakeStore`. It is started by Playwright's `webServer` array and run with
 * Node's native TypeScript type stripping (no build step, no test runner
 * dependency):
 *
 *   node --experimental-strip-types apps/client/test-server/main.ts
 *
 * Auth, teams, and seasons are implemented for real against the fake store.
 * Event, match, and player services are registered with no methods so the
 * router answers them with Connect's `unimplemented` code; tickets 07-08
 * replace each empty object with an implementation backed by `FakeStore`. Add
 * new RPCs the same way: `router.service(<GeneratedService>, implementation)`.
 *
 * Test-only endpoints (not part of the Connect surface):
 *   GET  /__test/health  -> 200 "ok" (Playwright readiness)
 *   POST /__test/reset   -> 204 (clears the store between tests)
 *
 * Client reachability (decision recorded in ticket 04): the SPA's Connect
 * transport is same-origin (`baseUrl: '/'`, ticket 01). Vite proxies
 * `/mybadcoach` to this server in both dev (`server.proxy`) and the preview
 * used by the integration seam (`preview.proxy`), so production and tests use
 * the same URL shape. CORS with credentials is still enforced here so the
 * boundary behaves like the real cross-origin deployment (Cloudflare in
 * front) if the client is ever served from another origin.
 */
import { createServer } from 'node:http';
import type { IncomingMessage, ServerResponse } from 'node:http';

import { cors } from '@connectrpc/connect';
import { connectNodeAdapter } from '@connectrpc/connect-node';

import { createAuthImplementation } from './auth.ts';
import { FakeStore } from './store.ts';
import { createSeasonImplementation, createTeamImplementation } from './structure.ts';
import { AuthService } from '../../../packages/shared/src/gen/mybadcoach/auth/v1/auth_pb.ts';
import { EventService } from '../../../packages/shared/src/gen/mybadcoach/events/v1/events_pb.ts';
import { MatchService } from '../../../packages/shared/src/gen/mybadcoach/matches/v1/matches_pb.ts';
import { PlayerService } from '../../../packages/shared/src/gen/mybadcoach/players/v1/players_pb.ts';
import { SeasonService } from '../../../packages/shared/src/gen/mybadcoach/seasons/v1/seasons_pb.ts';
import { TeamService } from '../../../packages/shared/src/gen/mybadcoach/teams/v1/teams_pb.ts';

const HOST = process.env.MBC_STUB_HOST ?? '127.0.0.1';
const PORT = Number(process.env.MBC_STUB_PORT ?? '4174');

const CSRF_HEADER = 'X-Requested-With';

/** Mutating RPCs that require the CSRF header, matching the Go middleware. */
const MUTATING_PROCEDURES = new Set([
	'/mybadcoach.auth.v1.AuthService/Register',
	'/mybadcoach.auth.v1.AuthService/Login',
	'/mybadcoach.auth.v1.AuthService/Logout',
	'/mybadcoach.teams.v1.TeamService/CreateTeam',
	'/mybadcoach.teams.v1.TeamService/UpdateTeam',
	'/mybadcoach.teams.v1.TeamService/DeleteTeam',
	'/mybadcoach.teams.v1.TeamService/AddMembership',
	'/mybadcoach.teams.v1.TeamService/UpdateMembership',
	'/mybadcoach.teams.v1.TeamService/RemoveMembership',
	'/mybadcoach.seasons.v1.SeasonService/CreateSeason',
	'/mybadcoach.seasons.v1.SeasonService/UpdateSeason',
	'/mybadcoach.seasons.v1.SeasonService/DeleteSeason',
	'/mybadcoach.seasons.v1.SeasonService/AddSeasonRosterEntry',
	'/mybadcoach.seasons.v1.SeasonService/RemoveSeasonRosterEntry',
	'/mybadcoach.seasons.v1.SeasonService/CopySeasonRoster'
]);

const ALLOWED_HEADERS = [...cors.allowedHeaders, CSRF_HEADER].join(', ');
const ALLOWED_METHODS = [...cors.allowedMethods, 'OPTIONS'].join(', ');
const EXPOSED_HEADERS = [...cors.exposedHeaders, 'Connect-Protocol-Version'].join(', ');

const store = new FakeStore();

const connectHandler = connectNodeAdapter({
	routes(router) {
		router.service(AuthService, createAuthImplementation(store));
		router.service(TeamService, createTeamImplementation(store));
		router.service(SeasonService, createSeasonImplementation(store));

		// Deliberately empty: the router answers every method with
		// `unimplemented`. Tickets 07-08 fill these in over `FakeStore`.
		router.service(EventService, {});
		router.service(MatchService, {});
		router.service(PlayerService, {});
	}
});

function requestPath(request: IncomingMessage): string {
	return new URL(request.url ?? '/', 'http://stub.invalid').pathname;
}

function writeJson(
	response: ServerResponse,
	status: number,
	body: Record<string, unknown>
): void {
	response.writeHead(status, { 'Content-Type': 'application/json' });
	response.end(JSON.stringify(body));
}

/** Applies CORS headers. Returns true when the preflight was fully handled. */
function applyCors(request: IncomingMessage, response: ServerResponse): boolean {
	const origin = request.headers.origin;
	if (origin !== undefined) {
		// Reflect the caller's origin: credentials are allowed, so a wildcard
		// would be rejected by browsers.
		response.setHeader('Access-Control-Allow-Origin', origin);
		response.setHeader('Access-Control-Allow-Credentials', 'true');
		response.setHeader('Vary', 'Origin');
	}
	response.setHeader('Access-Control-Allow-Methods', ALLOWED_METHODS);
	response.setHeader('Access-Control-Allow-Headers', ALLOWED_HEADERS);
	response.setHeader('Access-Control-Expose-Headers', EXPOSED_HEADERS);
	response.setHeader('Access-Control-Max-Age', '600');

	if (request.method === 'OPTIONS') {
		response.writeHead(204);
		response.end();
		return true;
	}
	return false;
}

/**
 * Mirrors the Go CSRF middleware: a mutating RPC needs a non-empty
 * `X-Requested-With` and, when an Origin is present, a same-origin Origin.
 */
function csrfAllows(request: IncomingMessage): boolean {
	if (!MUTATING_PROCEDURES.has(requestPath(request))) {
		return true;
	}
	const requestedWith = request.headers['x-requested-with'];
	const hasCsrfHeader =
		typeof requestedWith === 'string' && requestedWith.length > 0;
	if (!hasCsrfHeader) {
		return false;
	}

	const origin = request.headers.origin;
	if (origin === undefined) {
		return true;
	}
	try {
		return new URL(origin).host === request.headers.host;
	} catch {
		return false;
	}
}

/** Handles the two test-only endpoints; returns true when it owns the request. */
function handleTestRoute(
	request: IncomingMessage,
	response: ServerResponse
): boolean {
	const path = requestPath(request);
	if (path === '/__test/health' && request.method === 'GET') {
		response.writeHead(200, { 'Content-Type': 'text/plain' });
		response.end('ok');
		return true;
	}
	if (path === '/__test/reset' && request.method === 'POST') {
		store.reset();
		response.writeHead(204);
		response.end();
		return true;
	}
	return false;
}

const server = createServer((request, response) => {
	if (handleTestRoute(request, response)) {
		return;
	}
	if (applyCors(request, response)) {
		return;
	}
	if (!csrfAllows(request)) {
		writeJson(response, 403, {
			code: 'permission_denied',
			message: 'request rejected by CSRF protection'
		});
		return;
	}
	connectHandler(request, response);
});

server.listen(PORT, HOST, () => {
	// Startup logging is the explicit exception to the no-console rule.
	console.log(`[stub] fake Connect API listening on http://${HOST}:${PORT}`);
});
