/**
 * Fake structure services for the frontend-only Connect seam: TeamService,
 * SeasonService, EventService, MatchService, and PlayerService.
 *
 * Mirrors the structure-CRUD contract (`.scratch/backend-api/issues/05`):
 * - addressing is by nested slug (`team_slug` [+ `season_slug` [+ `event_slug`]]);
 *   ids never appear in requests;
 * - the caller must be a team member to read or mutate that team's structure;
 * - `CreateTeam` makes the caller `owner`; `UpdateTeam`, `DeleteTeam`,
 *   `DeleteSeason`, `DeletePlayer`, and membership management are owner-only;
 * - `DeleteEvent` and `DeleteMatch` are member-gated (any member), as are roster
 *   adds/removes, matching the contract;
 * - deletes are hard and cascade.
 *
 * Slugs are generated from the name and are stable across renames (ADR-0006):
 * team slugs are globally unique, season and player slugs are unique within a
 * team, event slugs within a season, and match slugs (from `opponent_name`)
 * within an event.
 *
 * Roster adds follow the server's `INSERT OR IGNORE`: adding an already
 * registered player is an idempotent no-op that still returns the player.
 */
import { Code, type HandlerContext, type ServiceImpl } from '@connectrpc/connect';

import { connectError, requireUser } from './auth.ts';
import type {
	FakeEvent,
	FakeMatch,
	FakePlayer,
	FakeSeason,
	FakeStore,
	FakeTeam
} from './store.ts';
import { EventService } from '../../../packages/shared/src/gen/mybadcoach/events/v1/events_pb.ts';
import { MatchService } from '../../../packages/shared/src/gen/mybadcoach/matches/v1/matches_pb.ts';
import { PlayerService } from '../../../packages/shared/src/gen/mybadcoach/players/v1/players_pb.ts';
import { SeasonService } from '../../../packages/shared/src/gen/mybadcoach/seasons/v1/seasons_pb.ts';
import { TeamService } from '../../../packages/shared/src/gen/mybadcoach/teams/v1/teams_pb.ts';

export type TeamImplementation = Partial<ServiceImpl<typeof TeamService>>;
export type SeasonImplementation = Partial<ServiceImpl<typeof SeasonService>>;
export type EventImplementation = Partial<ServiceImpl<typeof EventService>>;
export type MatchImplementation = Partial<ServiceImpl<typeof MatchService>>;
export type PlayerImplementation = Partial<ServiceImpl<typeof PlayerService>>;

function notFound(message: string): Error {
	return connectError(Code.NotFound, 'not_found', message);
}

function permissionDenied(message: string): Error {
	return connectError(Code.PermissionDenied, 'permission_denied', message);
}

function invalidField(field: string, description: string): Error {
	return connectError(Code.InvalidArgument, 'invalid_argument', 'validation failed', [
		{ field, description }
	]);
}

function requiredName(value: string, field: string): string {
	const name = value.trim();
	if (name === '') {
		throw invalidField(field, 'must not be empty');
	}
	return name;
}

/** Mirrors the server: at least one non-empty name, each trimmed, order kept. */
function normalizeNames(names: string[]): string[] {
	if (names.length === 0) {
		throw invalidField('names', 'must contain at least one name');
	}
	return names.map((name) => {
		const trimmed = name.trim();
		if (trimmed === '') {
			throw invalidField('names', 'must not contain empty names');
		}
		return trimmed;
	});
}

interface Membership {
	userId: string;
	role: string;
}

/** Any membership grants reads + structure mutations (contract ticket 05). */
function requireMembership(
	store: FakeStore,
	context: HandlerContext,
	teamSlug: string
): Membership {
	if (teamSlug === '') {
		throw invalidField('team_slug', 'must not be empty');
	}
	const user = requireUser(store, context);
	if (store.findTeamBySlug(teamSlug) === undefined) {
		throw notFound('team not found');
	}
	const role = store.memberRole(teamSlug, user.id);
	if (role === undefined) {
		throw permissionDenied('caller is not a member of this team');
	}
	return { userId: user.id, role };
}

function requireOwner(store: FakeStore, context: HandlerContext, teamSlug: string): Membership {
	const membership = requireMembership(store, context, teamSlug);
	if (membership.role !== 'owner') {
		throw permissionDenied('only a team owner may perform this action');
	}
	return membership;
}

