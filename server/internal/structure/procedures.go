package structure

import (
	eventsv1connect "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1/eventsv1connect"
	matchesv1connect "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1/matchesv1connect"
	playersv1connect "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1/playersv1connect"
	seasonsv1connect "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1/seasonsv1connect"
	teamsv1connect "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1/teamsv1connect"
)

// MutatingProcedures returns the structure RPCs that change state. They are
// CSRF-gated in addition to requiring a session.
func MutatingProcedures() []string {
	return []string{
		teamsv1connect.TeamServiceCreateTeamProcedure,
		teamsv1connect.TeamServiceUpdateTeamProcedure,
		teamsv1connect.TeamServiceDeleteTeamProcedure,
		teamsv1connect.TeamServiceAddMembershipProcedure,
		teamsv1connect.TeamServiceUpdateMembershipProcedure,
		teamsv1connect.TeamServiceRemoveMembershipProcedure,
		seasonsv1connect.SeasonServiceCreateSeasonProcedure,
		seasonsv1connect.SeasonServiceUpdateSeasonProcedure,
		seasonsv1connect.SeasonServiceDeleteSeasonProcedure,
		seasonsv1connect.SeasonServiceAddSeasonRosterEntryProcedure,
		seasonsv1connect.SeasonServiceRemoveSeasonRosterEntryProcedure,
		seasonsv1connect.SeasonServiceCopySeasonRosterProcedure,
		eventsv1connect.EventServiceCreateEventProcedure,
		eventsv1connect.EventServiceUpdateEventProcedure,
		eventsv1connect.EventServiceDeleteEventProcedure,
		eventsv1connect.EventServiceAddEventRosterEntryProcedure,
		eventsv1connect.EventServiceRemoveEventRosterEntryProcedure,
		playersv1connect.PlayerServiceCreatePlayerProcedure,
		playersv1connect.PlayerServiceUpdatePlayerProcedure,
		playersv1connect.PlayerServiceDeletePlayerProcedure,
		matchesv1connect.MatchServiceCreateMatchProcedure,
		matchesv1connect.MatchServiceUpdateMatchProcedure,
		matchesv1connect.MatchServiceDeleteMatchProcedure,
		matchesv1connect.MatchServiceAppendMatchEventsProcedure,
	}
}

// AuthenticatedProcedures returns every structure RPC. None is reachable
// anonymously: the session middleware rejects a request without a valid
// session before the handler runs.
func AuthenticatedProcedures() []string {
	procedures := MutatingProcedures()
	return append(
		procedures,
		teamsv1connect.TeamServiceGetTeamProcedure,
		teamsv1connect.TeamServiceListMyTeamsProcedure,
		teamsv1connect.TeamServiceGetTeamRecordProcedure,
		seasonsv1connect.SeasonServiceGetSeasonProcedure,
		seasonsv1connect.SeasonServiceListSeasonsProcedure,
		seasonsv1connect.SeasonServiceListSeasonRosterProcedure,
		eventsv1connect.EventServiceGetEventProcedure,
		eventsv1connect.EventServiceListEventsProcedure,
		eventsv1connect.EventServiceListEventRosterProcedure,
		playersv1connect.PlayerServiceGetPlayerProcedure,
		playersv1connect.PlayerServiceListPlayersProcedure,
		matchesv1connect.MatchServiceGetMatchProcedure,
		matchesv1connect.MatchServiceListMatchesProcedure,
		matchesv1connect.MatchServiceListMatchEventsProcedure,
		matchesv1connect.MatchServiceGetMatchStateProcedure,
		matchesv1connect.MatchServiceGetMatchStatsProcedure,
	)
}
