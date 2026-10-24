import { players, teams } from '#lib/api';
import { session } from '#lib/session.svelte';
import { teamRoleOf, type PlayerListData } from '#lib/structure-data';

// Players are scoped to a team; the membership role gates owner-only deletion
// in the UI, while any member may create and edit players.
export const load = async ({
	params,
}: {
	params: { team_slug: string };
}): Promise<PlayerListData> => {
	const { team_slug: teamSlug } = params;
	const empty = (): PlayerListData => ({
		teamSlug,
		teamName: '',
		role: 'member',
		players: [],
		playersTotal: 0,
	});
	if ((await session.load()) === null) {
		return empty();
	}
	const myTeams = await teams.listMyTeams({});
	const membership = myTeams.memberships.find((entry) => entry.team?.slug === teamSlug);
	if (membership?.team === undefined) {
		return empty();
	}

	const response = await players.listPlayers({ teamSlug, offset: 0, limit: 100 });
	return {
		teamSlug,
		teamName: membership.team.name,
		role: teamRoleOf(membership.role),
		players: response.players,
		playersTotal: Number(response.total),
	};
};
