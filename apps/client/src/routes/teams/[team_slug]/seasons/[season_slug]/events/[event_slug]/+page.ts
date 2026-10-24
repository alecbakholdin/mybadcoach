import {
	events as eventClient,
	matches as matchClient,
	seasons as seasonClient,
	teams
} from '#lib/api';
import { undefinedIfNotFound } from '#lib/api/errors';
import { session } from '#lib/session.svelte';
import type { EventDetailData } from '#lib/structure-data';

// Nested-slug addressing: every request carries the whole ancestor chain. The
// caller's role comes from `ListMyTeams`, and a non-member sees the not-found
// state without a second permission-denied request. Missing descendants map to
// undefined so the screen renders its own not-found state instead of an error
// page.
export const load = async ({
	params
}: {
	params: { team_slug: string; season_slug: string; event_slug: string };
}): Promise<EventDetailData> => {
	const {
		team_slug: teamSlug,
		season_slug: seasonSlug,
		event_slug: eventSlug
	} = params;
	const empty = (): EventDetailData => ({
		teamSlug,
		seasonSlug,
		eventSlug,
		teamName: '',
		seasonName: '',
		role: '',
		event: undefined,
		matches: [],
		matchesTotal: 0
	});
	if ((await session.load()) === null) {
		return empty();
	}
	const myTeams = await teams.listMyTeams({});
	const membership = myTeams.memberships.find((entry) => entry.team?.slug === teamSlug);
	if (membership?.team === undefined) {
		return empty();
	}

	const [seasonResponse, eventResponse, matchResponse] = await Promise.all([
		undefinedIfNotFound(seasonClient.getSeason({ teamSlug, seasonSlug })),
		undefinedIfNotFound(eventClient.getEvent({ teamSlug, seasonSlug, eventSlug })),
		undefinedIfNotFound(
			matchClient.listMatches({ teamSlug, seasonSlug, eventSlug, offset: 0, limit: 100 })
		)
	]);
	return {
		teamSlug,
		seasonSlug,
		eventSlug,
		teamName: membership.team.name,
		seasonName: seasonResponse?.season?.name ?? '',
		role: membership.role,
		event: eventResponse?.event,
		matches: matchResponse?.matches ?? [],
		matchesTotal: Number(matchResponse?.total ?? 0n)
	};
};
