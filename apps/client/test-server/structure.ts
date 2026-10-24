/**
 * Fake TeamService + SeasonService for the frontend-only Connect seam.
 *
 * Mirrors the structure-CRUD contract (`.scratch/backend-api/issues/05`):
 * - addressing is by nested slug (`team_slug` [+ `season_slug`]); ids never
 *   appear in requests;
 * - the caller must be a team member to read or mutate that team's structure;
 * - `CreateTeam` makes the caller `owner`; `UpdateTeam`, `DeleteTeam`,
 *   `DeleteSeason`, and membership management are owner-only;
 * - deletes are hard and cascade.
 *
 * Slugs are generated from the name and are stable across renames (ADR-0006):
 * team slugs are globally unique and season slugs are unique within a team.
 * Season rosters stay empty until players land in ticket 08.
 */
import { Code, type HandlerContext, type ServiceImpl } from '@connectrpc/connect';

import { connectError, requireUser } from './auth.ts';
import type { FakeSeason, FakeStore, FakeTeam } from './store.ts';
import { SeasonService } from '../../../packages/shared/src/gen/mybadcoach/seasons/v1/seasons_pb.ts';
import { TeamService } from '../../../packages/shared/src/gen/mybadcoach/teams/v1/teams_pb.ts';

export type TeamImplementation = Partial<ServiceImpl<typeof TeamService>>;
export type SeasonImplementation = Partial<ServiceImpl<typeof SeasonService>>;

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

function teamMessage(team: FakeTeam): FakeTeam {
	return { id: team.id, slug: team.slug, name: team.name };
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

		// Players arrive in ticket 08; the roster is a real, empty list until then.
		listSeasonRoster(request, context) {
			requireMembership(store, context, request.teamSlug);
			requireSeason(store, request.teamSlug, request.seasonSlug);
			return { players: [], total: 0n };
		}
	};
}
