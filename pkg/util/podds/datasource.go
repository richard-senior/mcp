package podds

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/transport"
	"github.com/richard-senior/mcp/pkg/util"
)

// Datasource provides methods to fetch football data from external sources
type Datasource struct {
	BaseURL      string
	MatchesURL   string
	LeaguesURL   string
	TeamsURL     string
	PlayerURL    string
	MatchDetails string
	SearchURL    string
	Teams        []*Team
	Matches      []*Match
}

var (
	datasourceInstance *Datasource
	datasourceOnce     sync.Once
)

// GetDatasource returns the singleton instance of Datasource
func GetDatasourceInstance() *Datasource {
	datasourceOnce.Do(func() {
		baseURL := "https://www.fotmob.com/api"
		datasourceInstance = &Datasource{
			BaseURL:      baseURL,
			MatchesURL:   fmt.Sprintf("%s/matches?", baseURL),
			LeaguesURL:   fmt.Sprintf("%s/leagues?", baseURL),
			TeamsURL:     fmt.Sprintf("%s/teams?", baseURL),
			PlayerURL:    fmt.Sprintf("%s/playerData?", baseURL),
			MatchDetails: fmt.Sprintf("%s/matchDetails?", baseURL),
			SearchURL:    fmt.Sprintf("%s/searchData?", baseURL),
			Teams:        make([]*Team, 0),
			Matches:      make([]*Match, 0),
		}
		err := datasourceInstance.Update()
		if err != nil {
			logger.Error("Error loading data", err)
		}
	})
	return datasourceInstance
}

/////////////////////////////////////////////////////////////////////////
////// Persistance and Updating
/////////////////////////////////////////////////////////////////////////

