package podds

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/util"
)

// Compile-time check to ensure Match implements Persistable interface
var _ Persistable = (*Match)(nil)

// Match represents a football match with database persistence and JSON processing annotations
type Match struct {
	// Primary key
	ID string `json:"id" column:"id" dbtype:"TEXT" primary:"true" index:"true"`
	// Info
	UTCTime  time.Time `json:"utcTime" column:"utcTime" dbtype:"DATETIME" index:"true"`
	Round    string    `json:"round" column:"round" dbtype:"TEXT" index:"true"`
	LeagueID int       `json:"leagueId" column:"leagueId" dbtype:"INTEGER DEFAULT -1" index:"true"`
	Season   string    `json:"season" column:"season" dbtype:"TEXT" index:"true"`
	Status   string    `json:"status" column:"status" dbtype:"TEXT"` // "finished", "scheduled", "cancelled", etc.

	// Home team fields
	HomeTeamName string `json:"homeShortName" column:"homeTeamName" dbtype:"TEXT NOT NULL"`
	AwayTeamName string `json:"awayShortName" column:"awayTeamName" dbtype:"TEXT NOT NULL"`
	HomeID       string `json:"homeId" column:"homeId" dbtype:"TEXT NOT NULL" index:"true"`
	AwayID       string `json:"awayId" column:"awayId" dbtype:"TEXT NOT NULL" index:"true"`

	// Core match data (compressed from complex status fields)
	ActualHomeGoals         int `json:"actualHomeGoals" column:"actualHomeGoals" dbtype:"INTEGER DEFAULT -1"`
	ActualHalfTimeHomeGoals int `json:"actualHalfTimeHomeGoals" column:"actualHalfTimeHomeGoals" dbtype:"INTEGER DEFAULT -1"`
	ActualAwayGoals         int `json:"actualAwayGoals" column:"actualAwayGoals" dbtype:"INTEGER DEFAULT -1"`
	ActualHalfTimeAwayGoals int `json:"actualHalfTimeAwayGoals" column:"actualHalfTimeAwayGoals" dbtype:"INTEGER DEFAULT -1"`

	// Poisson Prediction Fields
	PoissonPredictedHomeGoals         int `json:"poissonPredictedHomeGoals,omitempty" column:"poissonPredictedHomeGoals" dbtype:"INTEGER DEFAULT -1" `
	PoissonPredictedHalfTimeHomeGoals int `json:"poissonPredictedHalfTimeHomeGoals,omitempty" column:"poissonPredictedHalfTimeHomeGoals" dbtype:"INTEGER DEFAULT -1" `
	PoissonPredictedAwayGoals         int `json:"poissonPredictedAwayGoals,omitempty" column:"poissonPredictedAwayGoals" dbtype:"INTEGER DEFAULT -1" `
	PoissonPredictedHalfTimeAwayGoals int `json:"poissonPredictedHalfTimeAwayGoals,omitempty" column:"poissonPredictedHalfTimeAwayGoals" dbtype:"INTEGER DEFAULT -1" `

	// Expected Goals (from Poisson calculation)
	HomeTeamGoalExpectency float64 `json:"homeTeamGoalExpectency,omitempty" column:"homeTeamGoalExpectency" dbtype:"REAL DEFAULT -1.0"`
	AwayTeamGoalExpectency float64 `json:"awayTeamGoalExpectency,omitempty" column:"awayTeamGoalExpectency" dbtype:"REAL DEFAULT -1.0"`

	// Win/Draw/Loss Probabilities (percentages)
	PoissonHomeWinProbability float64 `json:"poissonHomeWinProbability,omitempty" column:"poissonHomeWinProbability" dbtype:"REAL DEFAULT -1.0"`
	PoissonDrawProbability    float64 `json:"poissonDrawProbability,omitempty" column:"poissonDrawProbability" dbtype:"REAL DEFAULT -1.0"`
	PoissonAwayWinProbability float64 `json:"poissonAwayWinProbability,omitempty" column:"poissonAwayWinProbability" dbtype:"REAL DEFAULT -1.0"`

	// Over/Under Goals Probabilities (percentages)
	Over1p5Goals float64 `json:"over1p5Goals,omitempty" column:"over1p5Goals" dbtype:"REAL DEFAULT -1.0"`
	Over2p5Goals float64 `json:"over2p5Goals,omitempty" column:"over2p5Goals" dbtype:"REAL DEFAULT -1.0"`

	// Average Betting Odds (from football-data.co.uk)
	ActualHomeOdds float64 `json:"actualHomeOdds,omitempty" column:"actualHomeOdds" dbtype:"REAL DEFAULT -1.0"`
	ActualDrawOdds float64 `json:"actualDrawOdds,omitempty" column:"actualDrawOdds" dbtype:"REAL DEFAULT -1.0"`
	ActualAwayOdds float64 `json:"actualAwayOdds,omitempty" column:"actualAwayOdds" dbtype:"REAL DEFAULT -1.0"`

	// Match details
	MatchUrl string `json:"pageUrl" column:"matchUrl" dbtype:"TEXT"`
	Poke     int    `json:"poke,omitempty" column:"poke" dbtype:"INTEGER DEFAULT -1"`
	Referee  string `json:"referee,omitempty" column:"referee" dbtype:"TEXT"`

	// Action
	HomeShotsOnTarget int `json:"homeShotsOnTarget,omitempty" column:"homeShotsOnTarget" dbtype:"INTEGER DEFAULT -1"`
	AwayShotsOnTarget int `json:"awayShotsOnTarget,omitempty" column:"awayShotsOnTarget" dbtype:"INTEGER DEFAULT -1"`
	HomeCorners       int `json:"homeCorners,omitempty" column:"homeCorners" dbtype:"INTEGER DEFAULT -1"`
	AwayCorners       int `json:"awayCorners,omitempty" column:"awayCorners" dbtype:"INTEGER DEFAULT -1"`

	// Discipline
	HomeYellowCards int `json:"homeYellowCards,omitempty" column:"homeYellowCards" dbtype:"INTEGER DEFAULT -1"`
	AwayYellowCards int `json:"awayYellowCards,omitempty" column:"awayYellowCards" dbtype:"INTEGER DEFAULT -1"`
	HomeRedCards    int `json:"homeRedCards,omitempty" column:"homeRedCards" dbtype:"INTEGER DEFAULT -1"`
	AwayRedCards    int `json:"awayRedCards,omitempty" column:"awayRedCards" dbtype:"INTEGER DEFAULT -1"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" column:"created_at" dbtype:"DATETIME DEFAULT CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updatedAt" column:"updated_at" dbtype:"DATETIME DEFAULT CURRENT_TIMESTAMP"`
}

