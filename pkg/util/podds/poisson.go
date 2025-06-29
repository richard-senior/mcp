package podds

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/richard-senior/mcp/internal/logger"
)

// PoissonPredictor handles match prediction using Poisson distribution
type PoissonPredictor struct {
	// We'll add configuration fields here as needed
}

// PoissonResult holds the complete Poisson analysis results
type PoissonResult struct {
	HomeExpectedGoals       float64
	AwayExpectedGoals       float64
	PredictedHomeGoals      int
	PredictedAwayGoals      int
	HomeWinProbability      float64
	DrawProbability         float64
	AwayWinProbability      float64
	Over1p5GoalsProbability float64
	Over2p5GoalsProbability float64
}

// PredictMatch calculates Poisson-based predictions for a match
// Uses centralized configuration from Config for all parameters
func PredictMatch(match *Match, teamStats []*TeamStats) error {

	// Only predict in certain circumstances
	// Prevents reprediction after the fact (after the result is known) which skews
	// accuracy statistics
	// unless we're being invoked by a unit test
	if !testing.Testing() && !shouldPredict(match) {
		return nil
	}

	var err error
	var homeStats *TeamStats
	var awayStats *TeamStats
	var hf = false
	var af = false
	// iterate teamStats and find the team stats out of the array for the match home and away teams
	// if they're not present then try and look them up in the db

	for _, teamStat := range teamStats {
		if hf && af {
			break
		}
		if teamStat.TeamID == match.HomeID {
			homeStats = teamStat
			hf = true
		} else if teamStat.TeamID == match.AwayID {
			awayStats = teamStat
			af = true
		}
	}
	if homeStats == nil || awayStats == nil {
		homeStats, err = getTeamStatsFromDb(match.HomeID, match.LeagueID, match.Season)
		if err != nil {
			logger.Warn("Could not get home team stats for prediction", match.HomeTeamName, err)
			return err
		}

		awayStats, err = getTeamStatsFromDb(match.AwayID, match.LeagueID, match.Season)
		if err != nil {
			logger.Warn("Could not get away team stats for prediction", match.AwayTeamName, err)
			return err
		}
	}

	return DoPredictMatch(match, homeStats, awayStats)

}

// DoPredictMatch calculates Poisson-based predictions for a match
// Uses centralized configuration from Config for all parameters
// amends the passed Match Instance with prediction data
func DoPredictMatch(match *Match, homeStats *TeamStats, awayStats *TeamStats) error {

	// Only predict in certain circumstances
	// Prevents reprediction after the fact (after the result is known) which skews
	// accuracy statistics
	// unless we're being invoked by a unit test
	if !shouldPredict(match) {
		return nil
	}

	// Calculate Poisson predictions using Monte Carlo simulation with poke adjustments
	result, err := calculatePoissonPrediction(homeStats, awayStats, match)
	if err != nil {
		return err
	}

	// Update match with prediction results
	match.PoissonPredictedHomeGoals = result.PredictedHomeGoals
	match.PoissonPredictedAwayGoals = result.PredictedAwayGoals
	match.HomeTeamGoalExpectency = result.HomeExpectedGoals
	match.AwayTeamGoalExpectency = result.AwayExpectedGoals
	match.PoissonHomeWinProbability = result.HomeWinProbability
	match.PoissonDrawProbability = result.DrawProbability
	match.PoissonAwayWinProbability = result.AwayWinProbability
	match.Over1p5Goals = result.Over1p5GoalsProbability
	match.Over2p5Goals = result.Over2p5GoalsProbability

	return nil
}

