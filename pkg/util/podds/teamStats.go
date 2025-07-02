package podds

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/richard-senior/mcp/internal/logger"
)

// TeamStats represents team statistics for a specific season/round with compound primary key
type TeamStats struct {
	// Compound primary key fields
	TeamID   string `json:"teamId" column:"team_id" dbtype:"TEXT NOT NULL" primary:"true" index:"true"`
	Season   string `json:"season" column:"season" dbtype:"TEXT NOT NULL" primary:"true" index:"true"`
	Round    int    `json:"round" column:"round" dbtype:"INTEGER NOT NULL" primary:"true" index:"true"`
	LeagueID string `json:"leagueId" column:"league_id" dbtype:"TEXT NOT NULL" primary:"true" index:"true"`

	// Foreign key reference to Match
	MatchID string `json:"matchId,omitempty" column:"match_id" dbtype:"TEXT" index:"true" fk:"match.id"`

	// Statistical fields
	GamesPlayed     int `json:"gamesPlayed" column:"games_played" dbtype:"INTEGER DEFAULT 0"`
	HomeGamesPlayed int `json:"homeGamesPlayed" column:"home_games_played" dbtype:"INTEGER DEFAULT 0"`
	AwayGamesPlayed int `json:"awayGamesPlayed" column:"away_games_played" dbtype:"INTEGER DEFAULT 0"`

	HomeWins   int `json:"homeWins" column:"home_wins" dbtype:"INTEGER DEFAULT 0"`
	HomeDraws  int `json:"homeDraws" column:"home_draws" dbtype:"INTEGER DEFAULT 0"`
	HomeLosses int `json:"homeLosses" column:"home_losses" dbtype:"INTEGER DEFAULT 0"`
	AwayWins   int `json:"awayWins" column:"away_wins" dbtype:"INTEGER DEFAULT 0"`
	AwayDraws  int `json:"awayDraws" column:"away_draws" dbtype:"INTEGER DEFAULT 0"`
	AwayLosses int `json:"awayLosses" column:"away_losses" dbtype:"INTEGER DEFAULT 0"`

	HomeGoals    int `json:"homeGoals" column:"home_goals" dbtype:"INTEGER DEFAULT 0"`
	HomeConceded int `json:"homeConceded" column:"home_conceded" dbtype:"INTEGER DEFAULT 0"`
	AwayGoals    int `json:"awayGoals" column:"away_goals" dbtype:"INTEGER DEFAULT 0"`
	AwayConceded int `json:"awayConceded" column:"away_conceded" dbtype:"INTEGER DEFAULT 0"`

	// Calculated averages
	HomeGoalsPerGame         float64 `json:"homeGoalsPerGame" column:"home_goals_per_game" dbtype:"REAL DEFAULT 0.0"`
	HomeGoalsConcededPerGame float64 `json:"homeGoalsConcededPerGame" column:"home_goals_conceded_per_game" dbtype:"REAL DEFAULT 0.0"`
	AwayGoalsPerGame         float64 `json:"awayGoalsPerGame" column:"away_goals_per_game" dbtype:"REAL DEFAULT 0.0"`
	AwayGoalsConcededPerGame float64 `json:"awayGoalsConcededPerGame" column:"away_goals_conceded_per_game" dbtype:"REAL DEFAULT 0.0"`

	// Attack and defense strengths
	HomeAttackStrength  float64 `json:"homeAttackStrength" column:"home_attack_strength" dbtype:"REAL DEFAULT 1.0"`
	HomeDefenseStrength float64 `json:"homeDefenseStrength" column:"home_defense_strength" dbtype:"REAL DEFAULT 1.0"`
	AwayAttackStrength  float64 `json:"awayAttackStrength" column:"away_attack_strength" dbtype:"REAL DEFAULT 1.0"`
	AwayDefenseStrength float64 `json:"awayDefenseStrength" column:"away_defense_strength" dbtype:"REAL DEFAULT 1.0"`

	// Form data (encoded as integers using quaternary system)
	Form     int `json:"form" column:"form" dbtype:"INTEGER DEFAULT 0"`
	HomeForm int `json:"homeForm" column:"home_form" dbtype:"INTEGER DEFAULT 0"`
	AwayForm int `json:"awayForm" column:"away_form" dbtype:"INTEGER DEFAULT 0"`

	// Form percentages (calculated from Round Averages)
	FormPercentage     float64 `json:"formPercentage" column:"form_percentage" dbtype:"REAL DEFAULT 0.0"`
	HomeFormPercentage float64 `json:"homeFormPercentage" column:"home_form_percentage" dbtype:"REAL DEFAULT 0.0"`
	AwayFormPercentage float64 `json:"awayFormPercentage" column:"away_form_percentage" dbtype:"REAL DEFAULT 0.0"`

	// Poisson prediction fields (calculated from Round Averages + Form)
	// These correspond to Python fields: fp, hfp, afp
	FP  float64 `json:"fp" column:"fp" dbtype:"REAL DEFAULT 0.0"`   // Overall form percentage (form/maxForm)
	HFP float64 `json:"hfp" column:"hfp" dbtype:"REAL DEFAULT 0.0"` // Home form percentage (homeForm/maxHomeForm)
	AFP float64 `json:"afp" column:"afp" dbtype:"REAL DEFAULT 0.0"` // Away form percentage (awayForm/maxAwayForm)

	// Points and position
	Points          int `json:"points" column:"points" dbtype:"INTEGER DEFAULT 0"`
	Position        int `json:"position" column:"position" dbtype:"INTEGER DEFAULT 0"`
	InitialPosition int `json:"initialposition,omitempty" column:"initialposition" dbtype:"INTEGER DEFAULT 0"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" column:"created_at" dbtype:"DATETIME DEFAULT CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updatedAt" column:"updated_at" dbtype:"DATETIME DEFAULT CURRENT_TIMESTAMP"`
}

