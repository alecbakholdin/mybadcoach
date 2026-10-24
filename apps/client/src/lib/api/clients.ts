/**
 * One typed Connect client per service, all sharing the transport in
 * `transport.ts`. Only `auth` is exercised by the auth screens (ticket 05);
 * tickets 06-08 consume the structure clients.
 */
import { createClient } from '@connectrpc/connect';

import { transport } from '#lib/api/transport';
import { AuthService } from '$shared/gen/mybadcoach/auth/v1/auth_pb';
import { EventService } from '$shared/gen/mybadcoach/events/v1/events_pb';
import { MatchService } from '$shared/gen/mybadcoach/matches/v1/matches_pb';
import { PlayerService } from '$shared/gen/mybadcoach/players/v1/players_pb';
import { SeasonService } from '$shared/gen/mybadcoach/seasons/v1/seasons_pb';
import { TeamService } from '$shared/gen/mybadcoach/teams/v1/teams_pb';

export const auth = createClient(AuthService, transport);
export const teams = createClient(TeamService, transport);
export const seasons = createClient(SeasonService, transport);
export const events = createClient(EventService, transport);
export const matches = createClient(MatchService, transport);
export const players = createClient(PlayerService, transport);