// shouldPredict determines if we should make a prediction for this match
// Now simplified since match caching is handled at extraction level
func shouldPredict(match *Match) bool {
	// For current season, apply time-based restrictions
	if match.Season == "" {
		return false
	}

	if testing.Testing() {
		return true
	}

	GetSecondYear(match.Season)
	if match.Season == GetCurrentSeason() {
		// Only predict for future matches that are more than the configured time buffer away
		now := time.Now()
		if match.UTCTime.Before(now) {
			return false
		}

		// Don't predict matches that start within the configured time buffer
		timeBuffer := time.Duration(GetPredictionTimeBuffer()) * time.Minute
		bufferTime := now.Add(timeBuffer)
		if match.UTCTime.Before(bufferTime) {
			return false
		}

		// Only predict for matches that haven't been played yet
		switch match.Status {
		case "finished", "played", "completed", "final", "ended":
			return false
		case "scheduled", "upcoming", "fixture", "not_started", "":
			return true
		default:
			// For unknown statuses, check if goals are already set
			if match.ActualHomeGoals >= 0 && match.ActualAwayGoals >= 0 {
				return false // Match has been played
			}
			return true // Assume it's a future match
		}
	}

	// Check if match already has predictions (from cache or new calculation)
	if match.PoissonPredictedHomeGoals != -1 || match.PoissonPredictedAwayGoals != -1 {
		return false // Already has predictions
	}

	// For historical seasons, predict if no existing prediction
	// This allows us to test our predictions against historical results
	return true
}

// EvaluatePredictionAccuracy compares predictions with actual results for testing
// Returns various accuracy metrics for a completed match
func EvaluatePredictionAccuracy(match *Match) *PredictionAccuracy {
	if match.ActualHomeGoals == -1 || match.ActualAwayGoals == -1 {
		return nil // Match not completed
	}

	if match.PoissonPredictedHomeGoals == -1 || match.PoissonPredictedAwayGoals == -1 {
		return nil // No prediction available
	}

	accuracy := &PredictionAccuracy{
		MatchID:            match.ID,
		HomeTeam:           match.HomeTeamName,
		AwayTeam:           match.AwayTeamName,
		ActualHomeGoals:    match.ActualHomeGoals,
		ActualAwayGoals:    match.ActualAwayGoals,
		PredictedHomeGoals: match.PoissonPredictedHomeGoals,
		PredictedAwayGoals: match.PoissonPredictedAwayGoals,
		Poke:               match.Poke,
	}

	// Calculate exact score accuracy
	accuracy.ExactScoreCorrect = (accuracy.ActualHomeGoals == accuracy.PredictedHomeGoals &&
		accuracy.ActualAwayGoals == accuracy.PredictedAwayGoals)

	// Calculate result accuracy (win/draw/loss)
	actualResult := getMatchResult(accuracy.ActualHomeGoals, accuracy.ActualAwayGoals)
	predictedResult := getMatchResult(accuracy.PredictedHomeGoals, accuracy.PredictedAwayGoals)
	accuracy.ResultCorrect = (actualResult == predictedResult)

	// Calculate goal difference accuracy
	actualGoalDiff := accuracy.ActualHomeGoals - accuracy.ActualAwayGoals
	predictedGoalDiff := accuracy.PredictedHomeGoals - accuracy.PredictedAwayGoals
	accuracy.GoalDifferenceError = abs(actualGoalDiff - predictedGoalDiff)

	// Calculate total goals accuracy
	actualTotalGoals := accuracy.ActualHomeGoals + accuracy.ActualAwayGoals
	predictedTotalGoals := accuracy.PredictedHomeGoals + accuracy.PredictedAwayGoals
	accuracy.TotalGoalsError = abs(actualTotalGoals - predictedTotalGoals)

	return accuracy
}

// PredictionAccuracy holds accuracy metrics for a single match prediction
type PredictionAccuracy struct {
	MatchID             string
	HomeTeam            string
	AwayTeam            string
	ActualHomeGoals     int
	ActualAwayGoals     int
	PredictedHomeGoals  int
	PredictedAwayGoals  int
	Poke                int
	ExactScoreCorrect   bool
	ResultCorrect       bool
	GoalDifferenceError int
	TotalGoalsError     int
}