function requireSeason(store: FakeStore, teamSlug: string, seasonSlug: string): FakeSeason {
	const season = store.findSeason(teamSlug, seasonSlug);
	if (season === undefined) {
		throw notFound('season not found');
	}
	return season;
}

function requireEvent(
	store: FakeStore,
	teamSlug: string,
	seasonSlug: string,
	eventSlug: string
): FakeEvent {
	const event = store.findEvent(teamSlug, seasonSlug, eventSlug);
	if (event === undefined) {
		throw notFound('event not found');
	}
	return event;
}

function requireMatch(
	store: FakeStore,
	teamSlug: string,
	seasonSlug: string,
	eventSlug: string,
	matchSlug: string
): FakeMatch {
	const match = store.findMatch(teamSlug, seasonSlug, eventSlug, matchSlug);
	if (match === undefined) {
		throw notFound('match not found');
	}
	return match;
}

function requirePlayer(store: FakeStore, teamSlug: string, playerSlug: string): FakePlayer {
	const player = store.findPlayer(teamSlug, playerSlug);
	if (player === undefined) {
		throw notFound('player not found');
	}
	return player;
}

function teamMessage(team: FakeTeam): FakeTeam {
	return { id: team.id, slug: team.slug, name: team.name };
}

function playerMessage(player: FakePlayer): FakePlayer {
	return { id: player.id, slug: player.slug, names: [...player.names] };
}

/** Builds the fake TeamService implementation over a store. */
export function createTeamImplementation(store: FakeStore): TeamImplementation {
	return {
		createTeam(request, context) {
			const user = requireUser(store, context);
			const team = store.createTeam(requiredName(request.name, 'name'), user.id);
			return { team: teamMessage(team) };
		},

		getTeam(request, context) {
			requireMembership(store, context, request.teamSlug);
			const team = store.findTeamBySlug(request.teamSlug);
			if (team === undefined) {
				throw notFound('team not found');
			}
			return { team: teamMessage(team) };
		},

		updateTeam(request, context) {
			requireOwner(store, context, request.teamSlug);
			const name =
				request.name === undefined ? undefined : requiredName(request.name, 'name');
			const team =
				name === undefined
					? store.findTeamBySlug(request.teamSlug)
					: store.updateTeam(request.teamSlug, name);
			if (team === undefined) {
				throw notFound('team not found');
			}
			return { team: teamMessage(team) };
		},

		deleteTeam(request, context) {
			requireOwner(store, context, request.teamSlug);
			if (!store.deleteTeam(request.teamSlug)) {
				throw notFound('team not found');
			}
			return {};
		},

		listMyTeams(_request, context) {
			const user = requireUser(store, context);
			return {
				memberships: store.listTeamsForUser(user.id).map((membership) => ({
					team: teamMessage(membership.team),
					role: membership.role
				}))
			};
		},

		// Implemented so the owner-gating seam can seed a non-owner member;
		// there is no membership-management screen in this ticket.
		addMembership(request, context) {
			requireOwner(store, context, request.teamSlug);
			const email = request.email.trim().toLowerCase();
			if (email === '') {
				throw invalidField('email', 'must not be empty');
			}
			const user = store.findUserByEmail(email);
			if (user === undefined) {
				throw connectError(Code.NotFound, 'user_not_found', 'no user with that email');
			}
			store.addMembership(request.teamSlug, user.id);
			return {
				membership: { user: { id: user.id, email: user.email, name: user.name }, role: 'member' }
			};
		}
	};
}

