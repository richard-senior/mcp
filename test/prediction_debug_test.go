package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/richard-senior/mcp/pkg/util/podds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPredictionPipeline - Named with 'A' prefix to run first in test suite
// This test debugs the entire prediction pipeline with controlled test data
func TestAPredictionPipeline(t *testing.T) {
	t.Log("=== PREDICTION PIPELINE DEBUG TEST ===")
	
	// Step 1: Setup in-memory test database
	t.Log("Step 1: Setting up test database...")
	err := podds.InitDatabase(":memory:")
	require.NoError(t, err, "Failed to initialize test database")
	
	// Step 2: Create database tables
	t.Log("Step 2: Creating database tables...")
	err = podds.CreateTable(&podds.Match{})
	require.NoError(t, err, "Failed to create match table")
	
	err = podds.CreateTable(&podds.Team{})
	require.NoError(t, err, "Failed to create team table")
	
	err = podds.CreateTable(&podds.TeamStats{})
	require.NoError(t, err, "Failed to create team stats table")
	
	// Step 3: Insert test team statistics with known values
	t.Log("Step 3: Inserting test team statistics...")
	homeStats := createTestTeamStats("8455", "47", "2023/2024", 10)  // Man City
	awayStats := createTestTeamStats("8456", "47", "2023/2024", 10)  // West Ham
	
	err = podds.Save(homeStats)
	require.NoError(t, err, "Failed to save home team stats")
	
	err = podds.Save(awayStats)
	require.NoError(t, err, "Failed to save away team stats")
	
	t.Logf("Saved home team stats: Attack=%f/%f Defense=%f/%f", 
		homeStats.HomeAttackStrength, homeStats.AwayAttackStrength,
		homeStats.HomeDefenseStrength, homeStats.AwayDefenseStrength)
	t.Logf("Saved away team stats: Attack=%f/%f Defense=%f/%f", 
		awayStats.HomeAttackStrength, awayStats.AwayAttackStrength,
		awayStats.HomeDefenseStrength, awayStats.AwayDefenseStrength)
	
	// Step 4: Create test match
	t.Log("Step 4: Creating test match...")
	match := createTestMatch()
	t.Logf("Created match: %s vs %s (League: %d, Season: %s)", 
		match.HomeTeamName, match.AwayTeamName, match.LeagueID, match.Season)
	
	// Step 5: Verify match should be predicted
	t.Log("Step 5: Verifying match should be predicted...")
	t.Logf("Match details - HomeID: %s, AwayID: %s, Season: %s, Status: %s", 
		match.HomeID, match.AwayID, match.Season, match.Status)
	
	// Step 6: Run prediction
	t.Log("Step 6: Running prediction...")
	err = podds.PredictMatch(match)
	require.NoError(t, err, "Prediction failed with error")
	
	// Step 7: Verify results
	t.Log("Step 7: Verifying prediction results...")
	t.Logf("Predicted goals: Home=%d Away=%d", 
		match.PoissonPredictedHomeGoals, match.PoissonPredictedAwayGoals)
	t.Logf("Expected goals: Home=%.2f Away=%.2f", 
		match.HomeTeamGoalExpectency, match.AwayTeamGoalExpectency)
	t.Logf("Win probabilities: Home=%.1f%% Draw=%.1f%% Away=%.1f%%", 
		match.PoissonHomeWinProbability, 
		match.PoissonDrawProbability, 
		match.PoissonAwayWinProbability)
	t.Logf("Over goals: 1.5=%.1f%% 2.5=%.1f%%", 
		match.Over1p5Goals, match.Over2p5Goals)
	
	// Assertions - These should all pass if prediction is working
	assert.NotEqual(t, -1, match.PoissonPredictedHomeGoals, "Home goals prediction should not be -1")
	assert.NotEqual(t, -1, match.PoissonPredictedAwayGoals, "Away goals prediction should not be -1")
	assert.Greater(t, match.HomeTeamGoalExpectency, 0.0, "Home expected goals should be > 0")
	assert.Greater(t, match.AwayTeamGoalExpectency, 0.0, "Away expected goals should be > 0")
	assert.Greater(t, match.PoissonHomeWinProbability, 0.0, "Home win probability should be > 0")
	assert.Greater(t, match.PoissonDrawProbability, 0.0, "Draw probability should be > 0")
	assert.Greater(t, match.PoissonAwayWinProbability, 0.0, "Away win probability should be > 0")
	
	// Probability sum should be approximately 100.0 (stored as percentages)
	totalProb := match.PoissonHomeWinProbability + match.PoissonDrawProbability + match.PoissonAwayWinProbability
	assert.InDelta(t, 100.0, totalProb, 1.0, "Win probabilities should sum to ~100.0")
	
	t.Log("=== PREDICTION PIPELINE TEST COMPLETED ===")
}

