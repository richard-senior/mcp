package test

import (
	"fmt"
	"reflect"
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
	TotalScoreInaccuracy int     // Sum of all score inaccuracies
	ScoreInaccuracyCount int     // Number of matches with score predictions
}

// TuningParam defines a parameter to tune with its configuration path and values
type TuningParam struct {
	Name         string // Display name for the parameter
	ConfigPath   string // Path to the config field (e.g., "Config.DixonColesRho")
	FunctionCall string // Optional: function name to call instead of direct field access
	Values       []any  // Values to test
	Skip         bool   // If true then we should skip tuning this param
}

var (
	leagueID  = 47
	season    = "2024/2025"
	teamStats []*podds.TeamStats
	matches   []*podds.Match

	// Enhanced parameter definition with automatic setter generation
	params = []TuningParam{
		// === CORE PREDICTION PARAMETERS ===
		{
			Name:         "formWeight",
			FunctionCall: "SetFormWeight", // Uses podds.SetFormWeight(value)
			Values:       []any{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9},
			Skip:         false,
		},
		{
			Name:       "poissonSimulations",
			ConfigPath: "Config.PoissonSimulations",
			Values:     []any{50000, 75000, 100000, 125000, 150000, 200000},
			Skip:       true, // Skip by default - computationally expensive
		},
		{
			Name:       "poissonRange",
			ConfigPath: "Config.PoissonRange",
			Values:     []any{7, 8, 9, 10, 11, 12},
			Skip:       true, // Skip by default - affects matrix size
		},
		{
			Name:       "maxGoalsCap",
			ConfigPath: "Config.MaxGoalsCap",
			Values:     []any{8.0, 10.0, 12.0, 15.0, 20.0},
			Skip:       true, // Skip by default - rarely reached
		},
		{
			Name:       "minGoalsFloor",
			ConfigPath: "Config.MinGoalsFloor",
			Values:     []any{0.0, 0.01, 0.05, 0.1},
			Skip:       true, // Skip by default - rarely used
		},
		{
			Name:       "makeSensibleDefault",
			ConfigPath: "Config.MakeSensibleDefault",
			Values:     []any{0.8, 0.9, 1.0, 1.1, 1.2},
			Skip:       true, // Skip by default - division by zero protection
		},

		// === DIXON-COLES CORRECTION ===
		{
			Name:       "dixonColesRho",
			ConfigPath: "Config.DixonColesRho",
			Values:     []any{-0.01, -0.02, -0.03, -0.04, -0.05, -0.06, -0.07, -0.08},
			Skip:       true,
		},

		// === TRAVEL DISTANCE (POKE) ADJUSTMENTS ===
		{
			Name:       "derbyDistanceThreshold",
			ConfigPath: "Config.DerbyDistanceThreshold",
			Values:     []any{5, 10, 15, 20, 25, 30},
			Skip:       true, // Skip by default - affects few matches
		},
		{
			Name:       "derbyBoostMultiplier",
			ConfigPath: "Config.DerbyBoostMultiplier",
			Values:     []any{1.00, 1.02, 1.04, 1.06, 1.08, 1.10, 1.12, 1.15, 1.20},
			Skip:       true,
		},
		{
			Name:       "shortTravelThreshold",
			ConfigPath: "Config.ShortTravelThreshold",
			Values:     []any{30, 40, 50, 60, 70},
			Skip:       true, // Skip by default - threshold parameter
		},
		{
			Name:       "mediumTravelThreshold",
			ConfigPath: "Config.MediumTravelThreshold",
			Values:     []any{80, 90, 100, 110, 120},
			Skip:       true, // Skip by default - threshold parameter
		},
		{
			Name:       "longTravelThreshold",
			ConfigPath: "Config.LongTravelThreshold",
			Values:     []any{150, 175, 200, 225, 250},
			Skip:       true, // Skip by default - threshold parameter
		},
		{
			Name:       "veryLongTravelThreshold",
			ConfigPath: "Config.VeryLongTravelThreshold",
			Values:     []any{250, 275, 300, 325, 350},
			Skip:       true, // Skip by default - threshold parameter
		},
		{
			Name:       "shortTravelPenalty",
			ConfigPath: "Config.ShortTravelPenalty",
			Values:     []any{0.96, 0.97, 0.98, 0.99, 1.00},
			Skip:       true, // Skip by default - minor effect
		},
		{
			Name:       "mediumTravelPenalty",
			ConfigPath: "Config.MediumTravelPenalty",
			Values:     []any{0.92, 0.94, 0.96, 0.98, 1.00},
			Skip:       true, // Skip by default - moderate effect
		},
		{
			Name:       "longTravelPenalty",
			ConfigPath: "Config.LongTravelPenalty",
			Values:     []any{0.88, 0.90, 0.92, 0.94, 0.96},
			Skip:       true, // Keep enabled - significant effect
		},
		{
			Name:       "veryLongTravelPenalty",
			ConfigPath: "Config.VeryLongTravelPenalty",
			Values:     []any{0.82, 0.85, 0.88, 0.91, 0.94},
			Skip:       true, // Skip by default - affects few matches
		},

		// === OVER/UNDER GOALS THRESHOLDS ===
		{
			Name:       "over1p5GoalsThreshold",
			ConfigPath: "Config.Over1p5GoalsThreshold",
			Values:     []any{1.3, 1.4, 1.5, 1.6, 1.7},
			Skip:       true, // Skip by default - doesn't affect win/draw/loss predictions
		},
		{
			Name:       "over2p5GoalsThreshold",
			ConfigPath: "Config.Over2p5GoalsThreshold",
			Values:     []any{2.3, 2.4, 2.5, 2.6, 2.7},
			Skip:       true, // Skip by default - doesn't affect win/draw/loss predictions
		},

		// === FORM CALCULATION PARAMETERS ===
		{
			Name:       "formLossValue",
			ConfigPath: "Config.FormLossValue",
			Values:     []any{0, 1, 2},
			Skip:       true, // Skip by default - affects form calculation
		},
		{
			Name:       "formDrawValue",
			ConfigPath: "Config.FormDrawValue",
			Values:     []any{1, 2, 3},
			Skip:       true, // Skip by default - affects form calculation
		},
		{
			Name:       "formWinValue",
			ConfigPath: "Config.FormWinValue",
			Values:     []any{2, 3, 4, 5},
			Skip:       true, // Skip by default - affects form calculation
		},
	}

	bestAccuracy = 0.0
	bestVal      any
	result       *PredictionResult
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

	// Test each parameter using reflection-based setters
	for _, param := range params {
		if param.Skip {
			continue
		}
		setter, err := createConfigSetter(param)
		if err != nil {
			fmt.Printf("Warning: Could not create setter for %s: %v\n", param.Name, err)
			continue
		}
		doTest(param.Name, param.Values, setter)
		break // Only test first parameter for now
	}
	dumpMatches()
}