/////////////////////////////////////////////////////////////////////////
////// Team Statistics Processing Functions
/////////////////////////////////////////////////////////////////////////

/*
* ProcessAndSaveTeamStats processes matches and generates team statistics
* This method is called when the code is running normally (not in a unit test etc.)
* Makes use of the ProcessTeamStats method which actually does the work.
 */
func ProcessAndSaveTeamStats(matches []*Match, leagueID int, season string) ([]*TeamStats, error) {
	s, err := ProcessTeamStats(matches, leagueID, season)
	if err != nil {
		return nil, err
	}
	// Now save the completed team stats for later use
	for _, teamStat := range s {
		if err := Save(teamStat); err != nil {
			logger.Error("Failed to save updated team stats for team", teamStat.TeamID, "round", teamStat.Round, "error:", err)
			return nil, fmt.Errorf("failed to save updated team stats: %w", err)
		}
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

/*
* ProcessTeamStats processes matches and generates team statistics
* Does not persist the data, only calculates and returns it. This is a good entry
* for unit tests wishing to test this code without having to persist data.
 */
func ProcessTeamStats(matches []*Match, leagueID int, season string) ([]*TeamStats, error) {
	logger.Info("Processing team statistics for league", leagueID, "season", season)

	// Group matches by round
	roundMatches := GroupMatchesByRound(matches)

	// Process each round in order
	rounds := GetSortedRounds(roundMatches)
	ret := []*TeamStats{}

	for _, round := range rounds {
		var err error
		var rs []*TeamStats
		if rs, err = processRoundStats(roundMatches[round], leagueID, season, round); err != nil {
			logger.Error("Failed to process round stats", round, err)
			continue
		}
		// append these round stats to the ret array
		ret = append(ret, rs...)
	}

	logger.Info("Team statistics processed successfully")
	return ret, nil
}

// ProcessRoundStats processes statistics for a specific round
func processRoundStats(matches []*Match, leagueID int, season string, round int) ([]*TeamStats, error) {
	// Get all teams in this round
	teams := GetTeamsFromMatches(matches)
	// keep a record of all stats generated for later postprocessing
	roundStats := []*TeamStats{}
	// Loop over teams and calculate round by round statistics
	for _, teamID := range teams {
		// Get previous round stats for cumulative calculation
		var prevStats *TeamStats
		if round > 1 {
			prevStats = &TeamStats{}
			pk := map[string]any{
				"team_id":   teamID,
				"season":    season,
				"round":     round - 1,
				"league_id": strconv.Itoa(leagueID),
			}
			err := FindByPrimaryKey(prevStats, pk)
			if err != nil {
				logger.Debug("No previous stats found for team", teamID, "round", round-1)
				// Create empty previous stats
				prevStats = &TeamStats{}
			}
		} else {
			// First round, start with empty stats
			prevStats = &TeamStats{}
		}

		// Calculate current round stats
		currentStats := calculateTeamStatsForRound(teamID, matches, prevStats, leagueID, season, round)

		// do extra stuff here

		// First get the initial position of this team in this season if it's available
		ip, err := TData.GetInitialPosition(teamID, leagueID, season)
		if err == nil {
			currentStats.InitialPosition = ip
			logger.Debug("Set initial position for team", teamID, "to", ip)
		} else {
			logger.Debug("No initial position found for team", teamID, "error:", err)
		}

		// add this to the array so we can post process it
		roundStats = append(roundStats, currentStats)
	}

	// Calculate round averages for this round
	roundAverage, err := CalculateRoundAverages(roundStats, leagueID, season)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate round averages: %w", err)
	}
	// Now use these average calculations to recalculate each TeamStats objects adding new fields
	recalculateTeamStatsForRound(roundAverage, roundStats)

	// Calculate league positions based on points, goal difference, and goals scored
	calculateLeaguePositions(roundStats)

	// data will be persisted to db elsewhere
	return roundStats, nil
}

// Using round averages for all teams, calculate base statistics which will be used to
// calculate poisson based match predictions
func recalculateTeamStatsForRound(roundAverage *RoundAverage, roundStats []*TeamStats) error {
	// Now recalculate team stats with this round average
	// This implements the Python logic from Teams.calculateRoundAverages()
	// Use centralized configuration for weights
	formWeight := GetFormWeight()
	statsWeight := GetStatsWeight()

	for _, teamStat := range roundStats {
		// Calculate goals per game BEFORE calculating attack/defense strengths
		if teamStat.HomeGamesPlayed > 0 {
			teamStat.HomeGoalsPerGame = float64(teamStat.HomeGoals) / float64(teamStat.HomeGamesPlayed)
			teamStat.HomeGoalsConcededPerGame = float64(teamStat.HomeConceded) / float64(teamStat.HomeGamesPlayed)
		}

		if teamStat.AwayGamesPlayed > 0 {
			teamStat.AwayGoalsPerGame = float64(teamStat.AwayGoals) / float64(teamStat.AwayGamesPlayed)
			teamStat.AwayGoalsConcededPerGame = float64(teamStat.AwayConceded) / float64(teamStat.AwayGamesPlayed)
		}

		// Calculate form percentages (fp, hfp, afp) - normalized against round maximums
		teamStat.FP = float64(teamStat.Form) / makeSensible(roundAverage.MaxForm)
		teamStat.HFP = float64(teamStat.HomeForm) / makeSensible(roundAverage.MaxHomeForm)
		teamStat.AFP = float64(teamStat.AwayForm) / makeSensible(roundAverage.MaxAwayForm)

		// Round to 2 decimal places as in Python
		teamStat.FP = roundToDecimalPlaces(teamStat.FP, 2)
		teamStat.HFP = roundToDecimalPlaces(teamStat.HFP, 2)
		teamStat.AFP = roundToDecimalPlaces(teamStat.AFP, 2)

		// Calculate attack/defense strengths using weighted combination of stats and form
		// Home Attack Strength
		homeAttack := teamStat.HomeGoalsPerGame / makeSensible(roundAverage.MeanHomeGoalsPerGame)
		homeAttack = (statsWeight * homeAttack) + (formWeight * ((teamStat.FP + teamStat.HFP) / 2) * homeAttack)
		teamStat.HomeAttackStrength = homeAttack

		// Home Defense Strength
		homeDefense := teamStat.HomeGoalsConcededPerGame / makeSensible(roundAverage.MeanHomeGoalsConcededPerGame)
		homeDefense = (statsWeight * homeDefense) + (formWeight * (2 - (teamStat.FP+teamStat.HFP)/2) * homeDefense)
		teamStat.HomeDefenseStrength = homeDefense

		// Away Attack Strength
		awayAttack := teamStat.AwayGoalsPerGame / makeSensible(roundAverage.MeanAwayGoalsPerGame)
		awayAttack = (statsWeight * awayAttack) + (formWeight * ((teamStat.FP + teamStat.AFP) / 2) * awayAttack)
		teamStat.AwayAttackStrength = awayAttack

		// Away Defense Strength
		awayDefense := teamStat.AwayGoalsConcededPerGame / makeSensible(roundAverage.MeanAwayGoalsConcededPerGame)
		awayDefense = (statsWeight * awayDefense) + (formWeight * (2 - (teamStat.FP+teamStat.AFP)/2) * awayDefense)
		teamStat.AwayDefenseStrength = awayDefense
	}

	return nil
}

// calculateTeamStatsForRound calculates cumulative team statistics for a specific round
// Each TeamStats object represents one team's performance in one specific match (round)
// The MatchID field links each TeamStats to the specific match it represents
func calculateTeamStatsForRound(teamID string, matches []*Match, prevStats *TeamStats, leagueID int, season string, round int) *TeamStats {
	stats := &TeamStats{
		TeamID:   teamID,
		Season:   season,
		Round:    round,
		LeagueID: strconv.Itoa(leagueID),

		// Copy previous cumulative stats
		GamesPlayed:     prevStats.GamesPlayed,
		HomeGamesPlayed: prevStats.HomeGamesPlayed,
		AwayGamesPlayed: prevStats.AwayGamesPlayed,
		HomeWins:        prevStats.HomeWins,
		HomeDraws:       prevStats.HomeDraws,
		HomeLosses:      prevStats.HomeLosses,
		AwayWins:        prevStats.AwayWins,
		AwayDraws:       prevStats.AwayDraws,
		AwayLosses:      prevStats.AwayLosses,
		HomeGoals:       prevStats.HomeGoals,
		HomeConceded:    prevStats.HomeConceded,
		AwayGoals:       prevStats.AwayGoals,
		AwayConceded:    prevStats.AwayConceded,
		Points:          prevStats.Points,
		Form:            prevStats.Form,
		HomeForm:        prevStats.HomeForm,
		AwayForm:        prevStats.AwayForm,
	}

	// Find matches involving this team in this round
	for _, match := range matches {
		if !match.HasBeenPlayed() {
			continue // Skip unplayed matches
		}

		if match.HomeID == teamID {
			// Team played at home - populate MatchID for this specific fixture
			stats.MatchID = match.ID
			logger.Debug("Populated MatchID for home team", teamID, "match:", match.ID, "round:", round)
			stats.GamesPlayed++
			stats.HomeGamesPlayed++
			stats.HomeGoals += match.ActualHomeGoals
			stats.HomeConceded += match.ActualAwayGoals

			// Determine result
			if match.ActualHomeGoals > match.ActualAwayGoals {
				stats.HomeWins++
				stats.Points += 3
				stats.Form = UpdateFormData(stats.Form, 3) // Win
				stats.HomeForm = UpdateFormData(stats.HomeForm, 3)
			} else if match.ActualHomeGoals == match.ActualAwayGoals {
				stats.HomeDraws++
				stats.Points += 1
				stats.Form = UpdateFormData(stats.Form, 2) // Draw
				stats.HomeForm = UpdateFormData(stats.HomeForm, 2)
			} else {
				stats.HomeLosses++
				stats.Form = UpdateFormData(stats.Form, 1) // Loss
				stats.HomeForm = UpdateFormData(stats.HomeForm, 1)
			}

		} else if match.AwayID == teamID {
			// Team played away - populate MatchID for this specific fixture
			stats.MatchID = match.ID
			logger.Debug("Populated MatchID for away team", teamID, "match:", match.ID, "round:", round)
			stats.GamesPlayed++
			stats.AwayGamesPlayed++
			stats.AwayGoals += match.ActualAwayGoals
			stats.AwayConceded += match.ActualHomeGoals

			// Determine result
			if match.ActualAwayGoals > match.ActualHomeGoals {
				stats.AwayWins++
				stats.Points += 3
				stats.Form = UpdateFormData(stats.Form, 3) // Win
				stats.AwayForm = UpdateFormData(stats.AwayForm, 3)
			} else if match.ActualAwayGoals == match.ActualHomeGoals {
				stats.AwayDraws++
				stats.Points += 1
				stats.Form = UpdateFormData(stats.Form, 2) // Draw
				stats.AwayForm = UpdateFormData(stats.AwayForm, 2)
			} else {
				stats.AwayLosses++
				stats.Form = UpdateFormData(stats.Form, 1) // Loss
				stats.AwayForm = UpdateFormData(stats.AwayForm, 1)
			}
		}
	}

	// Validation: Ensure we found a match for this team in this round
	if stats.MatchID == "" {
		logger.Warn("No match found for team", teamID, "in round", round, "season", season, "league", leagueID)
		// This might happen for teams that didn't play in a particular round
		// or if there's a data inconsistency
	}

	return stats
}

// calculateLeaguePositions calculates and assigns league table positions to all teams
// Teams are ranked by: 1) Points (desc), 2) Goal Difference (desc), 3) Goals Scored (desc)
func calculateLeaguePositions(teamStats []*TeamStats) {
	if len(teamStats) == 0 {
		return
	}

	// Sort teams by league table criteria
	// Note: Go's sort is stable, so we sort by least important criteria first

	// Sort by goals scored (ascending, will be reversed by points sort)
	sort.Slice(teamStats, func(i, j int) bool {
		return (teamStats[i].HomeGoals + teamStats[i].AwayGoals) > (teamStats[j].HomeGoals + teamStats[j].AwayGoals)
	})

	// Sort by goal difference (ascending, will be reversed by points sort)
	sort.Slice(teamStats, func(i, j int) bool {
		goalDiffI := (teamStats[i].HomeGoals + teamStats[i].AwayGoals) - (teamStats[i].HomeConceded + teamStats[i].AwayConceded)
		goalDiffJ := (teamStats[j].HomeGoals + teamStats[j].AwayGoals) - (teamStats[j].HomeConceded + teamStats[j].AwayConceded)
		return goalDiffI > goalDiffJ
	})

	// Sort by points (descending) - this is the primary sort
	sort.Slice(teamStats, func(i, j int) bool {
		return teamStats[i].Points > teamStats[j].Points
	})

	// Assign positions (1st place = position 1)
	for i, team := range teamStats {
		team.Position = i + 1
	}

}

/////////////////////////////////////////////////////////////////////////
////// Persistable Interface Implementation
/////////////////////////////////////////////////////////////////////////

func SaveTeamStats(teamStats []*TeamStats) error {
	logger.Info("Saving TeamStats to database", len(teamStats))

	// Convert all matches to Persistable for BulkSave
	// The Save function in persistable handles INSERT/UPDATE automatically
	var persistableStats []Persistable
	for _, ts := range teamStats {
		persistableStats = append(persistableStats, ts)
	}

	if len(persistableStats) > 0 {
		if err := BulkSave(persistableStats); err != nil {
			return fmt.Errorf("failed to bulk save TeamStats: %w", err)
		}
		logger.Info("Bulk saved/updated TeamStats", len(persistableStats))
	} else {
		logger.Info("No TeamStats to save")
	}
	return nil
}

// GetPrimaryKey returns the compound primary key as a map
func (ts *TeamStats) GetPrimaryKey() map[string]interface{} {
	return map[string]any{
		"team_id":   ts.TeamID,
		"season":    ts.Season,
		"round":     ts.Round,
		"league_id": ts.LeagueID,
	}
}

// SetPrimaryKey sets the compound primary key from a map
func (ts *TeamStats) SetPrimaryKey(pk map[string]interface{}) error {
	if teamID, ok := pk["team_id"]; ok {
		if teamIDStr, ok := teamID.(string); ok {
			ts.TeamID = teamIDStr
		} else {
			return fmt.Errorf("primary key 'team_id' must be a string")
		}
	} else {
		return fmt.Errorf("primary key 'team_id' not found")
	}

	if season, ok := pk["season"]; ok {
		if seasonStr, ok := season.(string); ok {
			ts.Season = seasonStr
		} else {
			return fmt.Errorf("primary key 'season' must be a string")
		}
	} else {
		return fmt.Errorf("primary key 'season' not found")
	}

	if round, ok := pk["round"]; ok {
		if roundInt, ok := round.(int); ok {
			ts.Round = roundInt
		} else if roundInt64, ok := round.(int64); ok {
			ts.Round = int(roundInt64)
		} else {
			return fmt.Errorf("primary key 'round' must be an integer")
		}
	} else {
		return fmt.Errorf("primary key 'round' not found")
	}

	if leagueID, ok := pk["league_id"]; ok {
		if leagueIDStr, ok := leagueID.(string); ok {
			ts.LeagueID = leagueIDStr
		} else {
			return fmt.Errorf("primary key 'league_id' must be a string")
		}
	} else {
		return fmt.Errorf("primary key 'league_id' not found")
	}

	return nil
}

// GetTableName returns the table name for team stats
func (ts *TeamStats) GetTableName() string {
	return "team_stats"
}

// BeforeSave is called before saving the team stats
func (ts *TeamStats) BeforeSave() error {
	// Calculate averages
	if ts.HomeGamesPlayed > 0 {
		ts.HomeGoalsPerGame = float64(ts.HomeGoals) / float64(ts.HomeGamesPlayed)
		ts.HomeGoalsConcededPerGame = float64(ts.HomeConceded) / float64(ts.HomeGamesPlayed)
	}

	if ts.AwayGamesPlayed > 0 {
		ts.AwayGoalsPerGame = float64(ts.AwayGoals) / float64(ts.AwayGamesPlayed)
		ts.AwayGoalsConcededPerGame = float64(ts.AwayConceded) / float64(ts.AwayGamesPlayed)
	}

	// Set timestamps
	now := time.Now()
	if ts.CreatedAt.IsZero() {
		ts.CreatedAt = now
	}
	ts.UpdatedAt = now

	return nil
}

// AfterSave is called after saving the team stats
func (ts *TeamStats) AfterSave() error {
	return nil
}

// BeforeDelete is called before deleting the team stats
func (ts *TeamStats) BeforeDelete() error {
	return nil
}

// AfterDelete is called after deleting the team stats
func (ts *TeamStats) AfterDelete() error {
	return nil
}

// roundToDecimalPlaces rounds a float64 to the specified number of decimal places
func roundToDecimalPlaces(value float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(value*multiplier) / multiplier
}