/////////////////////////////////////////////////////////////////////////
////// Persistable Interface Implementation
/////////////////////////////////////////////////////////////////////////

// GetPrimaryKey returns the primary key as a map
func (m *Match) GetPrimaryKey() map[string]interface{} {
	return map[string]any{
		"id": m.ID,
	}
}

// SetPrimaryKey sets the primary key from a map
func (m *Match) SetPrimaryKey(pk map[string]interface{}) error {
	if id, ok := pk["id"]; ok {
		if idStr, ok := id.(string); ok {
			m.ID = idStr
			return nil
		}
		return fmt.Errorf("primary key 'id' must be a string")
	}
	return fmt.Errorf("primary key 'id' not found")
}

// GetTableName returns the table name for matches
func (m *Match) GetTableName() string {
	return "match"
}

// BeforeSave is called before saving the match
func (m *Match) BeforeSave() error {
	// Process and derive computed fields
	m.ProcessMatchData()

	// Set timestamps
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	return nil
}

// AfterSave is called after saving the match
func (m *Match) AfterSave() error {
	return nil
}

// BeforeDelete is called before deleting the match
func (m *Match) BeforeDelete() error {
	return nil
}

// AfterDelete is called after deleting the match
func (m *Match) AfterDelete() error {
	return nil
}