func doTest(paramName string, values []any, configSetter func(any)) {
	// tune parameter values
	bestAccuracy = 0.0
	printHeader(paramName)
	for _, value := range values {
		configSetter(value) // Use the generated setter function
		doIteration(value)
	}
	printFooter(paramName)
	// use configSetter to set the config to the discovered optimal value
	configSetter(bestVal)
}

func printHeader(paramName string) {
	fmt.Printf("%s | Correct | Total | Accuracy | Avg Home Win | Avg Draw | Avg Away Win | Avg Score Inaccuracy | Predicted | Skipped\n", paramName)
	fmt.Printf("-----------|---------|-------|----------|--------------|----------|--------------|----------------------|-----------|--------\n")
}

func printFooter(paramName string) {
	// Format best value appropriately based on type
	var bestValStr string
	switch v := bestVal.(type) {
	case int:
		bestValStr = fmt.Sprintf("%d", v)
	case float64:
		bestValStr = fmt.Sprintf("%.3f", v)
	default:
		bestValStr = fmt.Sprintf("%v", v)
	}
	fmt.Printf("\nBest %s: %s with accuracy: %.2f%%\n", paramName, bestValStr, bestAccuracy)
}

func doIteration(val any) {
	result = RunPredictionsWithConfig()
	accuracy := result.CalculateAccuracy()
	avgHomeWin, avgDraw, avgAwayWin := result.GetAverageProbabilities()
	avgScoreInaccuracy := result.GetAverageScoreInaccuracy()
	
	// Track best accuracy
	if accuracy > bestAccuracy {
		bestAccuracy = accuracy
		bestVal = val
	}

	// Format value appropriately based on type
	var valStr string
	switch v := val.(type) {
	case int:
		valStr = fmt.Sprintf("%d", v)
	case float64:
		valStr = fmt.Sprintf("%.3f", v)
	default:
		valStr = fmt.Sprintf("%v", v)
	}

	fmt.Printf("   %-8s |   %3d   |  %3d  |  %6.2f%%  |    %6.2f%%    |  %6.2f%%  |   %6.2f%%   |        %6.2f        |    %3d    |   %3d\n",
		valStr, result.CorrectPredictions, result.TotalPredictions, accuracy, avgHomeWin, avgDraw, avgAwayWin, avgScoreInaccuracy, result.PredictedMatches, result.SkippedMatches)
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
		match.PoissonPredictedHomeGoals = -1
		match.PoissonPredictedAwayGoals = -1
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

		// Calculate score inaccuracy if we have predicted scores
		if match.PoissonPredictedHomeGoals != -1 && match.PoissonPredictedAwayGoals != -1 {
			homeGoalDiff := abs(match.ActualHomeGoals - match.PoissonPredictedHomeGoals)
			awayGoalDiff := abs(match.ActualAwayGoals - match.PoissonPredictedAwayGoals)
			scoreInaccuracy := homeGoalDiff + awayGoalDiff
			
			result.TotalScoreInaccuracy += scoreInaccuracy
			result.ScoreInaccuracyCount++
		}

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

// GetAverageScoreInaccuracy returns the average score inaccuracy
func (pr *PredictionResult) GetAverageScoreInaccuracy() float64 {
	if pr.ScoreInaccuracyCount == 0 {
		return 0.0
	}
	return float64(pr.TotalScoreInaccuracy) / float64(pr.ScoreInaccuracyCount)
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// createConfigSetter creates a setter function for a parameter using reflection
func createConfigSetter(param TuningParam) (func(any), error) {
	if param.FunctionCall != "" {
		// Handle function calls
		return createFunctionSetter(param.FunctionCall)
	} else if param.ConfigPath != "" {
		// Handle direct field access
		return createFieldSetter(param.ConfigPath)
	}
	return nil, fmt.Errorf("parameter %s must specify either ConfigPath or FunctionCall", param.Name)
}

// createFunctionSetter creates a setter that calls a function in the podds package
func createFunctionSetter(functionName string) (func(any), error) {
	// For function calls, we need to handle them specifically since reflection
	// can't easily call package-level functions by name
	switch functionName {
	case "SetFormWeight":
		return func(val any) {
			if v, ok := val.(float64); ok {
				podds.SetFormWeight(v)
			}
		}, nil
	default:
		return nil, fmt.Errorf("unknown function: %s", functionName)
	}
}

// createFieldSetter creates a setter that directly sets a config field using reflection
func createFieldSetter(configPath string) (func(any), error) {
	return func(val any) {
		// Parse the config path (e.g., "Config.DixonColesRho")
		if len(configPath) < 7 || configPath[:7] != "Config." {
			fmt.Printf("Warning: Invalid config path format: %s\n", configPath)
			return
		}

		fieldName := configPath[7:] // Remove "Config." prefix

		// Get the config struct - Config is already a pointer, so we just need to dereference it
		configValue := reflect.ValueOf(podds.Config).Elem()

		// Get the field
		fieldValue := configValue.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			fmt.Printf("Warning: Field %s not found in Config\n", fieldName)
			return
		}

		if !fieldValue.CanSet() {
			fmt.Printf("Warning: Field %s cannot be set\n", fieldName)
			return
		}

		// Convert and set the value
		valReflect := reflect.ValueOf(val)
		if fieldValue.Type() == valReflect.Type() {
			fieldValue.Set(valReflect)
		} else if valReflect.CanConvert(fieldValue.Type()) {
			fieldValue.Set(valReflect.Convert(fieldValue.Type()))
		} else {
			fmt.Printf("Warning: Cannot convert %v to %s for field %s\n", val, fieldValue.Type(), fieldName)
		}
	}, nil
}