// BulkLoadData loads match data for specified leagues and seasons
func (datasource *Datasource) Update() error {
	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(Config.PoddsCachePath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// to start Load data for each league/season combination from fotmob
	for _, leagueID := range Config.Leagues {
		for _, season := range Config.Seasons {
			logger.Info("Loading data for league", leagueID, "season", season)

			// Pre-load existing matches from database for this league/season
			existingMatches, err := LoadExistingMatches(leagueID, season)
			if err != nil {
				logger.Warn("Failed to load existing matches for", leagueID, season, err)
				existingMatches = make(map[string]*Match) // Empty cache on error
			} else {
				logger.Info("Pre-loaded", len(existingMatches), "existing matches for", leagueID, season)
			}

			safeSeason := strings.ReplaceAll(season, "/", "-")
			cacheFilename := fmt.Sprintf(Config.PoddsCachePath+"fotmob-%d-%s-league.json", leagueID, safeSeason)
			var pageProps map[string]any
			// load cache file if it exists
			_, err = os.Stat(cacheFilename)
			if err == nil {
				// File exists, read from cache
				cacheData, err := os.ReadFile(cacheFilename)
				if err == nil {
					if ner := json.Unmarshal(cacheData, &pageProps); ner != nil {
						return fmt.Errorf("error unmarshaling cache file %s: %w", cacheFilename, ner)
					}
					logger.Info("Loaded data from cache:", cacheFilename)
				} else {
					return fmt.Errorf("error reading cache file, perhaps consider deleting cache files %s: %w", cacheFilename, err)
				}
			} else {
				// File doesn't exist, fetch new data
				logger.Warn("league/season not in cache: ", leagueID, season)
				// fetch and cache
				d, err := datasource.getLeagueData(leagueID, season)
				if err != nil {
					return fmt.Errorf("error fetching league data: %w", err)
				}
				// Extract the league data from the props.pageProps path
				props, ok := d["props"].(map[string]any)
				if !ok {
					return fmt.Errorf("could not find 'props' in data")
				}
				// populate our variable
				pageProps, ok := props["pageProps"].(map[string]any)
				if !ok {
					return fmt.Errorf("could not find 'pageProps' in props")
				}
				// write to cache
				cacheData, err := json.MarshalIndent(pageProps, "", "  ")
				if err != nil {
					return fmt.Errorf("error marshaling pageProps to JSON: %w", err)
				}
				// ok cache this
				if err := os.WriteFile(cacheFilename, cacheData, 0644); err != nil {
					return fmt.Errorf("error writing cache file %s: %w", cacheFilename, err)
				}
			}

			// lets start by processing and bulk saving matches etc.
			matches, err := datasource.extractMatchesWithCache(pageProps, existingMatches)
			if err != nil {
				return fmt.Errorf("error extracting matches: %w", err)
			}

			// Set league ID and season for all matches
			for _, match := range matches {
				match.LeagueID = leagueID
				match.Season = season
			}

			// Extract and save teams
			teams := ExtractTeamsFromMatches(matches)

			// Amend the teams list with any that are found in Fallback
			fallbackTeams, err := datasource.getFallbackTeams(pageProps)
			if err == nil && fallbackTeams != nil {
				//logger.Info("Got Fallback teams", len(fallbackTeams))
				for _, t := range datasource.Teams {
					if !ExistsInTeamsArray(teams, t) {
						tdata, err := TData.GetDataForTeam(t.ID)
						if err == nil && tdata != nil {
							//logger.Highlight("Adding team", tdata.Name)
							foo := &Team{
								ID:        tdata.Id,
								Name:      tdata.Name,
								Latitude:  tdata.Latitude,
								Longitude: tdata.Longitude,
							}
							teams = append(teams, foo)
						} else {
							// TODO just add these teams anyway? They're likely foreign teams
							logger.Highlight("Found a team in Fallback which does not exist in data:", t.Name)
						}
					}
				}
			} else {
				logger.Info("Didn't get fallback teams?", err)
			}

			// Remember processed team stats for later during poisson prediction
			var ts []*TeamStats
			// Now process team stats for all teams
			if ts, err = ProcessAndSaveTeamStats(matches, leagueID, season); err != nil {
				return fmt.Errorf("failed to process team stats: %w", err)
			}

			// Persist all data
			// cache teams on our instance
			datasource.Teams = teams
			//save teams to database
			if err := SaveTeams(teams); err != nil {
				return fmt.Errorf("failed to save teams: %w", err)
			}

			// cache matches on our instance
			datasource.Matches = matches

			// Run Poisson predictions for future matches before saving
			for _, match := range matches {
				err := PredictMatch(match, ts)
				if err != nil {
					logger.Warn("Failed to predict match", match.HomeTeamName, "vs", match.AwayTeamName, err)
					// Continue with other matches even if one fails
				}
			}

			// Save matches to database
			if err := SaveMatches(matches); err != nil {
				return fmt.Errorf("failed to save matches: %w", err)
			}

		}
	}

	logger.Info("Bulk data load completed")
	return nil
}

/////////////////////////////////////////////////////////////////////////
////// Transport and Parsing
/////////////////////////////////////////////////////////////////////////

// get performs an HTTP GET request to the specified URL
func (f *Datasource) get(url string) ([]byte, error) {
	logger.Inform("HTTP get called for ", url)
	ret, err := transport.GetHtml(url)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Uses the 'Fallback' section of the pageProps map to get any information about team name to team id mappings
func (f *Datasource) getFallbackTeams(pageProps map[string]any) ([]*Team, error) {
	fb, ok := pageProps["fallback"].(map[string]any)
	if !ok {
		return make([]*Team, 0), fmt.Errorf("could not find 'fallback' in pageProps")
	}
	var v map[string]any
	for k := range fb {
		if val, ok := fb[k].(map[string]any); ok {
			v = val
			break
		} else {
			return make([]*Team, 0), fmt.Errorf("couldn't get the first key in the fallback dictionary")
		}
	}
	if v == nil {
		return make([]*Team, 0), fmt.Errorf("Failed to get the first entry in the fallback dictionary")
	}

	sh, ok := v["Shortened"].(map[string]any)
	if !ok {
		return make([]*Team, 0), fmt.Errorf("couldn't find the Shortened dictionary in the fallback dictionary")
	}
	// ok now iterate the shortend teams map which looks like this
	// {
	//    "10003": "Swansea",
	//    "10005": "Grimsby",
	// etc.
	// }
	// and create a new Team object for each entry
	ret := []*Team{}
	for k := range sh {
		// convert k to an int
		teamID, err := util.GetAsInteger(k)
		if err != nil {
			return make([]*Team, 0), fmt.Errorf("failed to convert teamID to int: %w", err)
		}

		// Type assert the team name to string
		teamName, ok := sh[k].(string)
		if !ok {
			logger.Warn("Team name is not a string for ID", k, "got type:", fmt.Sprintf("%T", sh[k]))
			continue
		}

		t := &Team{
			ID:   teamID,
			Name: teamName,
		}
		// append this to the ret array
		ret = append(ret, t)
	}
	return ret, nil
}

// GetLeagueFromScreenScrape fetches match data for any given season by screen scraping the external website
// Does not cache, this method should be wrapped in a caching mechanism (which is why it's marked private)
func (f *Datasource) getLeagueData(leagueID int, season string) (map[string]any, error) {

	// Validate inputs
	if leagueID <= 0 {
		return nil, fmt.Errorf("must supply a valid leagueID")
	}

	seasonPattern := regexp.MustCompile(`^\d{4}/\d{4}$`)
	if !seasonPattern.MatchString(season) {
		return nil, fmt.Errorf("season must be in the format 'yyyy/yyyy'")
	}

	// TODO check the cache to see if we already have this data

	// Construct the URL
	url := fmt.Sprintf("https://www.fotmob.com/en-GB/leagues/%d/overview?season=%s", leagueID, season)

	// Fetch the HTML content
	htmlContent, err := f.get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from external source: %w", err)
	}
	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	// Find the script tag with id "__NEXT_DATA__"
	var scriptData string
	doc.Find("script#__NEXT_DATA__").Each(func(i int, s *goquery.Selection) {
		scriptData = s.Text()
	})

	if scriptData == "" {
		return nil, fmt.Errorf("could not find __NEXT_DATA__ script tag")
	}

	// Parse the JSON data
	var data map[string]any
	if err := json.Unmarshal([]byte(scriptData), &data); err != nil {
		return nil, fmt.Errorf("error parsing JSON data: %w", err)
	}
	return data, nil
}

// loadExistingMatches loads all existing matches for a specific league/season from database
// Uses the existing persistable FindWhere function for consistency and proper ORM handling
func LoadExistingMatches(leagueID int, season string) (map[string]*Match, error) {
	// Use the existing persistable FindWhere function
	results, err := FindWhere(&Match{}, "leagueId = ? AND season = ?", leagueID, season)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing matches: %w", err)
	}

	matches := make(map[string]*Match)

	for _, result := range results {
		if match, ok := result.(*Match); ok {
			matches[match.ID] = match
		} else {
			logger.Warn("Unexpected type in FindWhere results, expected *Match")
		}
	}

	return matches, nil
}