// ///////////////////////////////////////////////////////////////////////
// //// Data Processing Methods (Following PODDS Pattern)
// ///////////////////////////////////////////////////////////////////////
// Returns true if we should recalculate this match, based upon its UTCTime and
// whether or not certain fields are populated such as actualHomeGoals etc.
func (m *Match) ShouldProcess() bool {
	if m.Season == "" || m.LeagueID == -1 {
		return true
	}
	// if we have no predictions for this match then we should predict some
	if m.PoissonPredictedHomeGoals == -1 || m.PoissonPredictedAwayGoals == -1 {
		return true
	}
	// it's naughty but I'm going to do it
	// generally coding anything relating to testing into the codebase is
	// but.. since this is MY codebase I'll do what I like, because I'm not 8 years old
	// and you can't make me
	if testing.Testing() {
		return true
	}
	// ok we already have predictions so we should probably not predict
	// however if this is the current season we need to keep predicting as new data comes in
	// up to about 'n' (probs 15) minutes before the match
	now := time.Now()
	timeBuffer := time.Duration(GetPredictionTimeBuffer()) * time.Minute
	bufferTime := now.Add(timeBuffer)
	if !m.UTCTime.Before(bufferTime) {
		return true
	}
	return false
}

// ProcessMatchData processes and compresses incoming data
func (m *Match) ProcessMatchData() {
	m.deriveStatus()
	m.calculatePoke()
}

// calculatePoke calculates the travel distance between home and away teams
func (m *Match) calculatePoke() {
	if m.HomeID == "" || m.AwayID == "" {
		logger.Debug("Cannot calculate poke: missing team IDs")
		return
	}

	distance := CalculateDistanceForTeamIDs(m.HomeID, m.AwayID)
	if distance > 0.0 {
		// Convert to int as per original Python implementation
		m.Poke = int(math.Round(distance))
		logger.Debug("Calculated poke distance", m.Poke, "miles for", m.HomeTeamName, "vs", m.AwayTeamName)
	} else {
		logger.Debug("No distance data for teams", m.HomeTeamName, "vs", m.AwayTeamName)
		m.Poke = -1
	}
}

// deriveStatus sets a simple status based on the match data
func (m *Match) deriveStatus() {
	if m.Status != "" {
		return // Don't override explicitly set status
	}

	if m.HasBeenPlayed() {
		m.Status = "finished"
	} else if m.UTCTime.Before(time.Now()) {
		m.Status = "in_progress"
	} else {
		m.Status = "scheduled"
	}
}

/////////////////////////////////////////////////////////////////////////
////// Status Query Methods (Following PODDS Pattern)
/////////////////////////////////////////////////////////////////////////

// HasBeenPlayed determines if match has been completed
func (m *Match) HasBeenPlayed() bool {
	return m.ActualHomeGoals >= 0 && m.ActualAwayGoals >= 0
}

// IsScheduled determines if match is in the future
func (m *Match) IsScheduled() bool {
	return !m.HasBeenPlayed() && m.UTCTime.After(time.Now())
}

// IsFinished checks if the match is finished
func (m *Match) IsFinished() bool {
	return m.Status == "finished" || m.HasBeenPlayed()
}

// IsCancelled checks if the match is cancelled
func (m *Match) IsCancelled() bool {
	return m.Status == "cancelled"
}

// RecreateScoreStr generates score string from actual goals
func (m *Match) RecreateScoreStr() string {
	if !m.HasBeenPlayed() {
		return ""
	}
	return fmt.Sprintf("%d - %d", m.ActualHomeGoals, m.ActualAwayGoals)
}

/////////////////////////////////////////////////////////////////////////
////// Enhanced JSON Processing with Data Compression
/////////////////////////////////////////////////////////////////////////

// ParseMatchFromJSON parses JSON with intelligent data processing and compression
func ParseMatchFromJSON(jsonData []byte) (*Match, error) {
	// First, parse into a temporary struct that can handle all possible JSON fields
	var rawData map[string]interface{}
	err := json.Unmarshal(jsonData, &rawData)
	if err != nil {
		return nil, err
	}

	match := NewMatch()

	// Extract and process core fields
	match.extractCoreFields(rawData)

	// Process complex status information into simple fields
	match.processStatusFields(rawData)

	// Derive computed fields
	match.ProcessMatchData()

	return match, nil
}

