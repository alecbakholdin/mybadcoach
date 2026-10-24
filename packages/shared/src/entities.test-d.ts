import type {
	Event as DomainEvent,
	EventRosterEntry,
	Match,
	Player,
	Season,
	SeasonRosterEntry,
	Team,
} from './entities';

type Expect<T extends true> = T;
type HasKey<T, K extends PropertyKey> = K extends keyof T ? true : false;
type IsAssignable<From, To> = From extends To ? true : false;

const createdAt = '2026-03-01T00:00:00Z';

const team: Team = {
	id: 'team-1',
	name: 'My Bad',
	slug: 'my-bad',
	createdAt,
};

const season: Season = {
	id: 'season-1',
	teamId: team.id,
	name: '2026 Spring',
	slug: '2026-spring',
	startDate: '2026-03-01',
	endDate: '2026-05-31',
	createdAt,
};

const event: DomainEvent = {
	id: 'event-1',
	seasonId: season.id,
	name: 'Spring Fling',
	slug: 'spring-fling',
	startDate: '2026-03-14',
	endDate: '2026-03-15',
	location: 'Portland, OR',
};

const match: Match = {
	id: 'match-1',
	eventId: event.id,
	ourTeamId: team.id,
	opponentName: 'Other Team',
	slug: 'other-team-2026-03-14',
	scheduledAt: '2026-03-14T09:00:00Z',
	pointCap: 15,
	deleted: false,
	createdAt,
};

const player: Player = {
	id: 'player-1',
	teamId: team.id,
	names: ['Alex Bakholdin', 'A-B'],
	slug: 'alex-bakholdin',
	deleted: false,
	createdAt,
};

const seasonRosterEntry: SeasonRosterEntry = {
	id: 'season-roster-1',
	seasonId: season.id,
	playerId: player.id,
	createdAt,
};

const eventRosterEntry: EventRosterEntry = {
	id: 'event-roster-1',
	eventId: event.id,
	playerId: player.id,
	createdAt,
};

export const domainModelSample = {
	team,
	season,
	event,
	match,
	player,
	seasonRosterEntry,
	eventRosterEntry,
} as const;

// The hierarchy carries foreign keys: Team -> Season -> Event -> Match (ADR-0007).
type _SeasonBelongsToTeam = Expect<HasKey<Season, 'teamId'>>;
type _EventBelongsToSeason = Expect<HasKey<DomainEvent, 'seasonId'>>;
type _MatchBelongsToEvent = Expect<HasKey<Match, 'eventId'>>;
type _MatchHasOurTeam = Expect<HasKey<Match, 'ourTeamId'>>;

// The opponent is faceless: a name string only (ADR-0003).
type _MatchHasOpponentName = Expect<HasKey<Match, 'opponentName'>>;
type _MatchHasNoOpponentRoster = Expect<
	HasKey<Match, 'opponentRoster'> extends false ? true : false
>;

// Players carry one or more names: primary first, then aliases (GLOSSARY.md).
type _PlayerNamesNonEmpty = Expect<Player['names'] extends [string, ...string[]] ? true : false>;
type _PlayerNameIsString = Expect<IsAssignable<Player['names'][0], string>>;

// Rosters are season- and event-scoped and never store lineup columns (ADR-0005).
type _SeasonRosterHasPlayer = Expect<HasKey<SeasonRosterEntry, 'playerId'>>;
type _EventRosterHasPlayer = Expect<HasKey<EventRosterEntry, 'playerId'>>;
type _SeasonRosterHasNoStartingFlag = Expect<
	HasKey<SeasonRosterEntry, 'isStarting'> extends false ? true : false
>;
type _SeasonRosterHasNoPosition = Expect<
	HasKey<SeasonRosterEntry, 'position'> extends false ? true : false
>;
type _EventRosterHasNoStartingFlag = Expect<
	HasKey<EventRosterEntry, 'isStarting'> extends false ? true : false
>;
type _EventRosterHasNoPosition = Expect<
	HasKey<EventRosterEntry, 'position'> extends false ? true : false
>;
