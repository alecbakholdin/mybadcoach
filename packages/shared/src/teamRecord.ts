/**
 * The final score of one completed match: us versus the faceless opponent.
 *
 * A match's own score is not modeled on the `Match` entity yet (match-level
 * point lifecycle is out of scope), so W/L/T is computed from this minimal
 * input rather than from stored entity fields.
 */
export interface MatchScore {
	ourScore: number;
	opponentScore: number;
}

/** A team's win/loss/tie totals over completed matches. */
export interface TeamRecord {
	wins: number;
	losses: number;
	ties: number;
}

/**
 * Compute a team's W/L/T record from final match scores.
 *
 * The higher score wins; equal scores are a tie. Forfeits, time caps, and
 * score validation are out of scope, so this is a pure comparison per match.
 * No scores yields zeroed counts, never null.
 */
export function computeTeamRecord(scores: readonly MatchScore[]): TeamRecord {
	const record: TeamRecord = { wins: 0, losses: 0, ties: 0 };

	for (const { ourScore, opponentScore } of scores) {
		if (ourScore > opponentScore) {
			record.wins += 1;
		} else if (ourScore < opponentScore) {
			record.losses += 1;
		} else {
			record.ties += 1;
		}
	}

	return record;
}