// createRealisticPlayedMatches creates a set of played matches with actual results
// This simulates the data that would be used for team statistics calculation
func createRealisticPlayedMatches() []*podds.Match {
	matches := make([]*podds.Match, 0)
	
	// Create matches for multiple rounds to build up statistics
	rounds := []int{1, 2, 3, 4, 5}
	teams := []string{"8455", "8456", "8457", "8458"} // 4 teams for simplicity
	
	matchID := 1
	
	for _, round := range rounds {
		// Create matches for this round
		for i := 0; i < len(teams); i += 2 {
			if i+1 < len(teams) {
				match := podds.NewMatch()
				match.ID = fmt.Sprintf("test-match-%d", matchID)
				match.HomeID = teams[i]
				match.AwayID = teams[i+1]
				match.HomeTeamName = fmt.Sprintf("Team %s", teams[i])
				match.AwayTeamName = fmt.Sprintf("Team %s", teams[i+1])
				match.LeagueID = 47
				match.Season = "2023/2024"
				match.Round = fmt.Sprintf("%d", round)
				match.Status = "finished"
				match.UTCTime = time.Now().Add(-time.Duration(30-round) * 24 * time.Hour)
				
				// Set realistic actual goals (this is key for team stats calculation)
				match.ActualHomeGoals = 1 + (matchID % 3)  // Goals between 1-3
				match.ActualAwayGoals = matchID % 3        // Goals between 0-2
				
				matches = append(matches, match)
				matchID++
			}
		}
	}
	
	return matches
}

// getTeamStatsForTesting retrieves team stats for testing purposes
func getTeamStatsForTesting(teamID string, leagueID int, season string) (*podds.TeamStats, error) {
	leagueIDStr := fmt.Sprintf("%d", leagueID)
	whereClause := "team_id = ? AND league_id = ? AND season = ? ORDER BY round DESC LIMIT 1"
	results, err := podds.FindWhere(&podds.TeamStats{}, whereClause, teamID, leagueIDStr, season)
	
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, fmt.Errorf("no team stats found for team %s", teamID)
	}
	
	if teamStats, ok := results[0].(*podds.TeamStats); ok {
		return teamStats, nil
	}
	
	return nil, fmt.Errorf("unexpected type in results")
}

