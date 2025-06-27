package podds

import (
	"fmt"

	"github.com/richard-senior/mcp/internal/logger"
)

// RoundAverage represents the average statistics for all teams in a specific round
// we don't bother to implement the persistable interface as this data is ephemeral being
// used only once as a precursort to calculating fields on the TeamStats objects
type RoundAverage struct {
	// Primary key fields
	Round    int    `json:"round" column:"round" dbtype:"INTEGER NOT NULL" primary:"true"`
	LeagueID string `json:"leagueId" column:"league_id" dbtype:"TEXT NOT NULL" primary:"true"`
	Season   string `json:"season" column:"season" dbtype:"TEXT NOT NULL" primary:"true"`

	// Basic stats
	TotalTeams int `json:"totalTeams" column:"total_teams" dbtype:"INTEGER DEFAULT 0"`

	// Mean goals per game
	MeanHomeGoalsPerGame         float64 `json:"meanHomeGoalsPerGame" column:"mean_home_goals_per_game" dbtype:"REAL DEFAULT 0.0"`
	MeanAwayGoalsPerGame         float64 `json:"meanAwayGoalsPerGame" column:"mean_away_goals_per_game" dbtype:"REAL DEFAULT 0.0"`
	MeanHomeGoalsConcededPerGame float64 `json:"meanHomeGoalsConcededPerGame" column:"mean_home_goals_conceded_per_game" dbtype:"REAL DEFAULT 0.0"`
	MeanAwayGoalsConcededPerGame float64 `json:"meanAwayGoalsConcededPerGame" column:"mean_away_goals_conceded_per_game" dbtype:"REAL DEFAULT 0.0"`

	// Mean form
	MeanForm     float64 `json:"meanForm" column:"mean_form" dbtype:"REAL DEFAULT 0.0"`
	MeanHomeForm float64 `json:"meanHomeForm" column:"mean_home_form" dbtype:"REAL DEFAULT 0.0"`
	MeanAwayForm float64 `json:"meanAwayForm" column:"mean_away_form" dbtype:"REAL DEFAULT 0.0"`

	// Max form
	MaxForm     float64 `json:"maxForm" column:"max_form" dbtype:"REAL DEFAULT 0.0"`
	MaxHomeForm float64 `json:"maxHomeForm" column:"max_home_form" dbtype:"REAL DEFAULT 0.0"`
	MaxAwayForm float64 `json:"maxAwayForm" column:"max_away_form" dbtype:"REAL DEFAULT 0.0"`

	// Mean attack/defense strengths
	MeanHomeAttack  float64 `json:"meanHomeAttack" column:"mean_home_attack" dbtype:"REAL DEFAULT 1.0"`
	MeanHomeDefense float64 `json:"meanHomeDefense" column:"mean_home_defense" dbtype:"REAL DEFAULT 1.0"`
	MeanAwayAttack  float64 `json:"meanAwayAttack" column:"mean_away_attack" dbtype:"REAL DEFAULT 1.0"`
	MeanAwayDefense float64 `json:"meanAwayDefense" column:"mean_away_defense" dbtype:"REAL DEFAULT 1.0"`
}

// makeSensible ensures a value is not zero to avoid division by zero
func makeSensible(value float64) float64 {
	if value == 0.0 {
		return 1.0
	}
	return value
}

