package gameplay

// MatchScore is the final score of one completed match: us versus the faceless
// opponent (ADR-0003).
type MatchScore struct {
	OurScore      int64
	OpponentScore int64
}

// TeamRecord is a team's win/loss/tie totals over completed matches.
type TeamRecord struct {
	Wins   int64
	Losses int64
	Ties   int64
}

// ComputeTeamRecord computes a team's W/L/T record from final match scores,
// ported from computeTeamRecord in packages/shared/src/teamRecord.ts. The
// higher score wins; equal scores are a tie. No scores yields zeroed counts,
// never a nil result.
func ComputeTeamRecord(scores []MatchScore) TeamRecord {
	var record TeamRecord
	for _, score := range scores {
		switch {
		case score.OurScore > score.OpponentScore:
			record.Wins++
		case score.OurScore < score.OpponentScore:
			record.Losses++
		default:
			record.Ties++
		}
	}
	return record
}
