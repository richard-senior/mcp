package test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

// AdvancedTuningParam defines a parameter to tune with enhanced metadata
type AdvancedTuningParam struct {
	Name         string      // Display name for the parameter
	ConfigPath   string      // Path to the config field (e.g., "Config.DixonColesRho")
	FunctionCall string      // Optional: function name to call instead of direct field access
	Values       []any       // Values to test
	Description  string      // Description of what this parameter does
	DefaultValue any         // Default value to restore after testing
	ValueType    reflect.Type // Expected type for validation
}

// TuningResults holds comprehensive results for a parameter tuning session
type TuningResults struct {
	ParameterName string
	BestValue     any
	BestAccuracy  float64
	AllResults    []ParameterResult
}

// ParameterResult holds results for a single parameter value test
type ParameterResult struct {
	Value            any
	Accuracy         float64
	CorrectPredictions int
	TotalPredictions   int
	AvgHomeWinProb     float64
	AvgDrawProb        float64
	AvgAwayWinProb     float64
}

// AdvancedTuningConfig holds configuration for the tuning process
type AdvancedTuningConfig struct {
	LeagueID           int
	Season             string
	TestAllParameters  bool // If true, test all parameters; if false, test only first
	SaveResults        bool // If true, save results to file
	VerboseOutput      bool // If true, show detailed output
	RestoreDefaults    bool // If true, restore default values after testing
}

var advancedParams = []AdvancedTuningParam{
	{
		Name:         "formWeight",
		FunctionCall: "SetFormWeight",
		Values:       []any{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9},
		Description:  "Weight given to recent form vs overall statistics in predictions",
		DefaultValue: 0.3,
		ValueType:    reflect.TypeOf(float64(0)),
	},
	{
		Name:         "dixonColesRho",
		ConfigPath:   "Config.DixonColesRho",
		Values:       []any{-0.01, -0.02, -0.03, -0.04, -0.05, -0.06, -0.07, -0.08},
		Description:  "Dixon-Coles correlation parameter for low-scoring games",
		DefaultValue: -0.03,
		ValueType:    reflect.TypeOf(float64(0)),
	},
	{
		Name:         "derbyBoostMultiplier",
		ConfigPath:   "Config.DerbyBoostMultiplier",
		Values:       []any{1.00, 1.02, 1.04, 1.06, 1.08, 1.10, 1.12, 1.15, 1.20},
		Description:  "Multiplier for expected goals in derby matches (local rivalries)",
		DefaultValue: 1.08,
		ValueType:    reflect.TypeOf(float64(0)),
	},
	{
		Name:         "poissonSimulations",
		ConfigPath:   "Config.PoissonSimulations",
		Values:       []any{50000, 75000, 100000, 125000, 150000},
		Description:  "Number of Monte Carlo simulations for Poisson distribution",
		DefaultValue: 100000,
		ValueType:    reflect.TypeOf(int(0)),
	},
	{
		Name:         "maxGoalsCap",
		ConfigPath:   "Config.MaxGoalsCap",
		Values:       []any{8.0, 10.0, 12.0, 15.0, 20.0},
		Description:  "Maximum expected goals cap to prevent unrealistic predictions",
		DefaultValue: 10.0,
		ValueType:    reflect.TypeOf(float64(0)),
	},
	{
		Name:         "derbyDistanceThreshold",
		ConfigPath:   "Config.DerbyDistanceThreshold",
		Values:       []any{5, 10, 15, 20, 25},
		Description:  "Distance threshold in miles for considering a match a derby",
		DefaultValue: 10,
		ValueType:    reflect.TypeOf(int(0)),
	},
	{
		Name:         "longTravelPenalty",
		ConfigPath:   "Config.LongTravelPenalty",
		Values:       []any{0.88, 0.90, 0.92, 0.94, 0.96, 0.98, 1.00},
		Description:  "Penalty multiplier for away teams traveling long distances",
		DefaultValue: 0.92,
		ValueType:    reflect.TypeOf(float64(0)),
	},
}

