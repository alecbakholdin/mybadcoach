/**
 * In-memory fake store for the frontend-only Connect boundary.
 *
 * This is a real stateful implementation, not a mock: handlers read and write
 * it exactly as the Go server reads and writes SQLite, and tests reset it
 * between cases via `POST /__test/reset` (see `main.ts`).
 *
 * Password hashing is deliberately not bcrypt: bcrypt is a Go-side
 * implementation detail and the fake only needs credential equivalence. Node's
 * built-in scrypt keeps the double dependency-free.
 */
import { createHash, randomBytes, randomUUID, scryptSync, timingSafeEqual } from 'node:crypto';

/** Session cookie name, matching `server/internal/auth`. */
export const SESSION_COOKIE = 'mbc_session';

/** Absolute session lifetime (30 days), matching the Go contract. */
export const SESSION_LIFETIME_MS = 30 * 24 * 60 * 60 * 1000;

export interface FakeUser {
	id: string;
	email: string;
	name: string;
}

interface StoredUser extends FakeUser {
	passwordHash: string;
}

interface StoredSession {
	userId: string;
	expiresAt: number;
}

export interface FakeTeam {
	id: string;
	slug: string;
	name: string;
}

export interface FakeSeason {
	id: string;
	slug: string;
	name: string;
	startDate: string;
	endDate: string;
}

export interface FakeEvent {
	id: string;
	slug: string;
	name: string;
	startDate: string;
	endDate: string;
	location: string;
}

export interface FakeMatch {
	id: string;
	slug: string;
	opponentName: string;
	scheduledAt: string;
	pointCap: bigint;
}

export interface FakeTeamMembership {
	team: FakeTeam;
	role: string;
}

interface StoredTeam extends FakeTeam {
	/** User id → role (`owner` | `member`). */
	memberships: Map<string, string>;
}

function slugify(name: string): string {
	const base = name
		.normalize('NFKD')
		.replace(/[̀-ͯ]/g, '')
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
	return base === '' ? 'unnamed' : base;
}

/**
 * Nested-slug keys. Slugs only contain `[a-z0-9-]`, but the NUL separator keeps
 * the key unambiguous without relying on that.
 */
function seasonKey(teamSlug: string, seasonSlug: string): string {
	return `${teamSlug}\u0000${seasonSlug}`;
}

function eventKey(teamSlug: string, seasonSlug: string, eventSlug: string): string {
	return `${seasonKey(teamSlug, seasonSlug)}\u0000${eventSlug}`;
}

/** ADR-0006: collisions append a numeric suffix (`my-team`, `my-team-2`). */
function uniqueSlug(base: string, taken: (candidate: string) => boolean): string {
	if (!taken(base)) {
		return base;
	}
	let suffix = 2;
	while (taken(`${base}-${suffix}`)) {
		suffix += 1;
	}
	return `${base}-${suffix}`;
}

function hashToken(raw: string): string {
	return createHash('sha256').update(raw).digest('hex');
}

function hashPassword(password: string): string {
	const salt = randomBytes(16).toString('hex');
	const derived = scryptSync(password, salt, 32).toString('hex');
	return `${salt}:${derived}`;
}

function passwordMatches(password: string, stored: string): boolean {
	const [salt, expected] = stored.split(':');
	if (salt === undefined || expected === undefined) {
		return false;
	}
	const derived = scryptSync(password, salt, 32);
	const expectedBytes = Buffer.from(expected, 'hex');
	return derived.length === expectedBytes.length && timingSafeEqual(derived, expectedBytes);
}

/** A stateful in-memory stand-in for the server's SQLite-backed auth store. */
export class FakeStore {
	private readonly usersByEmail = new Map<string, StoredUser>();
	private readonly usersById = new Map<string, StoredUser>();
	private readonly sessions = new Map<string, StoredSession>();
	private readonly teamsBySlug = new Map<string, StoredTeam>();
	/** Insertion order is the stable list order; slugs are unique per team. */
	private readonly seasonsByTeamSlug = new Map<string, FakeSeason[]>();
	/** Keyed by `seasonKey(teamSlug, seasonSlug)`; slugs unique within a season. */
	private readonly eventsBySeason = new Map<string, FakeEvent[]>();
	/** Keyed by `eventKey(...)`; slugs unique within an event. */
	private readonly matchesByEvent = new Map<string, FakeMatch[]>();

