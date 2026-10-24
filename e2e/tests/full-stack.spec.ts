import { expect, test, type APIRequestContext } from '@playwright/test';

// The compose network addresses the services by name. Bare-metal runs can point
// these at localhost instead.
const SERVER_URL = process.env.MBC_SERVER_URL ?? 'http://server:8080';
const CLIENT_URL = process.env.MBC_CLIENT_URL ?? 'http://client:3000';

// Connect procedure paths are `/<proto package>.<Service>/<Method>`.
const AUTH = 'mybadcoach.auth.v1.AuthService';
const TEAMS = 'mybadcoach.teams.v1.TeamService';
const SEASONS = 'mybadcoach.seasons.v1.SeasonService';
const EVENTS = 'mybadcoach.events.v1.EventService';
const PLAYERS = 'mybadcoach.players.v1.PlayerService';
const MATCHES = 'mybadcoach.matches.v1.MatchService';

// Connect's JSON codec is protojson: enum values are their full proto names
// (not the fixtures' short names), oneof members are camelCase field names,
// and 64-bit integers are JSON strings. The same shape is used in requests and
// responses, so the response types below keep int64 fields as strings.
const enumPrefix = 'MATCH_EVENT_TYPE_';

interface UserResponse {
	user: { id: string; email: string; name: string };
}

interface TeamResponse {
	team: { id: string; slug: string; name: string };
}

interface SeasonResponse {
	season: { id: string; slug: string; name: string };
}

interface EventResponse {
	event: { id: string; slug: string; name: string };
}

interface MatchResponse {
	match: { id: string; slug: string; opponentName: string };
}

interface PlayerResponse {
	player: { id: string; slug: string; names: string[] };
}

interface AppendEventsResponse {
	acceptedIds: string[];
	// protojson omits empty repeated fields.
	skippedDuplicateIds?: string[];
}

interface WireEvent {
	id: string;
	matchId: string;
	pointNumber: string;
	effectiveTs: string;
	type: string;
}

interface ListEventsResponse {
	events: WireEvent[];
	total: string;
}

interface StateResponse {
	points: Array<{
		pointNumber: string;
		state: {
			view: string;
			initialRole: string;
			lineup: string[];
			playerStats: Record<string, Record<string, string>>;
		};
	}>;
}

interface StatsResponse {
	playerStats: Array<{
		playerId: string;
		base: Record<string, string>;
		derived: Record<string, string>;
	}>;
}

async function call<T>(
	api: APIRequestContext,
	procedure: string,
	body: unknown
): Promise<T> {
	const response = await api.post(`/${procedure}`, { data: body });
	const text = await response.text();
	if (!response.ok()) {
		throw new Error(`${procedure} failed: HTTP ${response.status()} ${text}`);
	}
	return JSON.parse(text) as T;
}

async function sessionCookieValue(api: APIRequestContext): Promise<string | undefined> {
	const state = await api.storageState();
	return state.cookies.find((cookie) => cookie.name === 'mbc_session')?.value;
}

