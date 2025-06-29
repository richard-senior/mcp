package test

import (
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

// TestUpdate tests the general operation of Podds
// and its interaction with various datasources and the database etc.
func TestPoddsUpdate(t *testing.T) {
	p := podds.NewPodds()
	err := p.Update()
	if err != nil {
		t.Error(err)
	}
}

// TestPoddsUpdateWithPredictions tests the Update method and forces predictions on all matches
// This is useful for testing prediction accuracy on historical data
func TestPoddsUpdateWithPredictions(t *testing.T) {
	// Load matches for testing
	leagueID := 47
	season := "2024/2025"
	
	matchesMap, err := podds.LoadExistingMatches(leagueID, season)
	if err != nil {
		t.Fatalf("Failed to load existing matches: %v", err)
	}

	// Convert map to slice for processing
	var matches []*podds.Match
	for _, match := range matchesMap {
		matches = append(matches, match)
	}

	if len(matches) == 0 {
		t.Fatalf("No matches found for League %d, Season %s", leagueID, season)
	}

	// Get processed TeamStats for those matches
	teamStats, err := podds.ProcessTeamStats(matches, leagueID, season)
	if err != nil {
		t.Fatalf("Failed to process team stats: %v", err)
	}

	// Clear existing predictions to force re-prediction
	for _, match := range matches {
		match.PoissonPredictedHomeGoals = -1
		match.PoissonPredictedAwayGoals = -1
		match.PoissonHomeWinProbability = -1.0
		match.PoissonDrawProbability = -1.0
		match.PoissonAwayWinProbability = -1.0
		match.Over1p5Goals = -1.0
		match.Over2p5Goals = -1.0
	}

	// Predict all matches using the original PredictMatch function
	predictedCount := 0
	for _, match := range matches {
		err := podds.PredictMatch(match, teamStats)
		if err != nil {
			t.Logf("Failed to predict match %s vs %s: %v", match.HomeTeamName, match.AwayTeamName, err)
			continue
		}
		if match.PoissonPredictedHomeGoals != -1 {
			predictedCount++
		}
	}

	t.Logf("Successfully predicted %d out of %d matches", predictedCount, len(matches))

	// Save the matches with predictions back to database
	if err := podds.SaveMatches(matches); err != nil {
		t.Fatalf("Failed to save matches with predictions: %v", err)
	}

	t.Logf("Saved %d matches with predictions to database", len(matches))
}