	/** Clears every user, session, team, and descendants; called between tests. */
	reset(): void {
		this.usersByEmail.clear();
		this.usersById.clear();
		this.sessions.clear();
		this.teamsBySlug.clear();
		this.seasonsByTeamSlug.clear();
		this.eventsBySeason.clear();
		this.matchesByEvent.clear();
	}

	findUserByEmail(email: string): FakeUser | undefined {
		return this.usersByEmail.get(email);
	}

	createUser(email: string, name: string, password: string): FakeUser {
		const user: StoredUser = {
			id: randomUUID(),
			email,
			name,
			passwordHash: hashPassword(password)
		};
		this.usersByEmail.set(email, user);
		this.usersById.set(user.id, user);
		return user;
	}

	/** Returns the user for a valid credential pair, or undefined. */
	authenticate(email: string, password: string): FakeUser | undefined {
		const user = this.usersByEmail.get(email);
		if (user === undefined || !passwordMatches(password, user.passwordHash)) {
			return undefined;
		}
		return user;
	}

	/** Mints an opaque session token for a user. Only its hash is retained. */
	createSession(userId: string, now: number = Date.now()): { token: string; expiresAt: number } {
		const token = randomBytes(32).toString('base64url');
		const expiresAt = now + SESSION_LIFETIME_MS;
		this.sessions.set(hashToken(token), { userId, expiresAt });
		return { token, expiresAt };
	}

	/** Resolves a raw cookie token to its user, deleting it when expired. */
	findSessionUser(token: string, now: number = Date.now()): FakeUser | undefined {
		const hash = hashToken(token);
		const session = this.sessions.get(hash);
		if (session === undefined) {
			return undefined;
		}
		if (session.expiresAt <= now) {
			this.sessions.delete(hash);
			return undefined;
		}
		return this.usersById.get(session.userId);
	}

	deleteSession(token: string): void {
		this.sessions.delete(hashToken(token));
	}

	// ----- Teams (ADR-0006: globally unique slugs; callers become owners) -----

	createTeam(name: string, ownerId: string): FakeTeam {
		const team: StoredTeam = {
			id: randomUUID(),
			slug: uniqueSlug(slugify(name), (candidate) => this.teamsBySlug.has(candidate)),
			name,
			memberships: new Map([[ownerId, 'owner']])
		};
		this.teamsBySlug.set(team.slug, team);
		return team;
	}

	findTeamBySlug(slug: string): FakeTeam | undefined {
		return this.teamsBySlug.get(slug);
	}

	/** Returns the caller's role on the team, or undefined when not a member. */
	memberRole(teamSlug: string, userId: string): string | undefined {
		return this.teamsBySlug.get(teamSlug)?.memberships.get(userId);
	}

	listTeamsForUser(userId: string): FakeTeamMembership[] {
		const memberships: FakeTeamMembership[] = [];
		for (const team of this.teamsBySlug.values()) {
			const role = team.memberships.get(userId);
			if (role !== undefined) {
				memberships.push({ team, role });
			}
		}
		return memberships;
	}

	updateTeam(slug: string, name: string): FakeTeam | undefined {
		const team = this.teamsBySlug.get(slug);
		if (team === undefined) {
			return undefined;
		}
		team.name = name;
		return team;
	}

	/** Hard delete; cascades to the team's seasons, events, and matches. */
	deleteTeam(slug: string): boolean {
		if (!this.teamsBySlug.delete(slug)) {
			return false;
		}
		for (const season of this.seasonsByTeamSlug.get(slug) ?? []) {
			this.deleteSeasonDescendants(slug, season.slug);
		}
		this.seasonsByTeamSlug.delete(slug);
		return true;
	}

	/** Adds an existing user to a team; role defaults to `member`. */
	addMembership(teamSlug: string, userId: string, role = 'member'): boolean {
		const team = this.teamsBySlug.get(teamSlug);
		if (team === undefined) {
			return false;
		}
		team.memberships.set(userId, role);
		return true;
	}

	// ----- Seasons (slugs are unique within their team) -----

