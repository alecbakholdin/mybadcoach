import { seasons as seasonClient, teams } from '#lib/api';
import { session } from '#lib/session.svelte';
import type { SeasonDetailData } from '#lib/structure-data';

// The membership role gates `DeleteSeason` in the UI; the season and its
// roster load together because this screen always shows both.
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
			role: '',
			season: undefined,
			roster: [],
			rosterTotal: 0
		};
	}
	const myTeams = await teams.listMyTeams({});
	const membership = myTeams.memberships.find((entry) => entry.team?.slug === teamSlug);
	if (membership?.team === undefined) {
		return {
			teamSlug,
			seasonSlug,
			teamName: '',
			role: '',
			season: undefined,
			roster: [],
			rosterTotal: 0
		};
	}

	const [seasonResponse, rosterResponse] = await Promise.all([
		seasonClient.getSeason({ teamSlug, seasonSlug }),
		seasonClient.listSeasonRoster({ teamSlug, seasonSlug, offset: 0, limit: 100 })
	]);
	return {
		teamSlug,
		seasonSlug,
		teamName: membership.team.name,
		role: membership.role,
		season: seasonResponse.season,
		roster: rosterResponse.players,
		rosterTotal: Number(rosterResponse.total)
	};
};
