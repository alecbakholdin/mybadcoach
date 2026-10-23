package gameplay

import "testing"

func TestComputeTeamRecordZeroesWithNoMatches(t *testing.T) {
	if got := ComputeTeamRecord(nil); got != (TeamRecord{}) {
		t.Fatalf("record = %#v, want zero", got)
	}
}

func TestComputeTeamRecordCountsWinsLossesAndTies(t *testing.T) {
	record := ComputeTeamRecord([]MatchScore{
		{OurScore: 15, OpponentScore: 12},
		{OurScore: 10, OpponentScore: 15},
		{OurScore: 13, OpponentScore: 13},
		{OurScore: 15, OpponentScore: 14},
	})

	want := TeamRecord{Wins: 2, Losses: 1, Ties: 1}
	if record != want {
		t.Fatalf("record = %#v, want %#v", record, want)
	}
}