/** Builds the fake SeasonService implementation over a store. */
export function createSeasonImplementation(store: FakeStore): SeasonImplementation {
	return {
		createSeason(request, context) {
			requireMembership(store, context, request.teamSlug);
			const season = store.createSeason(
				request.teamSlug,
				requiredName(request.name, 'name'),
				request.startDate ?? '',
				request.endDate ?? ''
			);
			if (season === undefined) {
				throw notFound('team not found');
			}
			return { season };
		},

		getSeason(request, context) {
			requireMembership(store, context, request.teamSlug);
			return { season: requireSeason(store, request.teamSlug, request.seasonSlug) };
		},

		updateSeason(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);

			const patch: { name?: string; startDate?: string; endDate?: string } = {};
			if (request.name !== undefined) {
				patch.name = requiredName(request.name, 'name');
			}
			if (request.startDate !== undefined) {
				patch.startDate = request.startDate;
			}
			if (request.endDate !== undefined) {
				patch.endDate = request.endDate;
			}
			const season = store.updateSeason(request.teamSlug, request.seasonSlug, patch);
			if (season === undefined) {
				throw notFound('season not found');
			}
			return { season };
		},

		listSeasons(request, context) {
			requireMembership(store, context, request.teamSlug);
			return {
				seasons: store.listSeasons(request.teamSlug, request.offset, request.limit),
				total: BigInt(store.countSeasons(request.teamSlug))
			};
		},

		deleteSeason(request, context) {
			requireOwner(store, context, request.teamSlug);
			if (!store.deleteSeason(request.teamSlug, request.seasonSlug)) {
				throw notFound('season not found');
			}
			return {};
		},

		// Member-gated roster mutations (structure contract ticket 05). A
		// duplicate add is a no-op that still returns the player, mirroring the
		// server's INSERT OR IGNORE.
		addSeasonRosterEntry(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			const player = requirePlayer(store, request.teamSlug, request.playerSlug);
			store.addSeasonRosterEntry(request.teamSlug, request.seasonSlug, player.id);
			return { player: playerMessage(player) };
		},

		removeSeasonRosterEntry(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			const player = requirePlayer(store, request.teamSlug, request.playerSlug);
			store.removeSeasonRosterEntry(request.teamSlug, request.seasonSlug, player.id);
			return {};
		},

		listSeasonRoster(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			return {
				players: store
					.listSeasonRoster(
						request.teamSlug,
						request.seasonSlug,
						request.offset,
						request.limit
					)
					.map(playerMessage),
				total: BigInt(store.countSeasonRoster(request.teamSlug, request.seasonSlug))
			};
		}

		// CopySeasonRoster stays unimplemented: it is out of this ticket's UI scope.
	};
}

/** Builds the fake EventService implementation over a store. */
export function createEventImplementation(store: FakeStore): EventImplementation {
	return {
		createEvent(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			const event = store.createEvent(
				request.teamSlug,
				request.seasonSlug,
				requiredName(request.name, 'name'),
				request.startDate ?? '',
				request.endDate ?? '',
				request.location ?? ''
			);
			if (event === undefined) {
				throw notFound('season not found');
			}
			return { event };
		},

		getEvent(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			return {
				event: requireEvent(
					store,
					request.teamSlug,
					request.seasonSlug,
					request.eventSlug
				)
			};
		},

		updateEvent(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);

			const patch: { name?: string; startDate?: string; endDate?: string; location?: string } =
				{};
			if (request.name !== undefined) {
				patch.name = requiredName(request.name, 'name');
			}
			if (request.startDate !== undefined) {
				patch.startDate = request.startDate;
			}
			if (request.endDate !== undefined) {
				patch.endDate = request.endDate;
			}
			if (request.location !== undefined) {
				patch.location = request.location;
			}
			const event = store.updateEvent(
				request.teamSlug,
				request.seasonSlug,
				request.eventSlug,
				patch
			);
			if (event === undefined) {
				throw notFound('event not found');
			}
			return { event };
		},

		listEvents(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			return {
				events: store.listEvents(
					request.teamSlug,
					request.seasonSlug,
					request.offset,
					request.limit
				),
				total: BigInt(store.countEvents(request.teamSlug, request.seasonSlug))
			};
		},

		// Member-gated, not owner-only (structure contract ticket 05).
		deleteEvent(request, context) {
			requireMembership(store, context, request.teamSlug);
			if (
				!store.deleteEvent(request.teamSlug, request.seasonSlug, request.eventSlug)
			) {
				throw notFound('event not found');
			}
			return {};
		},

		// Member-gated roster mutations; a duplicate add is an idempotent no-op.
		addEventRosterEntry(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			const player = requirePlayer(store, request.teamSlug, request.playerSlug);
			store.addEventRosterEntry(
				request.teamSlug,
				request.seasonSlug,
				request.eventSlug,
				player.id
			);
			return { player: playerMessage(player) };
		},

		removeEventRosterEntry(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			const player = requirePlayer(store, request.teamSlug, request.playerSlug);
			store.removeEventRosterEntry(
				request.teamSlug,
				request.seasonSlug,
				request.eventSlug,
				player.id
			);
			return {};
		},

		listEventRoster(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			return {
				players: store
					.listEventRoster(
						request.teamSlug,
						request.seasonSlug,
						request.eventSlug,
						request.offset,
						request.limit
					)
					.map(playerMessage),
				total: BigInt(
					store.countEventRoster(
						request.teamSlug,
						request.seasonSlug,
						request.eventSlug
					)
				)
			};
		}
	};
}

