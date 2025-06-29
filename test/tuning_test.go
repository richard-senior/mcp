package test

import (
	"fmt"
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

// PredictionResult holds the results of a prediction test
type PredictionResult struct {
	CorrectPredictions int
	TotalPredictions   int
	TotalHomeWinProb   float64
	TotalDrawProb      float64
	TotalAwayWinProb   float64
	PredictedMatches   int
	SkippedMatches     int
}

type IterResult struct {
}

var (
	leagueID  = 47
	season    = "2024/2025"
	teamStats []*podds.TeamStats
	matches   []*podds.Match
	params    = map[string][]any{
		"formWeight":      {0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9},
		"rhoValue":        {-0.01, -0.02, -0.03, -0.04, -0.05, -0.06, -0.07, -0.08},
		"derbyBoostValue": {1.00, 1.02, 1.04, 1.06, 1.08, 1.10, 1.12, 1.15, 1.20},
	}
	bestAccuracy   = 0.0
	bestFormWeight = 0.0
	bestVal        any
	result         *PredictionResult
)

func TestTuning(t *testing.T) {

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
		if matches[i].ActualAwayGoals == -1 || matches[i].ActualHomeGoals == -1 {
			// Remove element at index i
			matches = append(matches[:i], matches[i+1:]...)
		}
	}

	if len(matches) == 0 {
		t.Fatalf("No matches found for League %d, Season %s", leagueID, season)
	}

	// Get processed TeamStats for those matches
	teamStats, err = podds.ProcessTeamStats(matches, leagueID, season)
	if err != nil {
		t.Fatalf("Failed to process team stats: %v", err)
	}

	// Define config setter functions for each parameter
	configSetters := map[string]func(any){
		"formWeight": func(val any) {
			if v, ok := val.(float64); ok {
				podds.SetFormWeight(v)
			}
		},
		"rhoValue": func(val any) {
			if v, ok := val.(float64); ok {
				podds.Config.DixonColesRho = v
			}
		},
		"derbyBoostValue": func(val any) {
			if v, ok := val.(float64); ok {
				podds.Config.DerbyBoostMultiplier = v
			}
		},
	}

	// Test each parameter
	for key := range params {
		if setter, exists := configSetters[key]; exists {
			doTest(key, setter)
			break
		} else {
			fmt.Printf("Warning: No config setter found for parameter: %s\n", key)
		}
	}
	dumpMatches()
}

func doTest(param string, configSetter func(any)) {
	// tune parameter values
	bestAccuracy = 0.0
	valsArray := params[param]
	printHeader(param)
	for _, value := range valsArray {
		configSetter(value) // Use the passed function to set the config
		doIteration(value)
	}
	printFooter(param)
	// use configSetter to set the config to the discovered optimal value
	configSetter(bestVal)
}

func printHeader(paramName string) {
	fmt.Printf("%s | Correct | Total | Accuracy | Avg Home Win | Avg Draw | Avg Away Win | Predicted | Skipped\n", paramName)
	fmt.Printf("-----------|---------|-------|----------|--------------|----------|--------------|-----------|--------\n")
}
func printFooter(paramName string) {
	fmt.Printf("\nBest %s: %.3f with accuracy: %.2f%%\n", paramName, bestVal, bestAccuracy)
}

func doIteration(val any) {
	result = RunPredictionsWithConfig()
	accuracy := result.CalculateAccuracy()
	avgHomeWin, avgDraw, avgAwayWin := result.GetAverageProbabilities()
	// Track best accuracy
	if accuracy > bestAccuracy {
		bestAccuracy = accuracy
		bestVal = val
	}
	fmt.Printf("   %.1f     |   %3d   |  %3d  |  %6.2f%%  |    %6.2f%%    |  %6.2f%%  |   %6.2f%%   |    %3d    |   %3d\n",
		val, result.CorrectPredictions, result.TotalPredictions, accuracy, avgHomeWin, avgDraw, avgAwayWin, result.PredictedMatches, result.SkippedMatches)
}

// Dump out the matches to console showing:
// "homeTeamName vs awayTeamName" actualHomeGoals : actualAwayGoals predictedHomeGoals : predictedAwayGoals
func dumpMatches() {
	for _, match := range matches {
		if match.ActualHomeGoals == -1 || match.ActualAwayGoals == -1 {
			continue
		}
		fmt.Printf("%s vs %s %d - %d (%d - %d)\n", match.HomeTeamName, match.AwayTeamName, match.ActualHomeGoals, match.ActualAwayGoals, match.PoissonPredictedHomeGoals, match.PoissonPredictedAwayGoals)
	}
}

// RunPredictionsWithConfig tests predictions with a given configuration
// This is the core prediction testing function that can be reused for different parameters
func RunPredictionsWithConfig() *PredictionResult {
	result := &PredictionResult{}

	for _, match := range matches {
		// Only predict for matches that have results (for accuracy testing)
		if match.ActualHomeGoals == -1 || match.ActualAwayGoals == -1 {
			result.SkippedMatches++
			continue
		}

		// Clear any existing predictions to ensure fresh calculation
		match.PoissonHomeWinProbability = -1.0
		match.PoissonDrawProbability = -1.0
		match.PoissonAwayWinProbability = -1.0
		match.Over1p5Goals = -1.0
		match.Over2p5Goals = -1.0

		// Predict the match
		err := podds.PredictMatch(match, teamStats)
		if err != nil {
			result.SkippedMatches++
			continue
		}

		// Check if prediction was made
		if match.PoissonHomeWinProbability == -1.0 || match.PoissonDrawProbability == -1.0 || match.PoissonAwayWinProbability == -1.0 {
			result.SkippedMatches++
			continue
		}

		result.PredictedMatches++
		result.TotalPredictions++
		result.TotalHomeWinProb += match.PoissonHomeWinProbability
		result.TotalDrawProb += match.PoissonDrawProbability
		result.TotalAwayWinProb += match.PoissonAwayWinProbability

		// Determine actual result
		actualResult := ""
		if match.ActualHomeGoals > match.ActualAwayGoals {
			actualResult = "H"
		} else if match.ActualHomeGoals < match.ActualAwayGoals {
			actualResult = "A"
		} else {
			actualResult = "D"
		}

		// Determine predicted result (highest probability)
		predictedResult := ""
		if match.PoissonHomeWinProbability > match.PoissonDrawProbability && match.PoissonHomeWinProbability > match.PoissonAwayWinProbability {
			predictedResult = "H"
		} else if match.PoissonAwayWinProbability > match.PoissonDrawProbability && match.PoissonAwayWinProbability > match.PoissonHomeWinProbability {
			predictedResult = "A"
		} else {
			predictedResult = "D"
		}

		// Check if prediction was correct
		if actualResult == predictedResult {
			result.CorrectPredictions++
		}
	}
	return result
}

// CalculateAccuracy computes accuracy percentage from prediction results
func (pr *PredictionResult) CalculateAccuracy() float64 {
	if pr.TotalPredictions == 0 {
		return 0.0
	}
	return float64(pr.CorrectPredictions) / float64(pr.TotalPredictions) * 100
}

// GetAverageProbabilities returns average probabilities as percentages
func (pr *PredictionResult) GetAverageProbabilities() (homeWin, draw, awayWin float64) {
	if pr.TotalPredictions == 0 {
		return 0.0, 0.0, 0.0
	}
	return pr.TotalHomeWinProb / float64(pr.TotalPredictions),
		pr.TotalDrawProb / float64(pr.TotalPredictions),
		pr.TotalAwayWinProb / float64(pr.TotalPredictions)
}
