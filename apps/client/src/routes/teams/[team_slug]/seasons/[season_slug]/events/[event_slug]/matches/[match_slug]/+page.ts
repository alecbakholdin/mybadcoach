import { events as eventClient, matches as matchClient, teams } from '#lib/api';
import { undefinedIfNotFound } from '#lib/api/errors';
import { session } from '#lib/session.svelte';
import { teamRoleOf, type MatchDetailData } from '#lib/structure-data';

// Nested-slug addressing: every request carries the whole ancestor chain. A
// non-member sees the not-found state without a second permission-denied
// request; a missing event or match also maps to that state.
export const load = async ({
	params
}: {
	params: {
		team_slug: string;
		season_slug: string;
		event_slug: string;
		match_slug: string;
	};
}): Promise<MatchDetailData> => {
	const {
		team_slug: teamSlug,
		season_slug: seasonSlug,
		event_slug: eventSlug,
		match_slug: matchSlug
	} = params;
	const empty = (): MatchDetailData => ({
		teamSlug,
		seasonSlug,
		eventSlug,
		matchSlug,
		teamName: '',
		eventName: '',
		role: 'member',
		match: undefined
	});
	if ((await session.load()) === null) {
		return empty();
	}
	const myTeams = await teams.listMyTeams({});
	const membership = myTeams.memberships.find((entry) => entry.team?.slug === teamSlug);
	if (membership?.team === undefined) {
		return empty();
	}

	const [eventResponse, matchResponse] = await Promise.all([
		undefinedIfNotFound(eventClient.getEvent({ teamSlug, seasonSlug, eventSlug })),
		undefinedIfNotFound(
			matchClient.getMatch({ teamSlug, seasonSlug, eventSlug, matchSlug })
		)
	]);
	return {
		teamSlug,
		seasonSlug,
		eventSlug,
		matchSlug,
		teamName: membership.team.name,
		eventName: eventResponse?.event?.name ?? '',
		role: teamRoleOf(membership.role),
		match: matchResponse?.match
	};
};