/** Builds the fake MatchService implementation over a store. */
export function createMatchImplementation(store: FakeStore): MatchImplementation {
	return {
		createMatch(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			const match = store.createMatch(
				request.teamSlug,
				request.seasonSlug,
				request.eventSlug,
				requiredName(request.opponentName, 'opponent_name'),
				request.scheduledAt ?? '',
				request.pointCap ?? 0n
			);
			if (match === undefined) {
				throw notFound('event not found');
			}
			return { match };
		},

		getMatch(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			return {
				match: requireMatch(
					store,
					request.teamSlug,
					request.seasonSlug,
					request.eventSlug,
					request.matchSlug
				)
			};
		},

		updateMatch(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			requireMatch(
				store,
				request.teamSlug,
				request.seasonSlug,
				request.eventSlug,
				request.matchSlug
			);

			const patch: { opponentName?: string; scheduledAt?: string; pointCap?: bigint } = {};
			if (request.opponentName !== undefined) {
				patch.opponentName = requiredName(request.opponentName, 'opponent_name');
			}
			if (request.scheduledAt !== undefined) {
				patch.scheduledAt = request.scheduledAt;
			}
			if (request.pointCap !== undefined) {
				patch.pointCap = request.pointCap;
			}
			const match = store.updateMatch(
				request.teamSlug,
				request.seasonSlug,
				request.eventSlug,
				request.matchSlug,
				patch
			);
			if (match === undefined) {
				throw notFound('match not found');
			}
			return { match };
		},

		listMatches(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireEvent(store, request.teamSlug, request.seasonSlug, request.eventSlug);
			return {
				matches: store.listMatches(
					request.teamSlug,
					request.seasonSlug,
					request.eventSlug,
					request.offset,
					request.limit
				),
				total: BigInt(
					store.countMatches(
						request.teamSlug,
						request.seasonSlug,
						request.eventSlug
					)
				)
			};
		},

		// Member-gated, not owner-only (structure contract ticket 05).
		deleteMatch(request, context) {
			requireMembership(store, context, request.teamSlug);
			if (
				!store.deleteMatch(
					request.teamSlug,
					request.seasonSlug,
					request.eventSlug,
					request.matchSlug
				)
			) {
				throw notFound('match not found');
			}
			return {};
		},

		// Gameplay RPCs (Append/List/State/Stats) are out of scope for this map.
	};
}

/** Builds the fake PlayerService implementation over a store. */
export function createPlayerImplementation(store: FakeStore): PlayerImplementation {
	return {
		createPlayer(request, context) {
			requireMembership(store, context, request.teamSlug);
			const player = store.createPlayer(request.teamSlug, normalizeNames(request.names));
			if (player === undefined) {
				throw notFound('team not found');
			}
			return { player: playerMessage(player) };
		},

		getPlayer(request, context) {
			requireMembership(store, context, request.teamSlug);
			return {
				player: playerMessage(
					requirePlayer(store, request.teamSlug, request.playerSlug)
				)
			};
		},

		// Empty names means "leave unchanged" (proto3 repeated fields have no
		// presence); a non-empty list replaces the whole name list.
		updatePlayer(request, context) {
			requireMembership(store, context, request.teamSlug);
			const existing = requirePlayer(store, request.teamSlug, request.playerSlug);
			const player =
				request.names.length === 0
					? existing
					: store.updatePlayerNames(
							request.teamSlug,
							request.playerSlug,
							normalizeNames(request.names)
						);
			if (player === undefined) {
				throw notFound('player not found');
			}
			return { player: playerMessage(player) };
		},

		// Owner-only hard delete; roster rows cascade, the match-event log does not.
		deletePlayer(request, context) {
			requireOwner(store, context, request.teamSlug);
			requirePlayer(store, request.teamSlug, request.playerSlug);
			if (!store.deletePlayer(request.teamSlug, request.playerSlug)) {
				throw notFound('player not found');
			}
			return {};
		},

		listPlayers(request, context) {
			requireMembership(store, context, request.teamSlug);
			return {
				players: store
					.listPlayers(request.teamSlug, request.offset, request.limit)
					.map(playerMessage),
				total: BigInt(store.countPlayers(request.teamSlug))
			};
		}
	};
}