func TestAdvancedTuning(t *testing.T) {
	config := AdvancedTuningConfig{
		LeagueID:          47,
		Season:            "2024/2025",
		TestAllParameters: false, // Set to true to test all parameters
		SaveResults:       false,
		VerboseOutput:     true,
		RestoreDefaults:   true,
	}

	// Load and prepare data
	matches, teamStats, err := loadTuningData(config.LeagueID, config.Season)
	if err != nil {
		t.Fatalf("Failed to load tuning data: %v", err)
	}

	fmt.Printf("🎯 Advanced Parameter Tuning for %s Season %s\n", getLeagueName(config.LeagueID), config.Season)
	fmt.Printf("📊 Loaded %d matches with results\n", len(matches))
	fmt.Printf("📈 Generated %d team statistics entries\n", len(teamStats))
	fmt.Printf("\n")

	var allResults []TuningResults

	// Test parameters
	parametersToTest := advancedParams
	if !config.TestAllParameters {
		parametersToTest = advancedParams[:1] // Test only first parameter
	}

	for i, param := range parametersToTest {
		fmt.Printf("🔧 Testing Parameter %d/%d: %s\n", i+1, len(parametersToTest), param.Name)
		fmt.Printf("📝 Description: %s\n", param.Description)
		fmt.Printf("🎯 Default Value: %v\n", param.DefaultValue)
		fmt.Printf("\n")

		results, err := testParameter(param, matches, teamStats, config.VerboseOutput)
		if err != nil {
			fmt.Printf("❌ Error testing parameter %s: %v\n", param.Name, err)
			continue
		}

		allResults = append(allResults, results)

		fmt.Printf("✅ Best %s: %v (Accuracy: %.2f%%)\n", param.Name, results.BestValue, results.BestAccuracy)
		fmt.Printf("\n")

		// Restore default value if requested
		if config.RestoreDefaults {
			restoreDefaultValue(param)
		}
	}

	// Print summary
	printTuningSummary(allResults)
}

func loadTuningData(leagueID int, season string) ([]*podds.Match, []*podds.TeamStats, error) {
	matchesMap, err := podds.LoadExistingMatches(leagueID, season)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load matches: %w", err)
	}

	// Convert map to slice and filter completed matches
	var matches []*podds.Match
	for _, match := range matchesMap {
		if match.ActualAwayGoals != -1 && match.ActualHomeGoals != -1 {
			matches = append(matches, match)
		}
	}

	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("no completed matches found for League %d, Season %s", leagueID, season)
	}

	// Get processed TeamStats
	teamStats, err := podds.ProcessTeamStats(matches, leagueID, season)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process team stats: %w", err)
	}

	return matches, teamStats, nil
}

func testParameter(param AdvancedTuningParam, matches []*podds.Match, teamStats []*podds.TeamStats, verbose bool) (TuningResults, error) {
	setter, err := createAdvancedConfigSetter(param)
	if err != nil {
		return TuningResults{}, fmt.Errorf("failed to create setter: %w", err)
	}

	results := TuningResults{
		ParameterName: param.Name,
		AllResults:    make([]ParameterResult, 0, len(param.Values)),
	}

	if verbose {
		printParameterHeader(param.Name)
	}

	for _, value := range param.Values {
		// Validate value type
		if !isValidValueType(value, param.ValueType) {
			fmt.Printf("⚠️  Warning: Value %v has incorrect type for parameter %s\n", value, param.Name)
			continue
		}

		// Set the parameter value
		setter(value)

		// Run predictions and calculate accuracy
		paramResult := runParameterTest(value, matches, teamStats)
		results.AllResults = append(results.AllResults, paramResult)

		// Track best result
		if paramResult.Accuracy > results.BestAccuracy {
			results.BestAccuracy = paramResult.Accuracy
			results.BestValue = value
		}

		if verbose {
			printParameterResult(paramResult)
		}
	}

	if verbose {
		printParameterFooter(param.Name, results.BestValue, results.BestAccuracy)
	}

	return results, nil
}

func createAdvancedConfigSetter(param AdvancedTuningParam) (func(any), error) {
	if param.FunctionCall != "" {
		return createAdvancedFunctionSetter(param.FunctionCall)
	} else if param.ConfigPath != "" {
		return createAdvancedFieldSetter(param.ConfigPath)
	}
	return nil, fmt.Errorf("parameter %s must specify either ConfigPath or FunctionCall", param.Name)
}

func createAdvancedFunctionSetter(functionName string) (func(any), error) {
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

func createAdvancedFieldSetter(configPath string) (func(any), error) {
	return func(val any) {
		if len(configPath) < 7 || configPath[:7] != "Config." {
			return
		}

		fieldName := configPath[7:]
		configValue := reflect.ValueOf(&podds.Config).Elem()
		fieldValue := configValue.FieldByName(fieldName)

		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			return
		}

		valReflect := reflect.ValueOf(val)
		if fieldValue.Type() == valReflect.Type() {
			fieldValue.Set(valReflect)
		} else if valReflect.CanConvert(fieldValue.Type()) {
			fieldValue.Set(valReflect.Convert(fieldValue.Type()))
		}
	}, nil
}