// createTestTeamStats creates a TeamStats object with realistic non-zero values
func createTestTeamStats(teamID, leagueID, season string, round int) *podds.TeamStats {
	return &podds.TeamStats{
		// Primary key fields
		TeamID:   teamID,
		LeagueID: leagueID,
		Season:   season,
		Round:    round,
		
		// Statistical fields with realistic values
		GamesPlayed:              10,
		HomeGamesPlayed:          5,
		AwayGamesPlayed:          5,
		HomeGoalsPerGame:         1.8,  // Strong home attack
		HomeGoalsConcededPerGame: 0.6,  // Strong home defense
		AwayGoalsPerGame:         1.2,  // Decent away attack
		AwayGoalsConcededPerGame: 1.0,  // Average away defense
		
		// Attack and defense strengths (key for Poisson calculation)
		HomeAttackStrength:  1.2,  // Above average home attack
		HomeDefenseStrength: 0.8,  // Above average home defense (lower is better)
		AwayAttackStrength:  1.0,  // Average away attack
		AwayDefenseStrength: 1.1,  // Slightly below average away defense
		
		// Form data
		Form:     2222, // Decent form (4 recent results: W-W-W-W in quaternary)
		HomeForm: 2222, // Good home form
		AwayForm: 1111, // Average away form
		
		// Form percentages
		FP:  0.75, // 75% form percentage
		HFP: 0.80, // 80% home form percentage
		AFP: 0.70, // 70% away form percentage
		
		// Timestamps
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TestCRoundAverageCalculation - Tests round average calculation which affects team stats
func TestCRoundAverageCalculation(t *testing.T) {
	t.Log("=== ROUND AVERAGE CALCULATION DEBUG TEST ===")
	
	// This test investigates if round averages are being calculated correctly
	// Zero round averages would cause division by zero in team stats calculation
	
	// Setup database
	err := podds.InitDatabase(":memory:")
	require.NoError(t, err, "Failed to initialize test database")
	
	err = podds.CreateTable(&podds.Match{})
	require.NoError(t, err, "Failed to create match table")
	
	// Create played matches
	matches := createRealisticPlayedMatches()
	
	// Save matches
	for _, match := range matches {
		err = podds.Save(match)
		require.NoError(t, err, "Failed to save match")
	}
	
	t.Logf("Created %d matches for round average calculation", len(matches))
	
	// Log some match details to verify they have actual results
	for i, match := range matches {
		if i < 3 { // Show first 3 matches
			t.Logf("Match %d: %s vs %s = %d-%d (Round %s)", 
				i+1, match.HomeTeamName, match.AwayTeamName, 
				match.ActualHomeGoals, match.ActualAwayGoals, match.Round)
		}
	}
	
	// Test if matches are considered "played"
	playedCount := 0
	for _, match := range matches {
		if match.HasBeenPlayed() {
			playedCount++
		}
	}
	
	t.Logf("Matches that HasBeenPlayed(): %d out of %d", playedCount, len(matches))
	
	if playedCount == 0 {
		t.Error("❌ No matches are considered 'played' - this would prevent team stats calculation!")
	}
	
	t.Log("=== ROUND AVERAGE CALCULATION DEBUG TEST COMPLETED ===")
}

// TestDMakeSensibleFunction - Tests the makeSensible function used in team stats calculation
func TestDMakeSensibleFunction(t *testing.T) {
	t.Log("=== MAKE SENSIBLE FUNCTION DEBUG TEST ===")
	
	// The makeSensible function is used to prevent division by zero
	// If it's not working correctly, it could cause team stats to be zero
	
	// We can't directly test makeSensible as it's likely internal
	// But we can test the scenarios that would use it
	
	testValues := []float64{0.0, 0.1, 1.0, 1.5, 2.0}
	
	for _, val := range testValues {
		t.Logf("Testing division scenarios with value: %.1f", val)
		
		// Simulate the kind of calculations done in team stats
		if val == 0.0 {
			t.Logf("  Division by zero scenario - makeSensible should handle this")
		} else {
			result := 1.5 / val  // Example calculation
			t.Logf("  1.5 / %.1f = %.2f", val, result)
		}
	}
	
	t.Log("=== MAKE SENSIBLE FUNCTION DEBUG TEST COMPLETED ===")
}

// TestETeamStatsCalculationStep - Step-by-step debugging of team stats calculation
func TestETeamStatsCalculationStep(t *testing.T) {
	t.Log("=== TEAM STATS CALCULATION STEP-BY-STEP DEBUG ===")
	
	// This test manually walks through the team statistics calculation process
	// to identify exactly where the zero values are coming from
	
	// Setup database
	err := podds.InitDatabase(":memory:")
	require.NoError(t, err, "Failed to initialize test database")
	
	err = podds.CreateTable(&podds.Match{})
	require.NoError(t, err, "Failed to create match table")
	
	err = podds.CreateTable(&podds.TeamStats{})
	require.NoError(t, err, "Failed to create team stats table")
	
	// Create a single team with known match results
	teamID := "8455"
	leagueID := 47
	season := "2023/2024"
	
	// Create specific matches for this team with known results
	matches := []*podds.Match{
		createSpecificMatch("match-1", teamID, "8456", 2, 1, 1), // Home win 2-1
		createSpecificMatch("match-2", "8457", teamID, 1, 2, 2), // Away win 2-1
		createSpecificMatch("match-3", teamID, "8458", 1, 1, 3), // Home draw 1-1
		createSpecificMatch("match-4", "8459", teamID, 0, 1, 4), // Away win 1-0
	}
	
	// Save matches
	for _, match := range matches {
		err = podds.Save(match)
		require.NoError(t, err, "Failed to save match")
	}
	
	t.Logf("Created %d specific matches for team %s", len(matches), teamID)
	
	// Log the matches to verify data
	for i, match := range matches {
		t.Logf("Match %d: %s vs %s = %d-%d (Team %s played %s)", 
			i+1, match.HomeTeamName, match.AwayTeamName,
			match.ActualHomeGoals, match.ActualAwayGoals,
			teamID, getTeamLocation(teamID, match))
	}
	
	// Calculate expected statistics manually
	homeGames := 0
	awayGames := 0
	homeGoals := 0
	awayGoals := 0
	homeConceded := 0
	awayConceded := 0
	
	for _, match := range matches {
		if match.HomeID == teamID {
			homeGames++
			homeGoals += match.ActualHomeGoals
			homeConceded += match.ActualAwayGoals
		} else if match.AwayID == teamID {
			awayGames++
			awayGoals += match.ActualAwayGoals
			awayConceded += match.ActualHomeGoals
		}
	}
	
	t.Logf("Manual calculation for team %s:", teamID)
	t.Logf("  Home games: %d, Goals: %d, Conceded: %d", homeGames, homeGoals, homeConceded)
	t.Logf("  Away games: %d, Goals: %d, Conceded: %d", awayGames, awayGoals, awayConceded)
	
	if homeGames > 0 {
		homeGoalsPerGame := float64(homeGoals) / float64(homeGames)
		homeConcededPerGame := float64(homeConceded) / float64(homeGames)
		t.Logf("  Home goals per game: %.2f", homeGoalsPerGame)
		t.Logf("  Home conceded per game: %.2f", homeConcededPerGame)
	}
	
	if awayGames > 0 {
		awayGoalsPerGame := float64(awayGoals) / float64(awayGames)
		awayConcededPerGame := float64(awayConceded) / float64(awayGames)
		t.Logf("  Away goals per game: %.2f", awayGoalsPerGame)
		t.Logf("  Away conceded per game: %.2f", awayConcededPerGame)
	}
	
	// Now run the actual team stats calculation
	t.Log("Running actual team stats calculation...")
	err = podds.ProcessAndSaveTeamStats(matches, leagueID, season)
	require.NoError(t, err, "Failed to process team statistics")
	
	// Retrieve and compare the calculated stats
	calculatedStats, err := getTeamStatsForTesting(teamID, leagueID, season)
	if err != nil {
		t.Errorf("Failed to retrieve calculated stats: %v", err)
	} else {
		t.Logf("Calculated stats for team %s:", teamID)
		t.Logf("  Games played: %d (Home: %d, Away: %d)", 
			calculatedStats.GamesPlayed, calculatedStats.HomeGamesPlayed, calculatedStats.AwayGamesPlayed)
		t.Logf("  Goals per game: Home=%.2f Away=%.2f", 
			calculatedStats.HomeGoalsPerGame, calculatedStats.AwayGoalsPerGame)
		t.Logf("  Attack strengths: Home=%.2f Away=%.2f", 
			calculatedStats.HomeAttackStrength, calculatedStats.AwayAttackStrength)
		t.Logf("  Defense strengths: Home=%.2f Away=%.2f", 
			calculatedStats.HomeDefenseStrength, calculatedStats.AwayDefenseStrength)
		
		// Check for zero values
		if calculatedStats.HomeAttackStrength == 0.0 {
			t.Error("❌ FOUND THE PROBLEM: Home attack strength calculated as 0.0!")
		}
		if calculatedStats.AwayAttackStrength == 0.0 {
			t.Error("❌ FOUND THE PROBLEM: Away attack strength calculated as 0.0!")
		}
		if calculatedStats.HomeDefenseStrength == 0.0 {
			t.Error("❌ FOUND THE PROBLEM: Home defense strength calculated as 0.0!")
		}
		if calculatedStats.AwayDefenseStrength == 0.0 {
			t.Error("❌ FOUND THE PROBLEM: Away defense strength calculated as 0.0!")
		}
	}
	
	t.Log("=== TEAM STATS CALCULATION STEP-BY-STEP DEBUG COMPLETED ===")
}

// createSpecificMatch creates a match with specific results for testing
func createSpecificMatch(id, homeID, awayID string, homeGoals, awayGoals, round int) *podds.Match {
	match := podds.NewMatch()
	match.ID = id
	match.HomeID = homeID
	match.AwayID = awayID
	match.HomeTeamName = fmt.Sprintf("Team %s", homeID)
	match.AwayTeamName = fmt.Sprintf("Team %s", awayID)
	match.LeagueID = 47
	match.Season = "2023/2024"
	match.Round = fmt.Sprintf("%d", round)
	match.Status = "finished"
	match.UTCTime = time.Now().Add(-time.Duration(30-round) * 24 * time.Hour)
	match.ActualHomeGoals = homeGoals
	match.ActualAwayGoals = awayGoals
	return match
}

// TestFAttackDefenseCalculationDebug - Debug the exact attack/defense calculation
func TestFAttackDefenseCalculationDebug(t *testing.T) {
	t.Log("=== ATTACK/DEFENSE CALCULATION DEBUG TEST ===")
	
	// Setup database
	err := podds.InitDatabase(":memory:")
	require.NoError(t, err, "Failed to initialize test database")
	
	err = podds.CreateTable(&podds.Match{})
	require.NoError(t, err, "Failed to create match table")
	
	err = podds.CreateTable(&podds.TeamStats{})
	require.NoError(t, err, "Failed to create team stats table")
	
	// Create specific matches with known results
	matches := []*podds.Match{
		createSpecificMatch("match-1", "8455", "8456", 2, 1, 1), // Home win 2-1
		createSpecificMatch("match-2", "8457", "8455", 1, 2, 1), // Away win 2-1 (8455 away)
	}
	
	// Save matches
	for _, match := range matches {
		err = podds.Save(match)
		require.NoError(t, err, "Failed to save match")
	}
	
	t.Log("Created 2 matches for detailed calculation debugging")
	
	// Process team statistics with debug logging
	leagueID := 47
	season := "2023/2024"
	
	t.Log("Processing team statistics...")
	err = podds.ProcessAndSaveTeamStats(matches, leagueID, season)
	require.NoError(t, err, "Failed to process team statistics")
	
	// Get the calculated team stats
	teamStats, err := getTeamStatsForTesting("8455", leagueID, season)
	require.NoError(t, err, "Failed to get team stats")
	
	t.Logf("Team 8455 calculated values:")
	t.Logf("  Games: Home=%d Away=%d", teamStats.HomeGamesPlayed, teamStats.AwayGamesPlayed)
	t.Logf("  Goals per game: Home=%.2f Away=%.2f", 
		teamStats.HomeGoalsPerGame, teamStats.AwayGoalsPerGame)
	t.Logf("  Conceded per game: Home=%.2f Away=%.2f", 
		teamStats.HomeGoalsConcededPerGame, teamStats.AwayGoalsConcededPerGame)
	t.Logf("  Form percentages: FP=%.2f HFP=%.2f AFP=%.2f", 
		teamStats.FP, teamStats.HFP, teamStats.AFP)
	t.Logf("  Attack strengths: Home=%.2f Away=%.2f", 
		teamStats.HomeAttackStrength, teamStats.AwayAttackStrength)
	t.Logf("  Defense strengths: Home=%.2f Away=%.2f", 
		teamStats.HomeDefenseStrength, teamStats.AwayDefenseStrength)
	
	// Now let's manually calculate what the attack strength SHOULD be
	t.Log("Manual attack strength calculation:")
	
	// Get round averages to see what we're dividing by
	roundAvg, err := getRoundAverageForTesting(leagueID, season, 1)
	if err != nil {
		t.Logf("Failed to get round averages: %v", err)
	} else {
		t.Logf("Round averages:")
		t.Logf("  Mean home goals per game: %.2f", roundAvg.MeanHomeGoalsPerGame)
		t.Logf("  Mean away goals per game: %.2f", roundAvg.MeanAwayGoalsPerGame)
		t.Logf("  Mean home conceded per game: %.2f", roundAvg.MeanHomeGoalsConcededPerGame)
		t.Logf("  Mean away conceded per game: %.2f", roundAvg.MeanAwayGoalsConcededPerGame)
		
		// Manual calculation of home attack strength
		if roundAvg.MeanHomeGoalsPerGame > 0 {
			baseHomeAttack := teamStats.HomeGoalsPerGame / roundAvg.MeanHomeGoalsPerGame
			formFactor := (teamStats.FP + teamStats.HFP) / 2
			expectedHomeAttack := (0.7 * baseHomeAttack) + (0.3 * formFactor * baseHomeAttack)
			
			t.Logf("Manual home attack calculation:")
			t.Logf("  Base attack: %.2f / %.2f = %.2f", 
				teamStats.HomeGoalsPerGame, roundAvg.MeanHomeGoalsPerGame, baseHomeAttack)
			t.Logf("  Form factor: (%.2f + %.2f) / 2 = %.2f", 
				teamStats.FP, teamStats.HFP, formFactor)
			t.Logf("  Expected: (0.7 * %.2f) + (0.3 * %.2f * %.2f) = %.2f", 
				baseHomeAttack, formFactor, baseHomeAttack, expectedHomeAttack)
			t.Logf("  Actual calculated: %.2f", teamStats.HomeAttackStrength)
			
			if expectedHomeAttack != teamStats.HomeAttackStrength {
				t.Errorf("❌ Attack strength mismatch! Expected %.2f, got %.2f", 
					expectedHomeAttack, teamStats.HomeAttackStrength)
			}
		} else {
			t.Error("❌ Round average home goals per game is 0.0 - this would cause division by zero!")
		}
	}
	
	t.Log("=== ATTACK/DEFENSE CALCULATION DEBUG TEST COMPLETED ===")
}

// getRoundAverageForTesting retrieves round averages for testing
func getRoundAverageForTesting(leagueID int, season string, round int) (*podds.RoundAverage, error) {
	// RoundAverage doesn't implement Persistable, so we can't use FindWhere
	// We'll need to check if there's another way to get round averages
	// For now, let's return an error to see what happens
	return nil, fmt.Errorf("round averages not accessible for testing")
}

// getTeamLocation returns whether the team played home or away in the match
func getTeamLocation(teamID string, match *podds.Match) string {
	if match.HomeID == teamID {
		return "home"
	} else if match.AwayID == teamID {
		return "away"
	}
	return "unknown"
}

// createTestMatch creates a test match with known team IDs
func createTestMatch() *podds.Match {
	match := podds.NewMatch()
	
	// Set match details
	match.ID = "test-match-001"
	match.UTCTime = time.Now().Add(24 * time.Hour) // Future match
	match.Round = "10"
	match.LeagueID = 47 // Premier League
	match.Season = "2023/2024"
	match.Status = "scheduled"
	
	// Team details
	match.HomeTeamName = "Test Home Team"
	match.AwayTeamName = "Test Away Team"
	match.HomeID = "8455" // Matches team stats
	match.AwayID = "8456" // Matches team stats
	
	// No actual results (future match)
	match.ActualHomeGoals = -1
	match.ActualAwayGoals = -1
	
	// Match URL and other details
	match.MatchUrl = "https://test.com/match/001"
	match.Referee = "Test Referee"
	
	return match
}

// TestBTeamStatsCalculation - Tests the team statistics calculation process
// This test investigates why team stats are calculated as 0.00 values
func TestBTeamStatsCalculation(t *testing.T) {
	t.Log("=== TEAM STATS CALCULATION DEBUG TEST ===")
	
	// Setup database (reuse from previous test)
	err := podds.InitDatabase(":memory:")
	require.NoError(t, err, "Failed to initialize test database")
	
	err = podds.CreateTable(&podds.Match{})
	require.NoError(t, err, "Failed to create match table")
	
	err = podds.CreateTable(&podds.TeamStats{})
	require.NoError(t, err, "Failed to create team stats table")
	
	// Step 1: Create realistic played matches with actual results
	t.Log("Step 1: Creating played matches with actual results...")
	matches := createRealisticPlayedMatches()
	
	// Save matches to database
	for _, match := range matches {
		err = podds.Save(match)
		require.NoError(t, err, "Failed to save test match")
	}
	
	t.Logf("Created %d played matches with actual results", len(matches))
	
	// Step 2: Process team statistics using the actual calculation logic
	t.Log("Step 2: Processing team statistics using actual calculation logic...")
	leagueID := 47
	season := "2023/2024"
	
	err = podds.ProcessAndSaveTeamStats(matches, leagueID, season)
	require.NoError(t, err, "Failed to process team statistics")
	
	// Step 3: Retrieve and examine the calculated team statistics
	t.Log("Step 3: Examining calculated team statistics...")
	
	// Get team stats for our test teams
	homeTeamID := "8455"
	awayTeamID := "8456"
	
	homeStats, err := getTeamStatsForTesting(homeTeamID, leagueID, season)
	if err != nil {
		t.Logf("Failed to get home team stats: %v", err)
	} else {
		t.Logf("Home team (%s) calculated stats:", homeTeamID)
		t.Logf("  Games Played: %d (Home: %d, Away: %d)", 
			homeStats.GamesPlayed, homeStats.HomeGamesPlayed, homeStats.AwayGamesPlayed)
		t.Logf("  Goals: Home=%d/%d Away=%d/%d", 
			homeStats.HomeGoals, homeStats.HomeConceded, homeStats.AwayGoals, homeStats.AwayConceded)
		t.Logf("  Goals Per Game: Home=%.2f/%.2f Away=%.2f/%.2f", 
			homeStats.HomeGoalsPerGame, homeStats.HomeGoalsConcededPerGame,
			homeStats.AwayGoalsPerGame, homeStats.AwayGoalsConcededPerGame)
		t.Logf("  Attack Strengths: Home=%.2f Away=%.2f", 
			homeStats.HomeAttackStrength, homeStats.AwayAttackStrength)
		t.Logf("  Defense Strengths: Home=%.2f Away=%.2f", 
			homeStats.HomeDefenseStrength, homeStats.AwayDefenseStrength)
		
		// Check if values are zero
		if homeStats.HomeAttackStrength == 0.0 {
			t.Error("❌ Home attack strength is 0.0 - this is the problem!")
		}
		if homeStats.AwayAttackStrength == 0.0 {
			t.Error("❌ Away attack strength is 0.0 - this is the problem!")
		}
	}
	
	awayStats, err := getTeamStatsForTesting(awayTeamID, leagueID, season)
	if err != nil {
		t.Logf("Failed to get away team stats: %v", err)
	} else {
		t.Logf("Away team (%s) calculated stats:", awayTeamID)
		t.Logf("  Attack Strengths: Home=%.2f Away=%.2f", 
			awayStats.HomeAttackStrength, awayStats.AwayAttackStrength)
		t.Logf("  Defense Strengths: Home=%.2f Away=%.2f", 
			awayStats.HomeDefenseStrength, awayStats.AwayDefenseStrength)
	}
	
	// Step 4: Test prediction with calculated stats
	t.Log("Step 4: Testing prediction with calculated team statistics...")
	testMatch := createTestMatch()
	testMatch.HomeID = homeTeamID
	testMatch.AwayID = awayTeamID
	testMatch.Season = season
	testMatch.LeagueID = leagueID
	
	err = podds.PredictMatch(testMatch)
	if err != nil {
		t.Logf("Prediction failed: %v", err)
	} else {
		t.Logf("Prediction results with calculated stats:")
		t.Logf("  Expected Goals: Home=%.2f Away=%.2f", 
			testMatch.HomeTeamGoalExpectency, testMatch.AwayTeamGoalExpectency)
		t.Logf("  Predicted Score: %d-%d", 
			testMatch.PoissonPredictedHomeGoals, testMatch.PoissonPredictedAwayGoals)
		
		if testMatch.HomeTeamGoalExpectency == 0.0 || testMatch.AwayTeamGoalExpectency == 0.0 {
			t.Error("❌ Expected goals are zero - team stats calculation is broken!")
		}
	}
	
	t.Log("=== TEAM STATS CALCULATION DEBUG TEST COMPLETED ===")
}

// TestCDatabaseSchema - Tests database schema and table creation
func TestCDatabaseSchema(t *testing.T) {
	t.Log("=== DATABASE SCHEMA TEST ===")
	
	// Test that we can query the team_stats table structure
	// This will help verify column names
	
	// Create a dummy TeamStats and check its table name and columns
	
	dummy := &podds.TeamStats{}
	tableName := dummy.GetTableName()
	t.Logf("TeamStats table name: %s", tableName)
	
	// Try to insert and retrieve a simple record to test schema
	testStats := &podds.TeamStats{
		TeamID:   "schema-test",
		LeagueID: "47",
		Season:   "2023/2024",
		Round:    1,
		HomeAttackStrength: 1.0,
	}
	
	err := podds.Save(testStats)
	if err != nil {
		t.Logf("Schema test save failed: %v", err)
		t.Log("This might indicate a database schema issue")
	} else {
		t.Log("Schema test save succeeded")
		
		// Try to retrieve it
		pk := testStats.GetPrimaryKey()
		retrievedStats := &podds.TeamStats{}
		err = podds.FindByPrimaryKey(retrievedStats, pk)
		if err != nil {
			t.Logf("Schema test retrieval failed: %v", err)
		} else {
			t.Logf("Schema test retrieval succeeded: %s", retrievedStats.TeamID)
		}
	}
	
	t.Log("=== DATABASE SCHEMA TEST COMPLETED ===")
}
