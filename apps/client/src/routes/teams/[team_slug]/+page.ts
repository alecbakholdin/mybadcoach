import { seasons as seasonClient, teams } from '#lib/api';
import { session } from '#lib/session.svelte';
import { teamRoleOf, type TeamDetailData } from '#lib/structure-data';

// The caller's role comes from `ListMyTeams` (the contract's client-boot call),
// and the same call decides whether the team is visible at all: a non-member
// sees the "team not found" state without a second permission-denied request.
export const load = async ({
	params
}: {
	params: { team_slug: string };
}): Promise<TeamDetailData> => {
	const { team_slug: teamSlug } = params;
	if ((await session.load()) === null) {
		return { teamSlug, teamName: '', role: 'member', seasons: [] };
	}
	const myTeams = await teams.listMyTeams({});
	const membership = myTeams.memberships.find((entry) => entry.team?.slug === teamSlug);
	if (membership?.team === undefined) {
		return { teamSlug, teamName: '', role: 'member', seasons: [] };
	}

	const response = await seasonClient.listSeasons({ teamSlug, offset: 0, limit: 100 });
	return {
		teamSlug,
		teamName: membership.team.name,
		role: teamRoleOf(membership.role),
		seasons: response.seasons
	};
};