	createSeason(
		teamSlug: string,
		name: string,
		startDate: string,
		endDate: string
	): FakeSeason | undefined {
		if (!this.teamsBySlug.has(teamSlug)) {
			return undefined;
		}
		const teamSeasons = this.seasonsByTeamSlug.get(teamSlug) ?? [];
		const season: FakeSeason = {
			id: randomUUID(),
			slug: uniqueSlug(slugify(name), (candidate) =>
				teamSeasons.some((existing) => existing.slug === candidate)
			),
			name,
			startDate,
			endDate
		};
		teamSeasons.push(season);
		this.seasonsByTeamSlug.set(teamSlug, teamSeasons);
		return season;
	}

	findSeason(teamSlug: string, seasonSlug: string): FakeSeason | undefined {
		return this.seasonsByTeamSlug.get(teamSlug)?.find((season) => season.slug === seasonSlug);
	}

	listSeasons(teamSlug: string, offset: number, limit: number): FakeSeason[] {
		const seasons = this.seasonsByTeamSlug.get(teamSlug) ?? [];
		const start = Math.max(offset, 0);
		const end = limit > 0 ? start + limit : seasons.length;
		return seasons.slice(start, end);
	}

	countSeasons(teamSlug: string): number {
		return this.seasonsByTeamSlug.get(teamSlug)?.length ?? 0;
	}

	updateSeason(
		teamSlug: string,
		seasonSlug: string,
		patch: { name?: string; startDate?: string; endDate?: string }
	): FakeSeason | undefined {
		const season = this.findSeason(teamSlug, seasonSlug);
		if (season === undefined) {
			return undefined;
		}
		if (patch.name !== undefined) {
			season.name = patch.name;
		}
		if (patch.startDate !== undefined) {
			season.startDate = patch.startDate;
		}
		if (patch.endDate !== undefined) {
			season.endDate = patch.endDate;
		}
		return season;
	}

	/** Hard delete; cascades to the season's events and matches. */
	deleteSeason(teamSlug: string, seasonSlug: string): boolean {
		const seasons = this.seasonsByTeamSlug.get(teamSlug);
		if (seasons === undefined) {
			return false;
		}
		const index = seasons.findIndex((season) => season.slug === seasonSlug);
		if (index === -1) {
			return false;
		}
		seasons.splice(index, 1);
		this.deleteSeasonDescendants(teamSlug, seasonSlug);
		return true;
	}

	/** Drops every event (and match) beneath a season; assumes the season is gone. */
	private deleteSeasonDescendants(teamSlug: string, seasonSlug: string): void {
		const key = seasonKey(teamSlug, seasonSlug);
		for (const event of this.eventsBySeason.get(key) ?? []) {
			this.matchesByEvent.delete(eventKey(teamSlug, seasonSlug, event.slug));
		}
		this.eventsBySeason.delete(key);
	}

	// ----- Events (slugs are unique within their season) -----

	createEvent(
		teamSlug: string,
		seasonSlug: string,
		name: string,
		startDate: string,
		endDate: string,
		location: string
	): FakeEvent | undefined {
		if (this.findSeason(teamSlug, seasonSlug) === undefined) {
			return undefined;
		}
		const key = seasonKey(teamSlug, seasonSlug);
		const seasonEvents = this.eventsBySeason.get(key) ?? [];
		const event: FakeEvent = {
			id: randomUUID(),
			slug: uniqueSlug(slugify(name), (candidate) =>
				seasonEvents.some((existing) => existing.slug === candidate)
			),
			name,
			startDate,
			endDate,
			location
		};
		seasonEvents.push(event);
		this.eventsBySeason.set(key, seasonEvents);
		return event;
	}

	findEvent(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string
	): FakeEvent | undefined {
		return this.eventsBySeason
			.get(seasonKey(teamSlug, seasonSlug))
			?.find((event) => event.slug === eventSlug);
	}

	listEvents(teamSlug: string, seasonSlug: string, offset: number, limit: number): FakeEvent[] {
		const events = this.eventsBySeason.get(seasonKey(teamSlug, seasonSlug)) ?? [];
		const start = Math.max(offset, 0);
		const end = limit > 0 ? start + limit : events.length;
		return events.slice(start, end);
	}