// extractMatchesWithCache extracts and parses matches from pageProps data, using existing match cache
func (f *Datasource) extractMatchesWithCache(pageProps map[string]any, existingMatches map[string]*Match) ([]*Match, error) {
	var matches []*Match

	// Navigate to matches.allMatches
	matchesData, ok := pageProps["matches"].(map[string]any)
	if !ok {
		return matches, nil // Return empty slice if no matches found
	}

	allMatchesData, ok := matchesData["allMatches"].([]any)
	if !ok {
		return matches, nil // Return empty slice if no allMatches found
	}

	// Parse each match
	for i, matchData := range allMatchesData {
		// Convert match data to JSON bytes for parsing
		matchJSON, err := json.Marshal(matchData)
		if err != nil {
			return nil, fmt.Errorf("error marshaling match %d to JSON: %w", i, err)
		}

		// Parse JSON into Match struct
		newMatch, err := ParseMatchFromJSON(matchJSON)
		if err != nil {
			return nil, fmt.Errorf("error parsing match %d: %w", i, err)
		}

		// Check if match already exists in database
		if existingMatch, exists := existingMatches[newMatch.ID]; exists {
			// Use existing match if it has predictions, otherwise use new match for re-processing
			if existingMatch.PoissonPredictedHomeGoals != -1 {
				logger.Debug("Using existing match with predictions:", existingMatch.HomeTeamName, "vs", existingMatch.AwayTeamName)
				matches = append(matches, existingMatch)
			} else {
				logger.Debug("Re-processing match without predictions:", newMatch.HomeTeamName, "vs", newMatch.AwayTeamName)
				// Copy any existing actual results to new match
				if existingMatch.ActualHomeGoals != -1 {
					newMatch.ActualHomeGoals = existingMatch.ActualHomeGoals
					newMatch.ActualAwayGoals = existingMatch.ActualAwayGoals
				}
				// Copy existing metadata
				newMatch.CreatedAt = existingMatch.CreatedAt
				matches = append(matches, newMatch)
			}
		} else {
			// New match, add to processing list
			logger.Debug("Processing new match:", newMatch.HomeTeamName, "vs", newMatch.AwayTeamName)
			matches = append(matches, newMatch)
		}
	}

	// Log cache performance summary
	var newMatches, existingWithPredictions, existingWithoutPredictions int
	for _, match := range matches {
		if _, exists := existingMatches[match.ID]; exists {
			if match.PoissonPredictedHomeGoals != -1 {
				existingWithPredictions++
			} else {
				existingWithoutPredictions++
			}
		} else {
			newMatches++
		}
	}

	logger.Info("Match processing summary:",
		"New:", newMatches,
		"Existing with predictions:", existingWithPredictions,
		"Existing without predictions:", existingWithoutPredictions,
		"Total:", len(matches))

	return matches, nil
}