// NewMatch creates a new Match with default values for numeric fields
// All numeric fields default to -1 (int) or -1.0 (float64) to distinguish from valid zero values
func NewMatch() *Match {
	return &Match{
		LeagueID:                  -1,
		ActualHomeGoals:           -1,
		ActualAwayGoals:           -1,
		PoissonPredictedHomeGoals: -1,
		PoissonPredictedAwayGoals: -1,
		HomeTeamGoalExpectency:    -1.0,
		AwayTeamGoalExpectency:    -1.0,
		PoissonHomeWinProbability: -1.0,
		PoissonDrawProbability:    -1.0,
		PoissonAwayWinProbability: -1.0,
		Over1p5Goals:              -1.0,
		Over2p5Goals:              -1.0,
		Poke:                      -1,
		ActualHomeOdds:            -1.0,
		ActualDrawOdds:            -1.0,
		ActualAwayOdds:            -1.0,
		HomeShotsOnTarget:         -1,
		AwayShotsOnTarget:         -1,
		HomeCorners:               -1,
		AwayCorners:               -1,
		HomeYellowCards:           -1,
		AwayYellowCards:           -1,
		HomeRedCards:              -1,
		AwayRedCards:              -1,
	}
}

// Merges the data from n into m if the data in m
// is missing and n has it
func (m *Match) Merge(n *Match) error {
	if n == nil {
		return fmt.Errorf("must pass a match")
	}

	// Use reflection to iterate through all fields
	mVal := reflect.ValueOf(m).Elem()
	nVal := reflect.ValueOf(n).Elem()
	mType := mVal.Type()

	for i := 0; i < mVal.NumField(); i++ {
		field := mVal.Field(i)
		fieldType := mType.Field(i)
		nField := nVal.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			// If string field is empty, copy from n
			if field.String() == "" && nField.String() != "" {
				field.SetString(nField.String())
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// If int field is -1, copy from n
			if field.Int() == -1 && nField.Int() != -1 {
				field.SetInt(nField.Int())
			}
		case reflect.Float32, reflect.Float64:
			// If float field is -1.0, copy from n
			if field.Float() == -1.0 && nField.Float() != -1.0 {
				field.SetFloat(nField.Float())
			}
		case reflect.Struct:
			// Special handling for time.Time
			if fieldType.Type == reflect.TypeOf(time.Time{}) {
				timeField := field.Interface().(time.Time)
				nTimeField := nField.Interface().(time.Time)
				if timeField.IsZero() && !nTimeField.IsZero() {
					field.Set(nField)
				}
			}
		default:
			// if the field in m is nil, just replace it with the content of the field in n
			logger.Info("failed to determine field type", fieldType.Type)
		}
	}
	return nil
}

/**
* Returns true if the given Match object is ostensibly the same match
* That is, if it has the same match ID or refers to the same date, league, teams etc.
 */
func (m *Match) Equals(n *Match) bool {
	if n == nil {
		return false
	}
	// if the matchId is the same, it's the same match
	// regardless of any other content
	if m.ID == n.ID {
		return true
	}
	if m.HomeID == "" && m.LeagueID == -1 && m.Season == "" && m.UTCTime.IsZero() {
		return false
	}
	// If we're not the same teams
	if m.HomeID != n.HomeID || m.AwayID != n.AwayID {
		return false
	}
	// not same league or season?
	if m.LeagueID != n.LeagueID || m.Season != n.Season {
		return false
	}
	if m.ActualHomeGoals != -1 && n.ActualAwayGoals != -1 {
		if m.ActualHomeGoals != n.ActualHomeGoals || m.ActualAwayGoals != n.ActualAwayGoals {
			return false
		}
	}
	if !m.UTCTime.IsZero() {
		if n.UTCTime.IsZero() {
			return false
		}
		// if these two matches aren't on the same day, they're not the same match
		if m.UTCTime.Year() != n.UTCTime.Year() || m.UTCTime.YearDay() != n.UTCTime.YearDay() {
			return false
		}
	}
	return true
}