// getMatchResult returns "H" for home win, "D" for draw, "A" for away win
func getMatchResult(homeGoals, awayGoals int) string {
	if homeGoals > awayGoals {
		return "H"
	} else if homeGoals < awayGoals {
		return "A"
	}
	return "D"
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// EvaluateAllPredictions evaluates prediction accuracy across multiple matches
// Returns aggregate statistics for testing purposes
func EvaluateAllPredictions(matches []*Match) *AggregateAccuracy {
	var accuracies []*PredictionAccuracy

	for _, match := range matches {
		if accuracy := EvaluatePredictionAccuracy(match); accuracy != nil {
			accuracies = append(accuracies, accuracy)
		}
	}

	if len(accuracies) == 0 {
		return nil
	}

	aggregate := &AggregateAccuracy{
		TotalMatches: len(accuracies),
	}

	// Calculate aggregate statistics
	var exactScoreCount, resultCorrectCount int
	var totalGoalDiffError, totalGoalsError int

	for _, acc := range accuracies {
		if acc.ExactScoreCorrect {
			exactScoreCount++
		}
		if acc.ResultCorrect {
			resultCorrectCount++
		}
		totalGoalDiffError += acc.GoalDifferenceError
		totalGoalsError += acc.TotalGoalsError
	}

	aggregate.ExactScoreAccuracy = float64(exactScoreCount) / float64(aggregate.TotalMatches) * 100
	aggregate.ResultAccuracy = float64(resultCorrectCount) / float64(aggregate.TotalMatches) * 100
	aggregate.AverageGoalDiffError = float64(totalGoalDiffError) / float64(aggregate.TotalMatches)
	aggregate.AverageTotalGoalsError = float64(totalGoalsError) / float64(aggregate.TotalMatches)

	return aggregate
}

// AggregateAccuracy holds aggregate prediction accuracy statistics
type AggregateAccuracy struct {
	TotalMatches           int
	ExactScoreAccuracy     float64 // Percentage
	ResultAccuracy         float64 // Percentage
	AverageGoalDiffError   float64
	AverageTotalGoalsError float64
}

// calculatePoissonPrediction performs Monte Carlo simulation using Poisson distribution
// This mirrors the Python numpy approach: np.random.poisson(expectancy, 100000)
// Enhanced with Dixon-Coles correction for low-scoring games and poke (travel distance) adjustments
func calculatePoissonPrediction(homeStats, awayStats *TeamStats, match *Match) (*PoissonResult, error) {
	if homeStats == nil || awayStats == nil || match == nil {
		return nil, fmt.Errorf("Must pass non-null values to this function")
	}
	// Calculate expected goals with poke adjustments
	homeExpectedGoals := calculateExpectedGoalsWithPoke(homeStats, awayStats, match, true)
	awayExpectedGoals := calculateExpectedGoalsWithPoke(awayStats, homeStats, match, false)

	// Generate Poisson samples (equivalent to np.random.poisson)
	homeGoalSamples := generatePoissonSamples(homeExpectedGoals, Config.PoissonSimulations)
	awayGoalSamples := generatePoissonSamples(awayExpectedGoals, Config.PoissonSimulations)

	// Calculate probability distributions for each goal count
	homeProbabilities := calculateGoalProbabilities(homeGoalSamples, Config.PoissonRange)
	awayProbabilities := calculateGoalProbabilities(awayGoalSamples, Config.PoissonRange)

	// Create probability matrix (equivalent to np.outer)
	probabilityMatrix := createProbabilityMatrix(homeProbabilities, awayProbabilities)

	// Apply Dixon-Coles correction for low-scoring games
	correctedMatrix := dixonColesCorrection(probabilityMatrix, homeExpectedGoals, awayExpectedGoals)

	// Calculate win/draw/loss probabilities using corrected matrix
	homeWinProb, drawProb, awayWinProb := calculateMatchOutcomeProbabilities(correctedMatrix)

	// Find most likely goal counts using Dixon-Coles corrected matrix
	predictedHomeGoals := findMostLikelyGoalsFromMatrix(correctedMatrix, true)
	predictedAwayGoals := findMostLikelyGoalsFromMatrix(correctedMatrix, false)

	// Calculate over/under probabilities (using original samples for consistency)
	over1p5Prob := calculateOverGoalsProbability(homeGoalSamples, awayGoalSamples, Config.Over1p5GoalsThreshold)
	over2p5Prob := calculateOverGoalsProbability(homeGoalSamples, awayGoalSamples, Config.Over2p5GoalsThreshold)

	return &PoissonResult{
		HomeExpectedGoals:       homeExpectedGoals,
		AwayExpectedGoals:       awayExpectedGoals,
		PredictedHomeGoals:      predictedHomeGoals,
		PredictedAwayGoals:      predictedAwayGoals,
		HomeWinProbability:      homeWinProb * 100.0,
		DrawProbability:         drawProb * 100.0,
		AwayWinProbability:      awayWinProb * 100.0,
		Over1p5GoalsProbability: over1p5Prob * 100.0,
		Over2p5GoalsProbability: over2p5Prob * 100.0,
	}, nil
}

// generatePoissonSamples generates random samples from Poisson distribution
// Equivalent to numpy's np.random.poisson(lambda, size)
func generatePoissonSamples(lambda float64, size int) []int {
	samples := make([]int, size)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < size; i++ {
		samples[i] = poissonRandom(lambda, rng)
	}

	return samples
}

// poissonRandom generates a single random number from Poisson distribution
// Uses Knuth's algorithm for Poisson random number generation
func poissonRandom(lambda float64, rng *rand.Rand) int {
	if lambda < 30 {
		// Use Knuth's algorithm for small lambda
		L := math.Exp(-lambda)
		k := 0
		p := 1.0

		for p > L {
			k++
			p *= rng.Float64()
		}

		return k - 1
	} else {
		// Use normal approximation for large lambda
		normal := rng.NormFloat64()
		return int(math.Round(lambda + math.Sqrt(lambda)*normal))
	}
}

// calculateGoalProbabilities calculates probability for each goal count
// Equivalent to: [np.sum(samples == i) / len(samples) for i in range(POISSON_RANGE)]
func calculateGoalProbabilities(samples []int, maxGoals int) []float64 {
	probabilities := make([]float64, maxGoals)
	totalSamples := float64(len(samples))

	for goals := 0; goals < maxGoals; goals++ {
		count := 0
		for _, sample := range samples {
			if sample == goals {
				count++
			}
		}
		probabilities[goals] = float64(count) / totalSamples
	}

	return probabilities
}

// createProbabilityMatrix creates outcome probability matrix
// Equivalent to: np.outer(np.array(h), np.array(a))
func createProbabilityMatrix(homeProbs, awayProbs []float64) [][]float64 {
	rows := len(homeProbs)
	cols := len(awayProbs)
	matrix := make([][]float64, rows)

	for i := 0; i < rows; i++ {
		matrix[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			matrix[i][j] = homeProbs[i] * awayProbs[j]
		}
	}

	return matrix
}

// calculateMatchOutcomeProbabilities calculates win/draw/loss probabilities
// Equivalent to Python's triangle calculations: np.tril, np.diag, np.triu
func calculateMatchOutcomeProbabilities(matrix [][]float64) (homeWin, draw, awayWin float64) {
	rows := len(matrix)
	cols := len(matrix[0])

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if i > j {
				homeWin += matrix[i][j] // Home team scores more (lower triangle)
			} else if i == j {
				draw += matrix[i][j] // Equal scores (diagonal)
			} else {
				awayWin += matrix[i][j] // Away team scores more (upper triangle)
			}
		}
	}

	return homeWin, draw, awayWin
}