	countEvents(teamSlug: string, seasonSlug: string): number {
		return this.eventsBySeason.get(seasonKey(teamSlug, seasonSlug))?.length ?? 0;
	}

	updateEvent(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string,
		patch: { name?: string; startDate?: string; endDate?: string; location?: string }
	): FakeEvent | undefined {
		const event = this.findEvent(teamSlug, seasonSlug, eventSlug);
		if (event === undefined) {
			return undefined;
		}
		if (patch.name !== undefined) {
			event.name = patch.name;
		}
		if (patch.startDate !== undefined) {
			event.startDate = patch.startDate;
		}
		if (patch.endDate !== undefined) {
			event.endDate = patch.endDate;
		}
		if (patch.location !== undefined) {
			event.location = patch.location;
		}
		return event;
	}

	/** Hard delete; cascades to the event's matches. */
	deleteEvent(teamSlug: string, seasonSlug: string, eventSlug: string): boolean {
		const key = seasonKey(teamSlug, seasonSlug);
		const events = this.eventsBySeason.get(key);
		if (events === undefined) {
			return false;
		}
		const index = events.findIndex((event) => event.slug === eventSlug);
		if (index === -1) {
			return false;
		}
		events.splice(index, 1);
		this.matchesByEvent.delete(eventKey(teamSlug, seasonSlug, eventSlug));
		return true;
	}

	// ----- Matches (slugs are unique within their event) -----

	createMatch(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string,
		opponentName: string,
		scheduledAt: string,
		pointCap: bigint
	): FakeMatch | undefined {
		if (this.findEvent(teamSlug, seasonSlug, eventSlug) === undefined) {
			return undefined;
		}
		const key = eventKey(teamSlug, seasonSlug, eventSlug);
		const eventMatches = this.matchesByEvent.get(key) ?? [];
		const match: FakeMatch = {
			id: randomUUID(),
			slug: uniqueSlug(slugify(opponentName), (candidate) =>
				eventMatches.some((existing) => existing.slug === candidate)
			),
			opponentName,
			scheduledAt,
			pointCap
		};
		eventMatches.push(match);
		this.matchesByEvent.set(key, eventMatches);
		return match;
	}

	findMatch(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string,
		matchSlug: string
	): FakeMatch | undefined {
		return this.matchesByEvent
			.get(eventKey(teamSlug, seasonSlug, eventSlug))
			?.find((match) => match.slug === matchSlug);
	}

	listMatches(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string,
		offset: number,
		limit: number
	): FakeMatch[] {
		const matches = this.matchesByEvent.get(eventKey(teamSlug, seasonSlug, eventSlug)) ?? [];
		const start = Math.max(offset, 0);
		const end = limit > 0 ? start + limit : matches.length;
		return matches.slice(start, end);
	}

	countMatches(teamSlug: string, seasonSlug: string, eventSlug: string): number {
		return this.matchesByEvent.get(eventKey(teamSlug, seasonSlug, eventSlug))?.length ?? 0;
	}

	updateMatch(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string,
		matchSlug: string,
		patch: { opponentName?: string; scheduledAt?: string; pointCap?: bigint }
	): FakeMatch | undefined {
		const match = this.findMatch(teamSlug, seasonSlug, eventSlug, matchSlug);
		if (match === undefined) {
			return undefined;
		}
		if (patch.opponentName !== undefined) {
			match.opponentName = patch.opponentName;
		}
		if (patch.scheduledAt !== undefined) {
			match.scheduledAt = patch.scheduledAt;
		}
		if (patch.pointCap !== undefined) {
			match.pointCap = patch.pointCap;
		}
		return match;
	}

	/** Hard delete. */
	deleteMatch(
		teamSlug: string,
		seasonSlug: string,
		eventSlug: string,
		matchSlug: string
	): boolean {
		const matches = this.matchesByEvent.get(eventKey(teamSlug, seasonSlug, eventSlug));
		if (matches === undefined) {
			return false;
		}
		const index = matches.findIndex((match) => match.slug === matchSlug);
		if (index === -1) {
			return false;
		}
		matches.splice(index, 1);
		return true;
	}
}
