package test

import (
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

func TestPrediction(t *testing.T) {
	matchesMap, err := podds.LoadExistingMatches(leagueID, season)
	if err != nil {
		t.Fatalf("Failed to load existing matches: %v", err)
	}

	// Convert map to slice for processing
	for _, match := range matchesMap {
		matches = append(matches, match)
	}

	// Sanitise the array of matches removing any that have no actual home goals
	for i := len(matches) - 1; i >= 0; i-- {
		matches[i].PoissonPredictedAwayGoals = -1
		matches[i].PoissonPredictedHomeGoals = -1
		if matches[i].ActualAwayGoals == -1 || matches[i].ActualHomeGoals == -1 {
			// Remove element at index i
			matches = append(matches[:i], matches[i+1:]...)
		}
	}

	// run matches through the prediction
	ds := podds.Datasource{}
	m, err := ds.ProcessLeagueMatches(matches, []*podds.Match{})
	// show prediction results somehow?
}