// extractMatches extracts and parses matches from pageProps data
func (f *Datasource) extractMatches(pageProps map[string]any) ([]*Match, error) {
	var matches []*Match

	// Navigate to matches.allMatches
	matchesData, ok := pageProps["matches"].(map[string]any)
	if !ok {
		return matches, nil // Return empty slice if no matches found
	}

	allMatchesData, ok := matchesData["allMatches"].([]any)
	if !ok {
		return matches, nil // Return empty slice if no allMatches found
	}

	// Parse each match
	for i, matchData := range allMatchesData {
		// Convert match data to JSON bytes for parsing
		matchJSON, err := json.Marshal(matchData)
		if err != nil {
			return nil, fmt.Errorf("error marshaling match %d to JSON: %w", i, err)
		}

		// Parse JSON into Match struct
		match, err := ParseMatchFromJSON(matchJSON)
		if err != nil {
			return nil, fmt.Errorf("error parsing match %d: %w", i, err)
		}

		matches = append(matches, match)
	}

	return matches, nil
}

/**
* TODO this and it's reciprocal GetNameForTeamID etc
 */
func GetIdForTeamname(team any) (int, error) {
	// use the datasource to get the raw json from any league page
	// in there is a section with id:shortname mappings for all teams
	// under "Fallback.......Shortened"
	return -1, nil
}

// Uses external data source (remote http) to look up the full team name of a team for any given team ID
func LookupTeamNameForId(id int) (string, error) {
	ids, err := util.GetAsString(id)
	if err != nil {
		return "", fmt.Errorf("failed to convert id to string: %w", err)
	}
	url := "https://www.fotmob.com/en-GB/teams/" + ids + "/overview"

	body, err := transport.GetHtml(url)
	if err != nil {
		return "", fmt.Errorf("failed to get html: %w", err)
	}
	// convert body to string
	b := string(body)

	ss := "{\"@type\":\"ListItem\",\"position\":3,\"name\":\""
	st := strings.Index(b, ss)
	if st == -1 {
		return "", fmt.Errorf("Failed to find teamname for id %d", id)
	}
	// now add the length of URL onto loc
	st = st + len(ss)
	// now find the next occurance of a double quote after the loc position in body
	loc2 := strings.Index(b[st:], "\"")
	// extract the text between the two markers
	teamName := b[st : st+loc2]
	// now we have the team name, we can return it
	if len(teamName) > 0 {
		return teamName, nil
	}
	return "", fmt.Errorf("failed to find team name ")
}