// findMostLikelyGoals finds the goal count with highest probability
// Equivalent to: np.argmax(probabilities)
func findMostLikelyGoals(probabilities []float64) int {
	maxProb := 0.0
	mostLikely := 0

	for goals, prob := range probabilities {
		if prob > maxProb {
			maxProb = prob
			mostLikely = goals
		}
	}

	return mostLikely
}

// calculateOverGoalsProbability calculates probability of total goals over threshold
func calculateOverGoalsProbability(homeGoals, awayGoals []int, threshold float64) float64 {
	count := 0
	total := len(homeGoals)

	for i := 0; i < total; i++ {
		totalGoals := float64(homeGoals[i] + awayGoals[i])
		if totalGoals > threshold {
			count++
		}
	}

	return float64(count) / float64(total)
}

// getTeamStatsFromDb retrieves team statistics for Poisson calculation
// Gets the most recent team statistics available for the team
func getTeamStatsFromDb(teamID string, leagueID int, season string) (*TeamStats, error) {
	// Convert leagueID to string for TeamStats
	leagueIDStr := strconv.Itoa(leagueID)

	// Find the most recent team stats for this team/league/season
	// Use FindWhere to get all stats for this team and find the latest round
	// Note: Use database column names (with underscores) not struct field names
	whereClause := "team_id = ? AND league_id = ? AND season = ? ORDER BY round DESC LIMIT 1"

	results, err := FindWhere(&TeamStats{}, whereClause, teamID, leagueIDStr, season)
	if err != nil {
		return nil, fmt.Errorf("failed to find team stats for team %s in league %d season %s: %w", teamID, leagueID, season, err)
	}

	if len(results) == 0 {
		logger.Warn("No team stats found for query:", whereClause, "params:", teamID, leagueIDStr, season)
		return nil, fmt.Errorf("no team stats found for team %s in league %d season %s", teamID, leagueID, season)
	}

	// Get the first (most recent) result
	if teamStats, ok := results[0].(*TeamStats); ok {
		return teamStats, nil
	}

	return nil, fmt.Errorf("unexpected type in team stats results for team %s", teamID)
}

