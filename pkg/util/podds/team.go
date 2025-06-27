package podds

import (
	"fmt"
	"math"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/util"
)

// Team represents a team in a match with database persistence annotations
type Team struct {
	ID          int     `json:"id" column:"id" dbtype:"INTEGER DEFAULT -1" primary:"true" index:"true"`
	Name        string  `json:"shortName" column:"name" dbtype:"TEXT NOT NULL"`
	CurrentForm int     `json:"form,omitempty" column:"currentForm" dbtype:"INTEGER DEFAULT -1"`
	Latitude    float64 `json:"latitude,omitempty" column:"latitude" dbtype:"REAL DEFAULT -1.0"`
	Longitude   float64 `json:"longitude,omitempty" column:"longitude" dbtype:"REAL DEFAULT -1.0"`
}

/////////////////////////////////////////////////////////////////////////
////// Persistable Interface Implementation
/////////////////////////////////////////////////////////////////////////

// GetPrimaryKey returns the primary key as a map
func (t *Team) GetPrimaryKey() map[string]any {
	return map[string]any{
		"id": t.ID,
	}
}

// SetPrimaryKey sets the primary key from a map
func (t *Team) SetPrimaryKey(pk map[string]any) error {
	if id, ok := pk["id"]; ok {
		tid, err := util.GetAsInteger(id)
		if err != nil {
			return fmt.Errorf("primary key 'id' must be an integer or string representation of an integer")
		}
		t.ID = tid
	}
	return fmt.Errorf("primary key 'id' not found")
}

// GetTableName returns the table name for teams
func (t *Team) GetTableName() string {
	return "team"
}

// BeforeSave is called before saving the team
func (t *Team) BeforeSave() error {
	return nil
}

// AfterSave is called after saving the team
func (t *Team) AfterSave() error {
	return nil
}

// BeforeDelete is called before deleting the team
func (t *Team) BeforeDelete() error {
	return nil
}

// AfterDelete is called after deleting the team
func (t *Team) AfterDelete() error {
	return nil
}

/////////////////////////////////////////////////////////////////////////
////// Util and access methods
/////////////////////////////////////////////////////////////////////////

/////////////////////////////////////////////////////////////////////////
////// Team Collection Operations
/////////////////////////////////////////////////////////////////////////

// SaveTeams saves teams to database using BulkSave
func SaveTeams(teams []*Team) error {
	logger.Info("Saving teams to database", len(teams))

	// Filter out teams that already exist
	var newTeams []Persistable
	for _, team := range teams {
		exists, err := Exists(team)
		if err != nil {
			logger.Warn("Failed to check if team exists", team.ID, err)
			continue
		}

		if !exists {
			newTeams = append(newTeams, team)
			logger.Debug("Will save new team", team.ID, team.Name)
		} else {
			logger.Debug("Team already exists", team.ID, team.Name)
		}
	}

	if len(newTeams) > 0 {
		if err := BulkSave(newTeams); err != nil {
			return fmt.Errorf("failed to bulk save teams: %w", err)
		}
		logger.Info("Bulk saved teams", len(newTeams))
	} else {
		logger.Info("No new teams to save")
	}

	return nil
}

/////////////////////////////////////////////////////////////////////////
////// Form Calculation Functions (Following PODDS Methodology)

// CalculateDistance calculates the 'as the crow flies' distance between two teams in miles
// using the Haversine formula with latitude and longitude data
func CalculateDistance(homeTeam, awayTeam *Team) float64 {
	if homeTeam == nil || awayTeam == nil {
		return -1.0
	}

	hlat := homeTeam.Latitude
	hlon := homeTeam.Longitude
	alat := awayTeam.Latitude
	alon := awayTeam.Longitude

	// Check if we have valid coordinates (not default -1.0 values and not zero)
	if (hlat == -1.0 && hlon == -1.0) || (alat == -1.0 && alon == -1.0) || 
	   (hlat == 0.0 && hlon == 0.0) || (alat == 0.0 && alon == 0.0) {
		return -1.0
	}

	const R = 6371.0 // Earth's radius in kilometers

	// Convert latitude and longitude to radians
	hlatRad := hlat * math.Pi / 180.0
	hlonRad := hlon * math.Pi / 180.0
	alatRad := alat * math.Pi / 180.0
	alonRad := alon * math.Pi / 180.0

	// Calculate differences
	dlat := alatRad - hlatRad
	dlon := alonRad - hlonRad

	// Haversine formula
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(hlatRad)*math.Cos(alatRad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Calculate the distance in kilometers
	kilometers := R * c

	// Convert to miles (1 km = 0.621371 miles)
	const mpk = 0.621371
	miles := kilometers * mpk

	// Round to 2 decimal places
	return math.Round(miles*100) / 100
}

// GetTeamByID retrieves a team by its ID from the database
func GetTeamByID(teamID string) (*Team, error) {
	team := &Team{}
	err := FindByPrimaryKey(team, map[string]interface{}{
		"id": teamID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find team with ID %s: %w", teamID, err)
	}
	return team, nil
}

// CalculateDistanceForTeamIDs calculates distance between two teams by their IDs
func CalculateDistanceForTeamIDs(homeTeamID, awayTeamID string) float64 {
	homeTeam, err := GetTeamByID(homeTeamID)
	if err != nil {
		logger.Debug("Failed to get home team", homeTeamID, err)
		return -1.0
	}

	awayTeam, err := GetTeamByID(awayTeamID)
	if err != nil {
		logger.Debug("Failed to get away team", awayTeamID, err)
		return -1.0
	}

	return CalculateDistance(homeTeam, awayTeam)
}

// NewTeam creates a new Team with default values for numeric fields
// All numeric fields default to -1 (int) or -1.0 (float64) to distinguish from valid zero values
func NewTeam() *Team {
	return &Team{
		ID:          -1,
		CurrentForm: -1,
		Latitude:    -1.0,
		Longitude:   -1.0,
	}
}
/////////////////////////////////////////////////////////////////////////

// UpdateFormData updates form using quaternary encoding (following PODDS methodology)
func UpdateFormData(previousForm int, result int) int {
	// Convert previous form from decimal to quaternary (base-4)
	s := Quaternary(previousForm)

	// Add new result to the front (most recent)
	s = fmt.Sprintf("%d%s", result, s)

	// Keep only last 5 results (rolling window)
	if len(s) > 5 {
		s = s[:5]
	}

	// Convert back to decimal for storage
	ret := 0
	multiplier := 1
	for i := len(s) - 1; i >= 0; i-- {
		digit := int(s[i] - '0')
		ret += digit * multiplier
		multiplier *= 4
	}

	return ret
}

// Quaternary converts decimal to quaternary (base-4) string
func Quaternary(n int) string {
	if n == 0 {
		return "0"
	}

	var nums []string
	for n > 0 {
		remainder := n % 4
		nums = append([]string{fmt.Sprintf("%d", remainder)}, nums...)
		n = n / 4
	}

	return strings.Join(nums, "")
}

// Searches the Teams array for the given team (by ID)
func ExistsInTeamsArray(teams []*Team, team *Team) bool {
	if teams == nil || team == nil {
		return false
	}
	for _, t := range teams {
		if t.ID == team.ID {
			return true
		}
	}
	return false
}