// extractCoreFields pulls out the essential match data
func (m *Match) extractCoreFields(data map[string]interface{}) {
	// Extract basic match info
	if id, ok := data["id"].(string); ok {
		m.ID = id
	}

	if pageUrl, ok := data["pageUrl"].(string); ok {
		m.MatchUrl = pageUrl
	}

	if round, ok := data["round"].(string); ok {
		m.Round = round
	}

	// Extract league and season info
	if leagueId, ok := data["leagueId"]; ok {
		if leagueIdFloat, ok := leagueId.(float64); ok {
			m.LeagueID = int(leagueIdFloat)
		} else if leagueIdStr, ok := leagueId.(string); ok {
			if parsed, err := strconv.Atoi(leagueIdStr); err == nil {
				m.LeagueID = parsed
			}
		}
	}

	if season, ok := data["season"].(string); ok {
		m.Season = season
	}

	// Extract team information from home object
	if home, ok := data["home"].(map[string]interface{}); ok {
		if id, ok := home["id"].(string); ok {
			m.HomeID = id
		}
		if shortName, ok := home["shortName"].(string); ok {
			m.HomeTeamName = shortName
		}
	}

	// Extract team information from away object
	if away, ok := data["away"].(map[string]interface{}); ok {
		if id, ok := away["id"].(string); ok {
			m.AwayID = id
		}
		if shortName, ok := away["shortName"].(string); ok {
			m.AwayTeamName = shortName
		}
	}
}

// processStatusFields compresses complex status into simple fields
func (m *Match) processStatusFields(data map[string]interface{}) {
	// Look for status object
	if status, ok := data["status"].(map[string]interface{}); ok {
		// Extract UTC time
		if utcTime, ok := status["utcTime"].(string); ok {
			if t, err := time.Parse(time.RFC3339, utcTime); err == nil {
				m.UTCTime = t
			}
		}

		// Check if match is finished and extract goals
		if finished, ok := status["finished"].(bool); ok && finished {
			m.Status = "finished"

			if scoreStr, ok := status["scoreStr"].(string); ok {
				m.parseScoreString(scoreStr)
			}
		}

		// Check for cancellation
		if cancelled, ok := status["cancelled"].(bool); ok && cancelled {
			m.Status = "cancelled"
		}

		// Check if started but not finished
		if started, ok := status["started"].(bool); ok && started && m.Status == "" {
			m.Status = "in_progress"
		}
	}

	// Also check for direct goal fields in the main data
	if homeGoals, ok := data["actualHomeGoals"]; ok {
		if homeGoalsFloat, ok := homeGoals.(float64); ok {
			m.ActualHomeGoals = int(homeGoalsFloat)
		}
	}

	if awayGoals, ok := data["actualAwayGoals"]; ok {
		if awayGoalsFloat, ok := awayGoals.(float64); ok {
			m.ActualAwayGoals = int(awayGoalsFloat)
		}
	}
}

// parseScoreString extracts goals from score string like "2 - 1"
func (m *Match) parseScoreString(scoreStr string) {
	if scoreStr == "" {
		return
	}

	// Handle various score string formats: "2 - 1", "2-1", "2:1", etc.
	scoreStr = strings.ReplaceAll(scoreStr, " ", "")
	scoreStr = strings.ReplaceAll(scoreStr, ":", "-")

	parts := strings.Split(scoreStr, "-")
	if len(parts) != 2 {
		return
	}

	if homeGoals, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
		m.ActualHomeGoals = homeGoals
	}

	if awayGoals, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
		m.ActualAwayGoals = awayGoals
	}
}

// ParseMatchFromString parses a JSON string into a Match struct
func ParseMatchFromString(jsonStr string) (*Match, error) {
	return ParseMatchFromJSON([]byte(jsonStr))
}

