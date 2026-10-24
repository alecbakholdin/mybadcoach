import {
	events as eventClient,
	players as playerClient,
	seasons as seasonClient,
	teams
} from '#lib/api';
import { undefinedIfNotFound } from '#lib/api/errors';
import { session } from '#lib/session.svelte';
import { teamRoleOf, type SeasonDetailData } from '#lib/structure-data';

// The membership role gates `DeleteSeason` in the UI; the season, its events,
// and its roster load together because this screen always shows all three.
// A stale link maps to the screen's own not-found state instead of an error
// page, matching the event and match loaders.
export const load = async ({
	params
}: {
	params: { team_slug: string; season_slug: string };
}): Promise<SeasonDetailData> => {
	const { team_slug: teamSlug, season_slug: seasonSlug } = params;
	if ((await session.load()) === null) {
		return {
			teamSlug,
			seasonSlug,
			teamName: '',
			role: 'member',
			season: undefined,
			events: [],
			eventsTotal: 0,
			roster: [],
			rosterTotal: 0,
			teamPlayers: []
		};
	}
	const myTeams = await teams.listMyTeams({});
	const membership = myTeams.memberships.find((entry) => entry.team?.slug === teamSlug);
	if (membership?.team === undefined) {
		return {
			teamSlug,
			seasonSlug,
			teamName: '',
			role: 'member',
			season: undefined,
			events: [],
			eventsTotal: 0,
			roster: [],
			rosterTotal: 0,
			teamPlayers: []
		};
	}

	const [seasonResponse, eventResponse, rosterResponse, playerResponse] = await Promise.all([
		undefinedIfNotFound(seasonClient.getSeason({ teamSlug, seasonSlug })),
		undefinedIfNotFound(eventClient.listEvents({ teamSlug, seasonSlug, offset: 0, limit: 100 })),
		undefinedIfNotFound(
			seasonClient.listSeasonRoster({ teamSlug, seasonSlug, offset: 0, limit: 100 })
		),
		playerClient.listPlayers({ teamSlug, offset: 0, limit: 100 })
	]);
	return {
		teamSlug,
		seasonSlug,
		teamName: membership.team.name,
		role: teamRoleOf(membership.role),
		season: seasonResponse?.season,
		events: eventResponse?.events ?? [],
		eventsTotal: Number(eventResponse?.total ?? 0n),
		roster: rosterResponse?.players ?? [],
		rosterTotal: Number(rosterResponse?.total ?? 0n),
		teamPlayers: playerResponse.players
	};
};
