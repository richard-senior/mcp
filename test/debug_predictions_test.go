package test

import (
	"fmt"
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

// TestDebugPredictions examines a few sample predictions to understand the probability values
func TestDebugPredictions(t *testing.T) {
	// Load sample matches from Premier League 2024/2025
	leagueID := 47
	season := "2024/2025"
	
	fmt.Printf("Loading matches for debugging predictions...\n")
	
	matchesMap, err := podds.LoadExistingMatches(leagueID, season)
	if err != nil {
		t.Fatalf("Failed to load existing matches: %v", err)
	}
	
	// Convert map to slice for processing
	var matches []*podds.Match
	for _, match := range matchesMap {
		matches = append(matches, match)
		if len(matches) >= 5 { // Only examine first 5 matches
			break
		}
	}
	
	if len(matches) == 0 {
		t.Fatalf("No matches found for League %d, Season %s", leagueID, season)
	}
	
	// Get processed TeamStats for those matches
	teamStats, err := podds.ProcessTeamStats(matches, leagueID, season)
	if err != nil {
		t.Fatalf("Failed to process team stats: %v", err)
	}
	
	fmt.Printf("\n=== DEBUGGING SAMPLE PREDICTIONS ===\n")
	
	for i, match := range matches {
		if match.ActualHomeGoals == -1 || match.ActualAwayGoals == -1 {
			continue
		}
		
		// Clear any existing predictions
		match.PoissonHomeWinProbability = -1.0
		match.PoissonDrawProbability = -1.0
		match.PoissonAwayWinProbability = -1.0
		
		// Predict the match
		err := podds.PredictMatch(match, teamStats)
		if err != nil {
			fmt.Printf("Match %d: Error predicting - %v\n", i+1, err)
			continue
		}
		
		fmt.Printf("Match %d: %s vs %s\n", i+1, match.HomeTeamName, match.AwayTeamName)
		fmt.Printf("  Actual Result: %d-%d\n", match.ActualHomeGoals, match.ActualAwayGoals)
		fmt.Printf("  Home Win Prob: %.4f\n", match.PoissonHomeWinProbability)
		fmt.Printf("  Draw Prob: %.4f\n", match.PoissonDrawProbability)
		fmt.Printf("  Away Win Prob: %.4f\n", match.PoissonAwayWinProbability)
		fmt.Printf("  Sum: %.4f\n", match.PoissonHomeWinProbability + match.PoissonDrawProbability + match.PoissonAwayWinProbability)
		fmt.Printf("  Expected Goals: Home=%.2f, Away=%.2f\n", match.HomeTeamGoalExpectency, match.AwayTeamGoalExpectency)
		fmt.Printf("\n")
	}
}