// debugTeamStatsAvailability checks what team statistics are available for debugging
func debugTeamStatsAvailability(leagueID int, season string) {
	leagueIDStr := strconv.Itoa(leagueID)

	// Get a sample of team stats for this league/season
	whereClause := "league_id = ? AND season = ? ORDER BY team_id, round DESC LIMIT 10"
	results, err := FindWhere(&TeamStats{}, whereClause, leagueIDStr, season)

	if err != nil {
		logger.Debug("Error checking team stats availability:", err)
		return
	}

	if len(results) == 0 {
		logger.Warn("NO TEAM STATISTICS FOUND for league", leagueID, "season", season)
		return
	}

}

// getLeagueTeamCount returns the number of teams in a league using configuration
func getLeagueTeamCount(leagueID int) int {
	switch leagueID {
	case 47: // Premier League
		return Config.PremierLeagueTeams
	case 48: // Championship
		return Config.ChampionshipTeams
	case 108: // League One
		return Config.LeagueOneTeams
	case 109: // League Two
		return Config.LeagueTwoTeams
	default:
		return Config.DefaultLeagueTeams // Configurable default assumption
	}
}

// calculateExpectedGoals calculates expected goals using Poisson model
// Formula: Expected Goals = (Team Attack Strength × Opposition Defense Weakness)
// Note: League averages are already incorporated into the attack/defense strengths during team stats calculation
func calculateExpectedGoals(attackingTeam, defendingTeam *TeamStats, isHome bool) float64 {
	var attackStrength, defenseStrength float64

	if isHome {
		// Home team attacking
		attackStrength = attackingTeam.HomeAttackStrength
		defenseStrength = defendingTeam.AwayDefenseStrength
	} else {
		// Away team attacking
		attackStrength = attackingTeam.AwayAttackStrength
		defenseStrength = defendingTeam.HomeDefenseStrength
	}

	// Poisson model: Expected Goals = Attack Strength × Defense Weakness
	// The league averages are already baked into the attack/defense strengths
	expectedGoals := attackStrength * defenseStrength

	// Ensure we don't predict negative goals
	if expectedGoals < 0 {
		expectedGoals = 0
	}

	// Cap at reasonable maximum (e.g., 10 goals)
	if expectedGoals > 10 {
		expectedGoals = 10
	}

	return expectedGoals
}

