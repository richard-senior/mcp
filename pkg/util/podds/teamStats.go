package podds

import (
	"fmt"
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

	// Form percentages
	FormPercentage     float64 `json:"formPercentage" column:"form_percentage" dbtype:"REAL DEFAULT 0.0"`
	HomeFormPercentage float64 `json:"homeFormPercentage" column:"home_form_percentage" dbtype:"REAL DEFAULT 0.0"`
	AwayFormPercentage float64 `json:"awayFormPercentage" column:"away_form_percentage" dbtype:"REAL DEFAULT 0.0"`

	// Points and position
	Points   int `json:"points" column:"points" dbtype:"INTEGER DEFAULT 0"`
	Position int `json:"position" column:"position" dbtype:"INTEGER DEFAULT 0"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" column:"created_at" dbtype:"DATETIME DEFAULT CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updatedAt" column:"updated_at" dbtype:"DATETIME DEFAULT CURRENT_TIMESTAMP"`
}

/////////////////////////////////////////////////////////////////////////
////// Persistable Interface Implementation
/////////////////////////////////////////////////////////////////////////

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

/////////////////////////////////////////////////////////////////////////
////// Team Statistics Processing Functions
/////////////////////////////////////////////////////////////////////////

// ProcessAndSaveTeamStats processes matches and generates team statistics following PODDS methodology
func ProcessAndSaveTeamStats(matches []*Match, leagueID int, season string) error {
	logger.Info("Processing team statistics for league", leagueID, "season", season)

	// Group matches by round
	roundMatches := GroupMatchesByRound(matches)

	// Process each round in order
	rounds := GetSortedRounds(roundMatches)
	
	for _, round := range rounds {
		logger.Debug("Processing round", round, "for league", leagueID, "season", season)
		
		if err := ProcessRoundStats(roundMatches[round], leagueID, season, round); err != nil {
			logger.Error("Failed to process round stats", round, err)
			continue
		}
	}

	logger.Info("Team statistics processed successfully")
	return nil
}

// ProcessRoundStats processes statistics for a specific round
func ProcessRoundStats(matches []*Match, leagueID int, season string, round int) error {
	// Get all teams in this round
	teams := GetTeamsFromMatches(matches)
	
	for _, teamID := range teams {
		// Get previous round stats for cumulative calculation
		var prevStats *TeamStats
		if round > 1 {
			prevStats = &TeamStats{}
			pk := map[string]interface{}{
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
		currentStats := CalculateTeamStatsForRound(teamID, matches, prevStats, leagueID, season, round)
		
		// Save the stats using the generic Save method
		if err := Save(currentStats); err != nil {
			logger.Warn("Failed to save team stats", teamID, round, err)
			continue
		}
		
		logger.Debug("Saved stats for team", teamID, "round", round)
	}
	
	return nil
}

// CalculateTeamStatsForRound calculates cumulative team statistics for a specific round
func CalculateTeamStatsForRound(teamID string, matches []*Match, prevStats *TeamStats, leagueID int, season string, round int) *TeamStats {
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
			// Team played at home
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
			// Team played away
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
	
	return stats
}
