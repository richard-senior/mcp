package test

import (
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

// TestBTeamStatsRetrieval - Tests team statistics retrieval specifically
func TestBTeamStatsRetrieval(t *testing.T) {
	t.Log("=== TEAM STATS RETRIEVAL TEST ===")
	
	// This test assumes the database is already setup from TestAPredictionPipeline
	// Try to retrieve the team stats we inserted
	
	// Use FindWhere to search for team stats (same as getTeamStats function)
	whereClause := "team_id = ? AND league_id = ? AND season = ? ORDER BY round DESC LIMIT 1"
	results, err := podds.FindWhere(&podds.TeamStats{}, whereClause, "8455", "47", "2023/2024")
	
	if err != nil {
		t.Logf("FindWhere error: %v", err)
		t.Logf("This might indicate the SQL column name issue")
		return
	}
	
	t.Logf("Found %d team stats results", len(results))
	
	if len(results) > 0 {
		if teamStats, ok := results[0].(*podds.TeamStats); ok {
			t.Logf("Retrieved team stats: TeamID=%s Round=%d", teamStats.TeamID, teamStats.Round)
			t.Logf("Attack strengths: Home=%.2f Away=%.2f", 
				teamStats.HomeAttackStrength, teamStats.AwayAttackStrength)
			t.Logf("Defense strengths: Home=%.2f Away=%.2f", 
				teamStats.HomeDefenseStrength, teamStats.AwayDefenseStrength)
		} else {
			t.Log("Type assertion failed - unexpected result type")
		}
	} else {
		t.Log("No team stats found - this indicates the retrieval issue")
	}
	
	t.Log("=== TEAM STATS RETRIEVAL TEST COMPLETED ===")
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
