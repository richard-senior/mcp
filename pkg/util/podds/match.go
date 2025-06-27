package podds

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	LeagueID int       `json:"leagueId" column:"leagueId" dbtype:"INTEGER" index:"true"`
	Season   string    `json:"season" column:"season" dbtype:"TEXT" index:"true"`
	Status   string    `json:"status" column:"status" dbtype:"TEXT"` // "finished", "scheduled", "cancelled", etc.

	// Home team fields
	HomeTeamName string `json:"homeShortName" column:"homeTeamName" dbtype:"TEXT NOT NULL"`
	AwayTeamName string `json:"awayShortName" column:"awayTeamName" dbtype:"TEXT NOT NULL"`
	HomeID       string `json:"homeId" column:"homeId" dbtype:"TEXT NOT NULL" index:"true"`
	AwayID       string `json:"awayId" column:"awayId" dbtype:"TEXT NOT NULL" index:"true"`

	// Core match data (compressed from complex status fields)
	ActualHomeGoals int `json:"actualHomeGoals" column:"actualHomeGoals" dbtype:"INTEGER DEFAULT -1"`
	ActualAwayGoals int `json:"actualAwayGoals" column:"actualAwayGoals" dbtype:"INTEGER DEFAULT -1"`
	// Prediciton
	PoissonPredictedHomeGoals int `json:"poissonPredictedHomeGoals,omitempty" column:"poissonPredictedHomeGoals" dbtype:"INTEGER DEFAULT -1" `
	PoissonPredictedAwayGoals int `json:"poissonPredictedAwayGoals,omitempty" column:"poissonPredictedAwayGoals" dbtype:"INTEGER DEFAULT -1" `

	// Match details
	MatchUrl string `json:"pageUrl" column:"matchUrl" dbtype:"TEXT"`
	Poke     int    `json:"poke,omitempty" column:"poke" dbtype:"INTEGER"`
	Referee  string `json:"referee,omitempty" column:"referee" dbtype:"TEXT"`

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

/////////////////////////////////////////////////////////////////////////
////// Data Processing Methods (Following PODDS Pattern)
/////////////////////////////////////////////////////////////////////////

// ProcessMatchData processes and compresses incoming data
func (m *Match) ProcessMatchData() {
	m.deriveStatus()
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

	match := &Match{}

	// Extract and process core fields
	match.extractCoreFields(rawData)

	// Process complex status information into simple fields
	match.processStatusFields(rawData)

	// Derive computed fields
	match.ProcessMatchData()

	return match, nil
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

	// Filter out matches that already exist
	var newMatches []Persistable
	for _, match := range matches {
		exists, err := Exists(match)
		if err != nil {
			logger.Warn("Failed to check if match exists", match.ID, err)
			continue
		}

		if !exists {
			newMatches = append(newMatches, match)
			logger.Debug("Will save new match", match.ID, match.HomeTeamName, "vs", match.AwayTeamName)
		} else {
			logger.Debug("Match already exists", match.ID, match.HomeTeamName, "vs", match.AwayTeamName)
		}
	}

	if len(newMatches) > 0 {
		if err := BulkSave(newMatches); err != nil {
			return fmt.Errorf("failed to bulk save matches: %w", err)
		}
		logger.Info("Bulk saved matches", len(newMatches))
	} else {
		logger.Info("No new matches to save")
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