// calculateExpectedGoalsWithPoke calculates expected goals using Poisson model with poke (travel distance) adjustments
//
// Poke Adjustment Strategy:
// 1. Derby Matches (< 10 miles): Both teams get 8% boost - more attacking, higher intensity games
// 2. Away Team Travel Penalty: Graduated penalty based on distance
//   - 50-99 miles: 2% penalty (minimal impact)
//   - 100-199 miles: 4% penalty (2-3 hours travel)
//   - 200-299 miles: 8% penalty (several hours travel)
//   - 300+ miles: 12% penalty (cross-country, overnight stays)
//
// 3. Home teams are unaffected by travel distance (always playing at home)
//
// Formula: Expected Goals = (Base Poisson Calculation) × Derby Boost × Travel Penalty
func calculateExpectedGoalsWithPoke(attackingTeam, defendingTeam *TeamStats, match *Match, isHome bool) float64 {
	// Calculate base expected goals using standard Poisson model
	baseExpectedGoals := calculateExpectedGoals(attackingTeam, defendingTeam, isHome)

	// Apply poke-based adjustments
	adjustedExpectedGoals := applyPokeAdjustments(baseExpectedGoals, match.Poke, isHome)
	return adjustedExpectedGoals
}

// applyPokeAdjustments applies travel distance adjustments to expected goals
// Based on football analysis of travel impact on team performance
func applyPokeAdjustments(baseExpectedGoals float64, poke int, isHome bool) float64 {
	if poke <= 0 {
		// No poke data available, return base calculation
		return baseExpectedGoals
	}

	adjustedGoals := baseExpectedGoals

	// Derby Match Adjustment (configurable distance threshold)
	// Local derbies tend to be more attacking/open games with higher intensity
	// Both teams benefit from increased motivation and crowd atmosphere
	if poke < Config.DerbyDistanceThreshold {
		adjustedGoals *= Config.DerbyBoostMultiplier
	}

	// Long Distance Travel Adjustment (away team disadvantage only)
	// Away teams suffer from travel fatigue, disrupted routines, and less familiar environment
	if !isHome {
		var travelPenalty float64

		switch {
		case poke >= Config.VeryLongTravelThreshold:
			// Very long distance - significant disadvantage
			// Cross-country travel, potential overnight stays, jet lag effects
			travelPenalty = Config.VeryLongTravelPenalty
		case poke >= Config.LongTravelThreshold:
			// Long distance - moderate disadvantage
			// Several hours travel, disrupted preparation
			travelPenalty = Config.LongTravelPenalty
		case poke >= Config.MediumTravelThreshold:
			// Medium distance - small disadvantage
			// 2-3 hours travel, minor disruption
			travelPenalty = Config.MediumTravelPenalty
		case poke >= Config.ShortTravelThreshold:
			// Short-medium distance - minimal impact
			// 1-2 hours travel, very minor effect
			travelPenalty = Config.ShortTravelPenalty
		default:
			// Short distance - no significant impact
			travelPenalty = 1.0 // No penalty
		}

		adjustedGoals *= travelPenalty
	}

	// Ensure we don't predict negative goals
	if adjustedGoals < Config.MinGoalsFloor {
		adjustedGoals = Config.MinGoalsFloor
	}

	// Cap at reasonable maximum
	if adjustedGoals > Config.MaxGoalsCap {
		adjustedGoals = Config.MaxGoalsCap
	}

	return adjustedGoals
}

