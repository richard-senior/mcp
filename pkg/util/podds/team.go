package podds

import (
	"fmt"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/util"
)

// Team represents a team in a match with database persistence annotations
type Team struct {
	ID          int     `json:"id" column:"id" dbtype:"INTEGER" primary:"true" index:"true"`
	Name        string  `json:"shortName" column:"name" dbtype:"TEXT NOT NULL"`
	CurrentForm int     `json:"form,omitempty" column:"currentForm" dbtype:"INTEGER"`
	Latitude    float64 `json:"latitude,omitempty" column:"latitude" dbtype:"REAL"`
	Longitude   float64 `json:"longitude,omitempty" column:"longitude" dbtype:"REAL"`
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