// CalculateRoundAverages calculates round averages for all teams in a single round
func CalculateRoundAverages(teams []*TeamStats, leagueID string, season string) (*RoundAverage, error) {
	if len(teams) == 0 {
		return nil, fmt.Errorf("no teams provided for round average calculation")
	}

	// All teams should be from the same round - use the first team's round
	round := teams[0].Round

	logger.Info("Calculating round averages for league", leagueID, "season", season, "round", round)
	logger.Debug("Processing round", round, "with", len(teams), "teams")

	const FORM_WEIGHT = 0.3 // This should match Config.FORM_WEIGHT from Python
	const STATS_WEIGHT = 1.0 - FORM_WEIGHT

	// Initialize accumulators
	var (
		totalGamesPlayed    float64
		homeGoalsPerGame    float64
		awayGoalsPerGame    float64
		homeConcededPerGame float64
		awayConcededPerGame float64
		totalForm           float64
		totalHomeForm       float64
		totalAwayForm       float64
		maxForm             float64
		maxHomeForm         float64
		maxAwayForm         float64
	)

	// First pass: calculate sums and find maximums
	for _, team := range teams {
		totalGamesPlayed += float64(team.GamesPlayed) / 2.0 // Divide by 2 as per Python logic
		homeGoalsPerGame += team.HomeGoalsPerGame
		awayGoalsPerGame += team.AwayGoalsPerGame
		homeConcededPerGame += team.HomeGoalsConcededPerGame
		awayConcededPerGame += team.AwayGoalsConcededPerGame

		// Convert form integers to float64 for calculations
		formFloat := float64(team.Form)
		homeFormFloat := float64(team.HomeForm)
		awayFormFloat := float64(team.AwayForm)

		if formFloat > maxForm {
			maxForm = formFloat
		}
		if homeFormFloat > maxHomeForm {
			maxHomeForm = homeFormFloat
		}
		if awayFormFloat > maxAwayForm {
			maxAwayForm = awayFormFloat
		}

		totalForm += formFloat
		totalHomeForm += homeFormFloat
		totalAwayForm += awayFormFloat
	}

	// Create round average object
	roundAvg := &RoundAverage{
		Round:                        round,
		LeagueID:                     leagueID,
		Season:                       season,
		TotalTeams:                   len(teams),
		MeanHomeGoalsPerGame:         homeGoalsPerGame / float64(len(teams)),
		MeanAwayGoalsPerGame:         awayGoalsPerGame / float64(len(teams)),
		MeanHomeGoalsConcededPerGame: homeConcededPerGame / float64(len(teams)),
		MeanAwayGoalsConcededPerGame: awayConcededPerGame / float64(len(teams)),
		MeanForm:                     totalForm / float64(len(teams)),
		MeanHomeForm:                 totalHomeForm / float64(len(teams)),
		MeanAwayForm:                 totalAwayForm / float64(len(teams)),
		MaxForm:                      maxForm,
		MaxHomeForm:                  maxHomeForm,
		MaxAwayForm:                  maxAwayForm,
	}

	// Second pass: calculate attack/defense strengths and form percentages
	var (
		totalHomeAttack  float64
		totalHomeDefense float64
		totalAwayAttack  float64
		totalAwayDefense float64
	)

	for _, team := range teams {
		// Calculate form percentages
		team.FormPercentage = float64(team.Form) / makeSensible(roundAvg.MaxForm)
		team.HomeFormPercentage = float64(team.HomeForm) / makeSensible(roundAvg.MaxHomeForm)
		team.AwayFormPercentage = float64(team.AwayForm) / makeSensible(roundAvg.MaxAwayForm)

		// Also set the Poisson prediction form percentage fields (FP, HFP, AFP)
		team.FP = roundToDecimalPlaces(team.FormPercentage, 2)
		team.HFP = roundToDecimalPlaces(team.HomeFormPercentage, 2)
		team.AFP = roundToDecimalPlaces(team.AwayFormPercentage, 2)

		// Calculate attack strengths
		homeAttack := team.HomeGoalsPerGame / makeSensible(roundAvg.MeanHomeGoalsPerGame)
		homeAttack = (STATS_WEIGHT * homeAttack) + (FORM_WEIGHT * ((team.FormPercentage + team.HomeFormPercentage) / 2) * homeAttack)
		team.HomeAttackStrength = homeAttack
		totalHomeAttack += homeAttack

		awayAttack := team.AwayGoalsPerGame / makeSensible(roundAvg.MeanAwayGoalsPerGame)
		awayAttack = (STATS_WEIGHT * awayAttack) + (FORM_WEIGHT * ((team.FormPercentage + team.AwayFormPercentage) / 2) * awayAttack)
		team.AwayAttackStrength = awayAttack
		totalAwayAttack += awayAttack

		// Calculate defense strengths
		homeDefense := team.HomeGoalsConcededPerGame / makeSensible(roundAvg.MeanHomeGoalsConcededPerGame)
		homeDefense = (STATS_WEIGHT * homeDefense) + (FORM_WEIGHT * (2 - (team.FormPercentage+team.HomeFormPercentage)/2) * homeDefense)
		team.HomeDefenseStrength = homeDefense
		totalHomeDefense += homeDefense

		awayDefense := team.AwayGoalsConcededPerGame / makeSensible(roundAvg.MeanAwayGoalsConcededPerGame)
		awayDefense = (STATS_WEIGHT * awayDefense) + (FORM_WEIGHT * (2 - (team.FormPercentage+team.AwayFormPercentage)/2) * awayDefense)
		team.AwayDefenseStrength = awayDefense
		totalAwayDefense += awayDefense
	}

	// Set mean attack/defense values
	roundAvg.MeanHomeAttack = totalHomeAttack / float64(len(teams))
	roundAvg.MeanHomeDefense = totalHomeDefense / float64(len(teams))
	roundAvg.MeanAwayAttack = totalAwayAttack / float64(len(teams))
	roundAvg.MeanAwayDefense = totalAwayDefense / float64(len(teams))

	// Save the round average
	// we don't need to persist this object it's ephemeral
	/*
		if err := Save(roundAvg); err != nil {
			logger.Error("Failed to save round average for round", round, "error:", err)
			return nil, fmt.Errorf("failed to save round average for round %d: %w", round, err)
		}
	*/

	logger.Info("Successfully calculated round average for round", round, "with", len(teams), "teams")
	return roundAvg, nil
}

// GetTableName returns the database table name for RoundAverage
func (r *RoundAverage) GetTableName() string {
	return "round_average"
}

// GetPrimaryKey returns the primary key values for RoundAverage
func (r *RoundAverage) GetPrimaryKey() map[string]interface{} {
	return map[string]interface{}{
		"round":     r.Round,
		"league_id": r.LeagueID,
		"season":    r.Season,
	}
}

// SetPrimaryKey sets the primary key values for RoundAverage
func (r *RoundAverage) SetPrimaryKey(pk map[string]interface{}) error {
	if round, ok := pk["round"].(int); ok {
		r.Round = round
	}
	if leagueID, ok := pk["league_id"].(string); ok {
		r.LeagueID = leagueID
	}
	if season, ok := pk["season"].(string); ok {
		r.Season = season
	}
	return nil
}

// BeforeDelete is called before a RoundAverage record is deleted from the database
func (r *RoundAverage) BeforeDelete() error {
	// No special validation needed for RoundAverage
	return nil
}

// AfterDelete is called after a RoundAverage record is deleted from the database
func (r *RoundAverage) AfterDelete() error {
	// No special cleanup needed for RoundAverage
	return nil
}

// BeforeSave is called before a RoundAverage record is saved to the database
func (r *RoundAverage) BeforeSave() error {
	// No special validation needed for RoundAverage
	return nil
}

// AfterSave is called after a RoundAverage record is saved to the database
func (r *RoundAverage) AfterSave() error {
	// No special processing needed for RoundAverage
	return nil
}
