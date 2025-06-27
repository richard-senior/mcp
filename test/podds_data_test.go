package test

import (
	"testing"

	"github.com/richard-senior/mcp/pkg/util/podds"
)

func TestTeamData(t *testing.T) {
	t.Log("--TEAMS LIST--")
	data := podds.GetDataInstance()
	// iterate the map data.TeamsData and echo out the key and the value of fotmobid ie Accrington:8671, Aldershot:8465 etc.
	for k, v := range data.TeamsData {
		t.Log(k, v.Id)
	}
	t.Log("--TEAMS LIST--")
}

// Tests the online lookup of teamname for fotmobID
/*
func TestLookupTeamName(t *testing.T) {
	tn, err := podds.LookupTeamNameForId(8671)
	if err != nil {
		t.Error(err)
	}
	t.Log("Team name is " + tn)
}
*/

// TestInitialPositionsValidation validates INITIAL_POSITIONS against FINAL_POSITIONS from previous seasons
func TestInitialPositionsValidation(t *testing.T) {
	data := podds.GetDataInstance()
	for _, leagueID := range podds.Leagues {
		for _, season := range podds.Seasons {
			sy, err := podds.GetSecondYear(season)
			if podds.CurrentSeasonSecondYear == sy {
				// we have no finishing data for this season as it hasn't finished
				// for this year we must use AI to validate in some way against the table data
				continue
			}
			fp, err := data.GetFinalPositions(leagueID, season)
			if err != nil {
				t.Error(err)
			}
			ip, err := data.GetInitialPositions(leagueID, season)
			if err != nil {
				t.Error(err)
			}
			// now ensure that all teams in fp.Positions exist in ip.Positions
			// Positions is of type map[string]int so we need to
			// the team ID's are in the value of the map not the key
			for _, team := range fp.Positions {
				found := false
				for _, team2 := range ip.Positions {
					if team == team2 {
						found = true
						break
					}
				}
				if !found {
					// raise test error in the appropirate format
					t.Error("Team", team, "not found in INITIAL_POSITIONS for league", leagueID, "season", season)
				}
			}
		}
	}

}
