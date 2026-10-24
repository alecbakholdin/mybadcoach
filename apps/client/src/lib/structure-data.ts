/**
 * Data shapes returned by the structure route load functions.
 *
 * Declared explicitly instead of importing the generated `./$types` modules so
 * type-aware lint resolves them from a fresh checkout, before
 * `svelte-kit sync` has generated the per-route type files.
 */
import type { Event } from '$shared/gen/mybadcoach/events/v1/events_pb';
import type { Match } from '$shared/gen/mybadcoach/matches/v1/matches_pb';
import type { Player } from '$shared/gen/mybadcoach/players/v1/players_pb';
import type { Season } from '$shared/gen/mybadcoach/seasons/v1/seasons_pb';
import type { TeamMembership } from '$shared/gen/mybadcoach/teams/v1/teams_pb';

export interface TeamListData {
	memberships: TeamMembership[];
}

export interface TeamDetailData {
	teamSlug: string;
	teamName: string;
	role: string;
	seasons: Season[];
}

export interface SeasonDetailData {
	teamSlug: string;
	seasonSlug: string;
	teamName: string;
	role: string;
	season?: Season;
	events: Event[];
	eventsTotal: number;
	roster: Player[];
	rosterTotal: number;
}

export interface EventDetailData {
	teamSlug: string;
	seasonSlug: string;
	eventSlug: string;
	teamName: string;
	seasonName: string;
	role: string;
	event?: Event;
	matches: Match[];
	matchesTotal: number;
}

export interface MatchDetailData {
	teamSlug: string;
	seasonSlug: string;
	eventSlug: string;
	matchSlug: string;
	teamName: string;
	eventName: string;
	role: string;
	match?: Match;
}
