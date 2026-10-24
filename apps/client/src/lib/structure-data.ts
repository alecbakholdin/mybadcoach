/**
 * Data shapes returned by the structure route load functions.
 *
 * Declared explicitly instead of importing the generated `./$types` modules so
 * type-aware lint resolves them from a fresh checkout, before
 * `svelte-kit sync` has generated the per-route type files.
 */
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
	roster: Player[];
	rosterTotal: number;
}