// ToJSON serializes the Match struct to JSON bytes
func (m *Match) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ToJSONIndented serializes the Match struct to pretty-printed JSON bytes
func (m *Match) ToJSONIndented() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// ToJSONString serializes the Match struct to a JSON string
func (m *Match) ToJSONString() (string, error) {
	jsonData, err := m.ToJSON()
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// ToJSONStringIndented serializes the Match struct to a pretty-printed JSON string
func (m *Match) ToJSONStringIndented() (string, error) {
	jsonData, err := m.ToJSONIndented()
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

/////////////////////////////////////////////////////////////////////////
////// Match Collection Operations
/////////////////////////////////////////////////////////////////////////

// saveMatches saves matches to database using BulkSave
func SaveMatches(matches []*Match) error {
	logger.Info("Saving matches to database", len(matches))

	// Convert all matches to Persistable for BulkSave
	// The Save function in persistable handles INSERT/UPDATE automatically
	var persistableMatches []Persistable
	for _, match := range matches {
		persistableMatches = append(persistableMatches, match)
	}

	if len(persistableMatches) > 0 {
		if err := BulkSave(persistableMatches); err != nil {
			return fmt.Errorf("failed to bulk save matches: %w", err)
		}
		logger.Info("Bulk saved/updated matches", len(persistableMatches))
	} else {
		logger.Info("No matches to save")
	}

	return nil
}

// ExtractTeamsFromMatches extracts unique teams from match data
func ExtractTeamsFromMatches(matches []*Match) []*Team {
	teamMap := make(map[string]*Team)

	for _, match := range matches {
		// Add home team
		if match.HomeID != "" {
			if _, exists := teamMap[match.HomeID]; !exists {
				tid, err := util.GetAsInteger(match.HomeID)
				if err != nil {
					logger.Warn("Failed to convert team ID to integer", match.HomeID, err)
					// TODO what to do here?
				}
				td, err := TData.GetDataForTeam(match.HomeID)
				if err != nil {
					logger.Warn("TeamID "+match.HomeID+" ( "+match.HomeTeamName+" ) does not exist in the data lookup table, you should add it", err)
				}
				teamMap[match.HomeID] = &Team{
					ID:        tid,
					Name:      match.HomeTeamName,
					Latitude:  td.Latitude,
					Longitude: td.Longitude,
				}
			}
		}

		// Add away team
		if match.AwayID != "" {
			if _, exists := teamMap[match.AwayID]; !exists {
				tid, err := util.GetAsInteger(match.AwayID)
				if err != nil {
					logger.Warn("Failed to convert team ID to integer", match.AwayID, err)
					// TODO what to do here?
				}
				td, err := TData.GetDataForTeam(match.AwayID)
				if err != nil {
					logger.Warn("TeamID "+match.AwayID+" ( "+match.AwayTeamName+" ) does not exist in the data lookup table, you should add it", err)
				}
				teamMap[match.AwayID] = &Team{
					ID:        tid,
					Name:      match.AwayTeamName,
					Latitude:  td.Latitude,
					Longitude: td.Longitude,
				}
			}
		}
	}

	// Convert map to slice
	teams := make([]*Team, 0, len(teamMap))
	for _, team := range teamMap {
		teams = append(teams, team)
	}

	return teams
}

// GroupMatchesByRound groups matches by their round number
func GroupMatchesByRound(matches []*Match) map[int][]*Match {
	roundMatches := make(map[int][]*Match)

	for _, match := range matches {
		// Parse round number from round string
		roundNum := ParseRoundNumber(match.Round)
		if roundNum > 0 {
			roundMatches[roundNum] = append(roundMatches[roundNum], match)
		}
	}

	return roundMatches
}

// ParseRoundNumber extracts numeric round from round string
func ParseRoundNumber(roundStr string) int {
	// Handle various round formats: "Round 1", "1", "Matchday 1", etc.
	roundStr = strings.TrimSpace(roundStr)

	// Try to extract number from string
	parts := strings.Fields(roundStr)
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err == nil {
			return num
		}
	}

	// If no number found, try parsing the whole string
	if num, err := strconv.Atoi(roundStr); err == nil {
		return num
	}

	return 0 // Invalid round
}

// GetSortedRounds returns round numbers in ascending order
func GetSortedRounds(roundMatches map[int][]*Match) []int {
	rounds := make([]int, 0, len(roundMatches))
	for round := range roundMatches {
		rounds = append(rounds, round)
	}

	// Simple bubble sort for small datasets
	for i := 0; i < len(rounds)-1; i++ {
		for j := 0; j < len(rounds)-i-1; j++ {
			if rounds[j] > rounds[j+1] {
				rounds[j], rounds[j+1] = rounds[j+1], rounds[j]
			}
		}
	}

	return rounds
}

// GetTeamsFromMatches extracts unique team IDs from matches
func GetTeamsFromMatches(matches []*Match) []string {
	teamSet := make(map[string]bool)

	for _, match := range matches {
		if match.HomeID != "" {
			teamSet[match.HomeID] = true
		}
		if match.AwayID != "" {
			teamSet[match.AwayID] = true
		}
	}

	teams := make([]string, 0, len(teamSet))
	for teamID := range teamSet {
		teams = append(teams, teamID)
	}

	return teams
}
