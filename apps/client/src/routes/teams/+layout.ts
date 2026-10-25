import { seasons as seasonClient, teams } from '#lib/api';
import { session } from '#lib/session.svelte';
import type { TeamsShellData } from '#lib/structure-data';

// The master–detail shell's own load: the rail needs every membership (for the
// team list) and the active team's seasons (for the nested-season list). Only
// the active team's seasons are fetched — the rail never fans out `ListSeasons`
// across every team.
//
// Layout and page loads run in parallel, so this skips the RPC when signed out
// and lets the root layout guard's redirect win. A stale `team_slug` (not a
// member, or deleted) yields an empty season list here; the page loader renders
// its own not-found state.
export const load = async ({
	params,
}: {
	params: { team_slug?: string; season_slug?: string };
}): Promise<TeamsShellData> => {
	const { team_slug: teamSlug, season_slug: seasonSlug } = params;
	if ((await session.load()) === null) {
		return {
			memberships: [],
			activeTeamSlug: teamSlug,
			activeSeasonSlug: seasonSlug,
			activeTeamSeasons: [],
		};
	}

	const memberships = (await teams.listMyTeams({})).memberships;
	const activeMembership = memberships.find((entry) => entry.team?.slug === teamSlug);
	const activeTeamSeasons =
		activeMembership?.team === undefined
			? []
			: (await seasonClient.listSeasons({ teamSlug: teamSlug ?? '', offset: 0, limit: 100 }))
					.seasons;

	return {
		memberships,
		activeTeamSlug: teamSlug,
		activeSeasonSlug: seasonSlug,
		activeTeamSeasons,
	};
};