func runParameterTest(value any, matches []*podds.Match, teamStats []*podds.TeamStats) ParameterResult {
	result := ParameterResult{
		Value: value,
	}

	var totalHomeWinProb, totalDrawProb, totalAwayWinProb float64

	for _, match := range matches {
		// Clear existing predictions
		clearMatchPredictions(match)

		// Predict the match
		err := podds.PredictMatch(match, teamStats)
		if err != nil {
			continue
		}

		// Check if prediction was made
		if match.PoissonHomeWinProbability == -1.0 {
			continue
		}

		result.TotalPredictions++
		totalHomeWinProb += match.PoissonHomeWinProbability
		totalDrawProb += match.PoissonDrawProbability
		totalAwayWinProb += match.PoissonAwayWinProbability

		// Check accuracy
		actualResult := getActualResult(match)
		predictedResult := getPredictedResult(match)

		if actualResult == predictedResult {
			result.CorrectPredictions++
		}
	}

	if result.TotalPredictions > 0 {
		result.Accuracy = float64(result.CorrectPredictions) / float64(result.TotalPredictions) * 100
		result.AvgHomeWinProb = totalHomeWinProb / float64(result.TotalPredictions)
		result.AvgDrawProb = totalDrawProb / float64(result.TotalPredictions)
		result.AvgAwayWinProb = totalAwayWinProb / float64(result.TotalPredictions)
	}

	return result
}

func clearMatchPredictions(match *podds.Match) {
	match.PoissonPredictedHomeGoals = -1
	match.PoissonPredictedAwayGoals = -1
	match.PoissonHomeWinProbability = -1.0
	match.PoissonDrawProbability = -1.0
	match.PoissonAwayWinProbability = -1.0
	match.Over1p5Goals = -1.0
	match.Over2p5Goals = -1.0
}

func getActualResult(match *podds.Match) string {
	if match.ActualHomeGoals > match.ActualAwayGoals {
		return "H"
	} else if match.ActualHomeGoals < match.ActualAwayGoals {
		return "A"
	}
	return "D"
}

func getPredictedResult(match *podds.Match) string {
	if match.PoissonHomeWinProbability > match.PoissonDrawProbability && match.PoissonHomeWinProbability > match.PoissonAwayWinProbability {
		return "H"
	} else if match.PoissonAwayWinProbability > match.PoissonDrawProbability && match.PoissonAwayWinProbability > match.PoissonHomeWinProbability {
		return "A"
	}
	return "D"
}

func isValidValueType(value any, expectedType reflect.Type) bool {
	valueType := reflect.TypeOf(value)
	return valueType == expectedType || valueType.ConvertibleTo(expectedType)
}

func restoreDefaultValue(param AdvancedTuningParam) {
	setter, err := createAdvancedConfigSetter(param)
	if err != nil {
		return
	}
	setter(param.DefaultValue)
}

func getLeagueName(leagueID int) string {
	switch leagueID {
	case 47:
		return "Premier League"
	case 48:
		return "Championship"
	case 108:
		return "League One"
	case 109:
		return "League Two"
	default:
		return fmt.Sprintf("League %d", leagueID)
	}
}

func printParameterHeader(paramName string) {
	fmt.Printf("%-20s | Correct | Total | Accuracy | Avg Home Win | Avg Draw | Avg Away Win\n", paramName)
	fmt.Printf("---------------------|---------|-------|----------|--------------|----------|-------------\n")
}

func printParameterResult(result ParameterResult) {
	fmt.Printf("%-20v |   %3d   |  %3d  |  %6.2f%%  |    %6.2f%%    |  %6.2f%%  |   %6.2f%%\n",
		result.Value, result.CorrectPredictions, result.TotalPredictions, result.Accuracy,
		result.AvgHomeWinProb, result.AvgDrawProb, result.AvgAwayWinProb)
}

func printParameterFooter(paramName string, bestValue any, bestAccuracy float64) {
	fmt.Printf("\n🏆 Best %s: %v with accuracy: %.2f%%\n\n", paramName, bestValue, bestAccuracy)
}

func printTuningSummary(allResults []TuningResults) {
	fmt.Printf("📋 TUNING SUMMARY\n")
	fmt.Printf("=================\n")
	for _, result := range allResults {
		fmt.Printf("🔧 %-20s: %v (%.2f%% accuracy)\n", result.ParameterName, result.BestValue, result.BestAccuracy)
	}
	fmt.Printf("\n")
}