test('full-stack smoke: auth, structure CRUD, live gameplay, folded state, and stats', async ({
	page,
	playwright
}) => {
	const api = await playwright.request.newContext({
		baseURL: SERVER_URL,
		extraHTTPHeaders: {
			// Every mutating Connect procedure is CSRF-gated on this header.
			'X-Requested-With': 'XMLHttpRequest'
		}
	});

	try {
		// Wait for the compose stack even when `docker compose run` starts the
		// dependencies without waiting for their health checks.
		await expect
			.poll(async () => {
				try {
					return (await api.get('/')).status();
				} catch {
					return 0;
				}
			}, { timeout: 30_000 })
			.toBeGreaterThan(0);
		await expect
			.poll(async () => {
				try {
					return (await api.get(CLIENT_URL)).status();
				} catch {
					return 0;
				}
			}, { timeout: 30_000 })
			.toBe(200);

		const email = `smoke-${Date.now()}-${Math.random().toString(36).slice(2)}@mybadcoach.dev`;
		const password = 'password123';

		// Register a fresh user and assert the server minted a session cookie.
		const registered = await call<UserResponse>(api, `${AUTH}/Register`, {
			email,
			password,
			name: 'Smoke Tester'
		});
		expect(registered.user.email).toBe(email);
		expect(await sessionCookieValue(api)).toBeTruthy();

		// Login mints a new session; the cookie must be present afterwards.
		const loggedIn = await call<UserResponse>(api, `${AUTH}/Login`, { email, password });
		expect(loggedIn.user.id).toBe(registered.user.id);
		expect(await sessionCookieValue(api)).toBeTruthy();

		const { team } = await call<TeamResponse>(api, `${TEAMS}/CreateTeam`, {
			name: `Smoke Team ${Date.now()}`
		});
		expect(team.slug).toBeTruthy();

		const { season } = await call<SeasonResponse>(api, `${SEASONS}/CreateSeason`, {
			teamSlug: team.slug,
			name: 'Spring Season'
		});
		expect(season.slug).toBeTruthy();

		const { event } = await call<EventResponse>(api, `${EVENTS}/CreateEvent`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			name: 'Season Opener'
		});
		expect(event.slug).toBeTruthy();

		const { match } = await call<MatchResponse>(api, `${MATCHES}/CreateMatch`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			eventSlug: event.slug,
			opponentName: 'Rival Club'
		});
		expect(match.slug).toBeTruthy();

		const { player: handler } = await call<PlayerResponse>(api, `${PLAYERS}/CreatePlayer`, {
			teamSlug: team.slug,
			names: ['Alex Handler']
		});
		const { player: cutter } = await call<PlayerResponse>(api, `${PLAYERS}/CreatePlayer`, {
			teamSlug: team.slug,
			names: ['Sam Cutter']
		});

		// Add the handler to the season roster; the returned player is the
		// active roster member.
		const rosterEntry = await call<PlayerResponse>(api, `${SEASONS}/AddSeasonRosterEntry`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			playerSlug: handler.slug
		});
		expect(rosterEntry.player.id).toBe(handler.id);

		// Enqueue live taps for point 1: a lineup, two touches (the second
		// completes the first throw), and a goal with an assist.
		const eventId = (suffix: string) => `smoke-${match.id}-${suffix}`;
		const appended = await call<AppendEventsResponse>(api, `${MATCHES}/AppendMatchEvents`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			eventSlug: event.slug,
			matchSlug: match.slug,
			events: [
				{
					id: eventId('lineup'),
					pointNumber: 1,
					effectiveTs: 1_000,
					type: `${enumPrefix}SET_LINEUP`,
					setLineup: { playerIds: [handler.id, cutter.id] }
				},
				{
					id: eventId('touch-handler'),
					pointNumber: 1,
					effectiveTs: 2_000,
					type: `${enumPrefix}TOUCH`,
					touch: { playerId: handler.id }
				},
				{
					id: eventId('touch-cutter'),
					pointNumber: 1,
					effectiveTs: 3_000,
					type: `${enumPrefix}TOUCH`,
					touch: { playerId: cutter.id }
				},
				{
					id: eventId('score'),
					pointNumber: 1,
					effectiveTs: 4_000,
					type: `${enumPrefix}SCORE`,
					score: { playerId: cutter.id, assistPlayerId: handler.id }
				}
			]
		});
		expect(appended.acceptedIds).toHaveLength(4);
		expect(appended.skippedDuplicateIds ?? []).toHaveLength(0);

		// The event log round-trips in order and preserves the oneof payloads.
		const listed = await call<ListEventsResponse>(api, `${MATCHES}/ListMatchEvents`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			eventSlug: event.slug,
			matchSlug: match.slug
		});
		expect(listed.total).toBe('4');
		expect(listed.events.map((stored) => stored.id)).toEqual([
			eventId('lineup'),
			eventId('touch-handler'),
			eventId('touch-cutter'),
			eventId('score')
		]);
		expect(listed.events.at(-1)?.type).toBe(`${enumPrefix}SCORE`);

		// The server's Go reducer folds the log into a scored point.
		const state = await call<StateResponse>(api, `${MATCHES}/GetMatchState`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			eventSlug: event.slug,
			matchSlug: match.slug
		});
		expect(state.points).toHaveLength(1);
		const point = state.points[0];
		expect(point.pointNumber).toBe('1');
		expect(point.state.view).toBe('VIEW_SCORED');
		expect(point.state.lineup).toEqual([handler.id, cutter.id]);
		expect(Object.keys(point.state.playerStats).sort()).toEqual([handler.id, cutter.id].sort());
		expect(point.state.playerStats[handler.id]).toMatchObject({
			touches: '1',
			completions: '1',
			assists: '1'
		});
		expect(point.state.playerStats[cutter.id]).toMatchObject({
			touches: '1',
			goals: '1'
		});

		// Aggregated stats are derived from the same folded points.
		const stats = await call<StatsResponse>(api, `${MATCHES}/GetMatchStats`, {
			teamSlug: team.slug,
			seasonSlug: season.slug,
			eventSlug: event.slug,
			matchSlug: match.slug
		});
		expect(stats.playerStats.length).toBeGreaterThanOrEqual(2);
		const handlerStats = stats.playerStats.find((entry) => entry.playerId === handler.id);
		const cutterStats = stats.playerStats.find((entry) => entry.playerId === cutter.id);
		expect(handlerStats?.base).toMatchObject({ touches: '1', completions: '1', assists: '1' });
		expect(handlerStats?.derived).toMatchObject({ receptions: '1' });
		expect(cutterStats?.base).toMatchObject({ touches: '1', goals: '1' });
		expect(cutterStats?.derived).toMatchObject({ receptions: '1', plusMinus: '1' });

		// The real SvelteKit client builds and serves alongside the server.
		const response = await page.goto(CLIENT_URL);
		expect(response?.status()).toBe(200);
		await expect(page.getByRole('heading', { name: 'My Bad, Coach' })).toBeVisible();
	} finally {
		await api.dispose();
	}
});