// getTeamName helper function for logging
func getTeamName(teamStats *TeamStats, isHome bool, match *Match) string {
	if isHome {
		return match.HomeTeamName
	}
	return match.AwayTeamName
}

// Dixon-Coles correction functions
// dixonColesCorrection applies Dixon-Coles adjustment to probability matrix using configuration
func dixonColesCorrection(matrix [][]float64, homeExpected, awayExpected float64) [][]float64 {
	// Dixon-Coles correlation parameter (configurable)
	rho := GetDixonColesRho()

	correctedMatrix := make([][]float64, len(matrix))
	for i := range matrix {
		correctedMatrix[i] = make([]float64, len(matrix[i]))
		copy(correctedMatrix[i], matrix[i])
	}

	// Apply corrections to specific low-scoring combinations
	if len(matrix) > 2 && len(matrix[0]) > 2 {
		// 0-0 correction
		tau00 := calculateTau(0, 0, homeExpected, awayExpected, rho)
		correctedMatrix[0][0] *= tau00

		// 1-0 correction
		tau10 := calculateTau(1, 0, homeExpected, awayExpected, rho)
		correctedMatrix[1][0] *= tau10

		// 0-1 correction
		tau01 := calculateTau(0, 1, homeExpected, awayExpected, rho)
		correctedMatrix[0][1] *= tau01

		// 1-1 correction
		tau11 := calculateTau(1, 1, homeExpected, awayExpected, rho)
		correctedMatrix[1][1] *= tau11
	}

	// Renormalize the matrix to ensure probabilities sum to 1
	return renormalizeMatrix(correctedMatrix)
}

// calculateTau computes the Dixon-Coles correction factor for specific scorelines
func calculateTau(homeGoals, awayGoals int, lambda1, lambda2, rho float64) float64 {
	if homeGoals == 0 && awayGoals == 0 {
		return 1 - lambda1*lambda2*rho
	} else if homeGoals == 0 && awayGoals == 1 {
		return 1 + lambda1*rho
	} else if homeGoals == 1 && awayGoals == 0 {
		return 1 + lambda2*rho
	} else if homeGoals == 1 && awayGoals == 1 {
		return 1 - rho
	}
	return 1.0 // No correction for other scorelines
}

// renormalizeMatrix ensures all probabilities sum to 1 after Dixon-Coles correction
func renormalizeMatrix(matrix [][]float64) [][]float64 {
	total := 0.0

	// Calculate current total
	for i := range matrix {
		for j := range matrix[i] {
			total += matrix[i][j]
		}
	}

	// Normalize if total is valid
	if total > 0 {
		for i := range matrix {
			for j := range matrix[i] {
				matrix[i][j] /= total
			}
		}
	}

	return matrix
}

// findMostLikelyGoalsFromMatrix finds most likely goals using corrected probability matrix
func findMostLikelyGoalsFromMatrix(matrix [][]float64, isHome bool) int {
	maxProb := 0.0
	mostLikely := 0

	if isHome {
		// Sum probabilities for each home goal count
		for homeGoals := 0; homeGoals < len(matrix); homeGoals++ {
			prob := 0.0
			for awayGoals := 0; awayGoals < len(matrix[homeGoals]); awayGoals++ {
				prob += matrix[homeGoals][awayGoals]
			}
			if prob > maxProb {
				maxProb = prob
				mostLikely = homeGoals
			}
		}
	} else {
		// Sum probabilities for each away goal count
		for awayGoals := 0; awayGoals < len(matrix[0]); awayGoals++ {
			prob := 0.0
			for homeGoals := 0; homeGoals < len(matrix); homeGoals++ {
				prob += matrix[homeGoals][awayGoals]
			}
			if prob > maxProb {
				maxProb = prob
				mostLikely = awayGoals
			}
		}
	}

	return mostLikely
}
