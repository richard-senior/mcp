package podds

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

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
	TeamStats    []*TeamStats
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
			// get fotmob data from cache or remote
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
			//process all the data
			fotmobMatches, err := datasource.processFotmobMatchData(pageProps, existingMatches)
			if err != nil {
				return fmt.Errorf("error processing fotmob match data: %w", err)
			}

			csvData, err := datasource.GetFootballData(leagueID, season)
			if err != nil {
				return fmt.Errorf("error getting football data: %w", err)
			}
			footballDataMatches, err := datasource.ParseFootballDataCSV(csvData, leagueID, season)
			if err != nil {
				return fmt.Errorf("error parsing football data: %w", err)
			}
			nds, err := datasource.ProcessLeagueMatches(fotmobMatches, footballDataMatches)
			if err != nil || nds == nil || nds.Teams == nil || nds.Matches == nil || nds.TeamStats == nil {
				return fmt.Errorf("error calculating stats or predictions: %w", err)
			}
			datasource.Matches = nds.Matches
			datasource.Teams = nds.Teams
			datasource.TeamStats = nds.TeamStats

			// now persist all this
			//save teams to database
			if err := SaveTeams(datasource.Teams); err != nil {
				return fmt.Errorf("failed to save Teams: %w", err)
			}
			if err := SaveTeamStats(datasource.TeamStats); err != nil {
				return fmt.Errorf("failed to save TeamStats: %w", err)
			}
			// Save matches to database
			if err := SaveMatches(datasource.Matches); err != nil {
				return fmt.Errorf("failed to save Matches: %w", err)
			}
		}
	}
	logger.Info("Bulk data load completed")
	return nil
}

// Takes matches from fotmob (fm), and football-data (fdm) and merges them, and process them using Match.Merge
// Returns a new instance of Datasource since this is a convenient way of returning this data
// this method uses a kind of IOC so it may be called in order to unit test update core prediction functionality
func (datasource *Datasource) ProcessLeagueMatches(fm []*Match, fdm []*Match) (*Datasource, error) {
	if fm == nil || len(fm) == 0 {
		return &Datasource{}, fmt.Errorf("fotmob matches were empty")
	}
	var leagueID int
	var season string
	for _, m := range fm {
		leagueID = m.LeagueID
		season = m.Season
		for _, n := range fdm {
			if m.Equals(n) {
				m.Merge(n)
			}
		}
	}
	ret := &Datasource{}
	teams := ExtractTeamsFromMatches(fm)
	ret.Teams = teams
	ts, err := ProcessTeamStats(fm, leagueID, season)
	if err != nil {
		return nil, fmt.Errorf("failed to process team stats: %w", err)
	}
	ret.TeamStats = ts
	ret.Matches = fm

	// Run Poisson predictions for future matches before saving
	for _, match := range ret.Matches {
		err := PredictMatch(match, ts)
		if err != nil {
			logger.Warn("Failed to predict match", match.HomeTeamName, "vs", match.AwayTeamName, err)
		}
	}
	return ret, nil
}

/**
* ProcessData takes the raw match and team data and returns an array of partially populated matches
 */
func (datasource *Datasource) processFotmobMatchData(pageProps map[string]any, existingMatches map[string]*Match) ([]*Match, error) {
	// get leagueId and season from pageProps
	// does pageProps have a 'details' key?
	details, ok := pageProps["details"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Failed to find details stanza in pageProps")
	}

	id, ok := details["id"].([]any)
	if !ok {
		return nil, fmt.Errorf("Failed to find league id pageProps#details")
	}
	leagueID, err := util.GetAsInteger(id)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert pageProps#details#id (%w) to an integer", id)
	}
	s, ok := details["selectedSeason"].([]any)
	if !ok {
		return nil, fmt.Errorf("Failed to find season pageProps#details")
	}
	season, err := util.GetAsString(s)
	if err != nil {
		return nil, fmt.Errorf("Failed convert season (%w) to string", s)
	}
	// lets start by processing and bulk saving matches etc.
	// parse the pageProps to get an array of matches for this season
	matches, err := datasource.extractMatchesWithCache(pageProps, existingMatches)
	if err != nil {
		return nil, fmt.Errorf("error extracting matches: %w", err)
	}

	// Set league ID and season for all matches
	for _, match := range matches {
		match.LeagueID = leagueID
		match.Season = season
	}
	return matches, nil
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

// GetLeagueFromScreenScrape fetches (from fotmob) match data for any given season by screen scraping the external website
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
		// TODO find a way of doing this without parsing all the match data
		// before we've checked if this match is in the database already
		matchJSON, err := json.Marshal(matchData)
		if err != nil {
			return nil, fmt.Errorf("error marshaling match %d to JSON: %w", i, err)
		}

		// Parse JSON into Match struct
		newMatch, err := ParseMatchFromJSON(matchJSON)
		if err != nil {
			return nil, fmt.Errorf("error parsing match %d: %w", i, err)
		}
		newMatch.CreatedAt = time.Now()

		if existingMatch, exists := existingMatches[newMatch.ID]; exists {
			if existingMatch.ShouldProcess() {
				matches = append(matches, newMatch)
			} else {
				matches = append(matches, existingMatch)
			}
		} else {
			matches = append(matches, newMatch)
		}
	}
	return matches, nil
}

// //////////////////////////////////////////////////////////////////////
// Football-Data.co.uk
// //////////////////////////////////////////////////////////////////////

func (f *Datasource) GetFootballData(leagueID int, season string) (string, error) {
	// Validate inputs
	if leagueID <= 0 {
		return "", fmt.Errorf("must supply a valid leagueID")
	}

	seasonPattern := regexp.MustCompile(`^\d{4}/\d{4}$`)
	if !seasonPattern.MatchString(season) {
		return "", fmt.Errorf("season must be in the format 'yyyy/yyyy'")
	}

	// Convert fotmob league ID to football-data.co.uk league code
	leagueCode, err := f.FotmobLeagueIDToNative(leagueID)
	if err != nil {
		return "", fmt.Errorf("unsupported league ID %d: %w", leagueID, err)
	}

	// Convert season format from "2024/2025" to "2425"
	nativeSeason := f.FotmobSeasonToNative(season)

	// Generate cache filename
	safeSeason := strings.ReplaceAll(season, "/", "-")
	cacheFilename := fmt.Sprintf("%sraw-league-csv-%s-%d.csv", Config.PoddsCachePath, safeSeason, leagueID)

	// Check if we should delete cache for current season (to get fresh data)
	if IsCurrentSeason(season) {
		if _, err := os.Stat(cacheFilename); err == nil {
			logger.Info("Deleting stale cache file for current season:", cacheFilename)
			os.Remove(cacheFilename)
		}
	}

	var csvData string = ""

	// Try to read from cache first
	if cacheData, err := os.ReadFile(cacheFilename); err == nil {
		csvData = string(cacheData)
		logger.Debug("Returning data from cached file for", leagueID, season)
	} else {
		// Cache miss - fetch from football-data.co.uk
		logger.Info("Fetching historical data from football-data.co.uk for", leagueID, season)
		url := fmt.Sprintf("https://www.football-data.co.uk/mmz4281/%s/%s.csv", nativeSeason, leagueCode)
		response, err := f.get(url)
		if err != nil {
			return "", fmt.Errorf("failed to fetch data from external source: %w", err)
		}
		csvData = string(response)
		// Cache the data
		if err := os.WriteFile(cacheFilename, []byte(csvData), 0644); err != nil {
			logger.Warn("Failed to write cache file", cacheFilename, err)
			// Continue processing even if caching fails
		} else {
			logger.Info("Cached data to", cacheFilename)
		}
	}
	return csvData, nil

}

// Given a CSV in string format, parses each row as a Match Object
func (f *Datasource) ParseFootballDataCSV(csvData string, leagueID int, season string) ([]*Match, error) {

	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return []*Match{}, nil
	}

	// Get header row
	headers := records[0]

	// Clean up first header if it has BOM or other issues
	if len(headers) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff") // Remove BOM
		if headers[0] == "" || strings.Contains(headers[0], "Div") {
			headers[0] = "Div"
		}
	}

	var matches []*Match

	// Process data rows
	for i, record := range records[1:] {
		if len(record) < len(headers) {
			logger.Warn("Skipping incomplete record at row", i+2)
			continue
		}

		// Create row map
		row := make(map[string]string)
		for j, value := range record {
			if j < len(headers) {
				row[headers[j]] = strings.TrimSpace(value)
			}
		}

		// Skip empty rows
		if row["HomeTeam"] == "" || row["AwayTeam"] == "" {
			continue
		}

		match, err := f.ParseFootballDataRow(row, leagueID, season)
		if err != nil {
			logger.Warn("Failed to parse match at row", i+2, err)
			continue
		}

		if match != nil {
			matches = append(matches, match)
		}
	}

	return matches, nil
}

// ParseFootballDataRow converts a CSV row from football-data.co.uk to a Match struct
// This is a public version of parseFootballDataRow for testing purposes
func (f *Datasource) ParseFootballDataRow(row map[string]string, leagueID int, season string) (*Match, error) {
	// Extract team names
	homeTeamName := strings.TrimSpace(row["HomeTeam"])
	awayTeamName := strings.TrimSpace(row["AwayTeam"])

	if homeTeamName == "" || awayTeamName == "" {
		return nil, fmt.Errorf("missing team names")
	}

	// Clean team names (remove non-alphabetic characters except spaces)
	re := regexp.MustCompile(`[^a-zA-Z ]`)
	homeTeamName = strings.TrimSpace(re.ReplaceAllString(homeTeamName, ""))
	awayTeamName = strings.TrimSpace(re.ReplaceAllString(awayTeamName, ""))

	// Look up team IDs from our data
	homeTeamID, err := f.getTeamIDForName(homeTeamName)
	if err != nil {
		logger.Warn("Could not find team ID for home team:", homeTeamName)
		return nil, err
	}

	awayTeamID, err := f.getTeamIDForName(awayTeamName)
	if err != nil {
		logger.Warn("Could not find team ID for away team:", awayTeamName)
		return nil, err
	}

	match := NewMatch()
	match.LeagueID = leagueID
	match.Season = season
	match.HomeID = strconv.Itoa(homeTeamID)
	match.AwayID = strconv.Itoa(awayTeamID)
	match.Status = "finished"
	match.HomeTeamName = homeTeamName
	match.AwayTeamName = awayTeamName

	// Parse match date and time
	if dateStr := row["Date"]; dateStr != "" {
		if parsedTime, err := f.parseFootballDataDateTime(row); err == nil {
			match.UTCTime = parsedTime
		}
	}

	// Parse actual goals (if match is finished)
	if fthgStr := row["FTHG"]; fthgStr != "" {
		if fthg, err := strconv.Atoi(fthgStr); err == nil {
			match.ActualHomeGoals = fthg
		} else {
			match.ActualHomeGoals = -1
		}
	} else {
		match.ActualHomeGoals = -1
	}

	if ftagStr := row["FTAG"]; ftagStr != "" {
		if ftag, err := strconv.Atoi(ftagStr); err == nil {
			match.ActualAwayGoals = ftag
		} else {
			match.ActualAwayGoals = -1
		}
	} else {
		match.ActualAwayGoals = -1
	}

	// Parse half-time goals
	if hthgStr := row["HTHG"]; hthgStr != "" {
		if hthg, err := strconv.Atoi(hthgStr); err == nil {
			match.ActualHalfTimeHomeGoals = hthg
		} else {
			match.ActualHalfTimeHomeGoals = -1
		}
	} else {
		match.ActualHalfTimeHomeGoals = -1
	}

	if htagStr := row["HTAG"]; htagStr != "" {
		if htag, err := strconv.Atoi(htagStr); err == nil {
			match.ActualHalfTimeAwayGoals = htag
		} else {
			match.ActualHalfTimeAwayGoals = -1
		}
	} else {
		match.ActualHalfTimeAwayGoals = -1
	}

	// Parse shots on target
	if hstStr := row["HST"]; hstStr != "" {
		if hst, err := strconv.Atoi(hstStr); err == nil {
			match.HomeShotsOnTarget = hst
		}
	}

	if astStr := row["AST"]; astStr != "" {
		if ast, err := strconv.Atoi(astStr); err == nil {
			match.AwayShotsOnTarget = ast
		}
	}

	// Parse corners
	if hcStr := row["HC"]; hcStr != "" {
		if hc, err := strconv.Atoi(hcStr); err == nil {
			match.HomeCorners = hc
		}
	}

	if acStr := row["AC"]; acStr != "" {
		if ac, err := strconv.Atoi(acStr); err == nil {
			match.AwayCorners = ac
		}
	}

	// Parse yellow cards
	if hyStr := row["HY"]; hyStr != "" {
		if hy, err := strconv.Atoi(hyStr); err == nil {
			match.HomeYellowCards = hy
		}
	}

	if ayStr := row["AY"]; ayStr != "" {
		if ay, err := strconv.Atoi(ayStr); err == nil {
			match.AwayYellowCards = ay
		}
	}

	// Parse red cards
	if hrStr := row["HR"]; hrStr != "" {
		if hr, err := strconv.Atoi(hrStr); err == nil {
			match.HomeRedCards = hr
		}
	}

	if arStr := row["AR"]; arStr != "" {
		if ar, err := strconv.Atoi(arStr); err == nil {
			match.AwayRedCards = ar
		}
	}

	// Calculate average betting odds
	homeOdds, drawOdds, awayOdds := f.AverageOdds(row)
	match.ActualHomeOdds = homeOdds
	match.ActualDrawOdds = drawOdds
	match.ActualAwayOdds = awayOdds

	// Set referee if available
	if referee := row["Referee"]; referee != "" {
		match.Referee = referee
	}

	return match, nil
}

func (f *Datasource) GetMatchesFromFootballData(csvData string, leagueID int, season string) ([]*Match, error) {
	if csvData == "" {
		return nil, fmt.Errorf("no csv data given")
	}

	// Parse CSV data
	matches, err := f.ParseFootballDataCSV(csvData, leagueID, season)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV data: %w", err)
	}

	logger.Info("Processed", len(matches), "matches from football-data.co.uk for", leagueID, season)
	return matches, nil
}

// FotmobLeagueIDToNative converts fotmob league ID to football-data.co.uk league code
func (f *Datasource) FotmobLeagueIDToNative(leagueID int) (string, error) {
	leagues := map[int]string{
		47:  "E0", // Premier League
		48:  "E1", // Championship
		108: "E2", // League One
		109: "E3", // League Two
	}

	if code, exists := leagues[leagueID]; exists {
		return code, nil
	}

	return "", fmt.Errorf("unsupported league ID: %d", leagueID)
}

// FotmobSeasonToNative converts season format from "2024/2025" to "2425"
func (f *Datasource) FotmobSeasonToNative(season string) string {
	if len(season) != 9 {
		return season // Return as-is if not in expected format
	}
	// Extract last 2 digits of each year: "2024/2025" -> "2425"
	return season[2:4] + season[7:9]
}

// parseFootballDataDateTime parses date and time from football-data.co.uk format
// Matches the Python implementation in getUtcTimeFromSourceDataRow
func (f *Datasource) parseFootballDataDateTime(row map[string]string) (time.Time, error) {
	// Check if we already have a utcTime field (already converted)
	if utcTime, exists := row["utcTime"]; exists && utcTime != "" {
		return time.Parse(time.RFC3339, utcTime)
	}

	// Must have a Date field
	dateStr, exists := row["Date"]
	if !exists || dateStr == "" {
		return time.Time{}, fmt.Errorf("no Date field found")
	}

	// Build datetime string - combine Date and Time fields like Python implementation
	var dtStr string
	if timeStr, hasTime := row["Time"]; hasTime && timeStr != "" {
		// Combine date and time: "DD/MM/YYYY HH:MM" or "DD/MM/YY HH:MM"
		dtStr = strings.TrimSpace(dateStr) + " " + strings.TrimSpace(timeStr)
	} else {
		// No time field, default to 15:00 (3PM) like Python implementation
		dtStr = strings.TrimSpace(dateStr) + " 15:00"
	}

	// Try date+time formats in same order as Python implementation
	// Python tries: '%d/%m/%Y %H:%M' first, then '%d/%m/%y %H:%M'
	dateTimeFormats := []string{
		"02/01/2006 15:04", // DD/MM/YYYY HH:MM (matches Python '%d/%m/%Y %H:%M')
		"02/01/06 15:04",   // DD/MM/YY HH:MM (matches Python '%d/%m/%y %H:%M')
	}

	var parsedTime time.Time
	var parseErr error

	for _, format := range dateTimeFormats {
		if t, err := time.Parse(format, dtStr); err == nil {
			parsedTime = t
			parseErr = nil
			break
		} else {
			parseErr = err
		}
	}

	if parseErr != nil {
		return time.Time{}, fmt.Errorf("could not parse date from %s: %w", dtStr, parseErr)
	}

	// Convert from GMT/London time to UTC like Python implementation
	// Python: london_tz.localize(d) then d.astimezone(pytz.UTC)
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		// Fallback: assume the parsed time is already in UTC
		return parsedTime.UTC(), nil
	}

	// Create the time in London timezone (GMT/BST depending on date)
	londonTime := time.Date(
		parsedTime.Year(), parsedTime.Month(), parsedTime.Day(),
		parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(),
		parsedTime.Nanosecond(), loc,
	)

	// Convert to UTC
	return londonTime.UTC(), nil
}

// getTeamIDForName looks up team ID by name using the existing data
func (f *Datasource) getTeamIDForName(teamName string) (int, error) {
	// This would need to use the existing team data lookup
	// For now, we'll use a simple approach and extend as needed
	teamData, err := TData.GetDataForTeam(teamName)
	if err != nil {
		return -1, fmt.Errorf("team not found: %s - %w", teamName, err)
	}
	if teamData != nil {
		return teamData.Id, nil
	}

	return -1, fmt.Errorf("team not found: %s", teamName)
}

// AverageOdds calculates average betting odds from football-data.co.uk CSV row
// Returns (homeOdds, drawOdds, awayOdds) or (-1.0, -1.0, -1.0) if no odds available
func (f *Datasource) AverageOdds(row map[string]string) (float64, float64, float64) {

	// Check if we already have calculated odds
	if !f.FieldIsBlank("aho", row) {
		ho, _ := strconv.ParseFloat(row["aho"], 64)
		do, _ := strconv.ParseFloat(row["ado"], 64)
		ao, _ := strconv.ParseFloat(row["aao"], 64)
		return ho, do, ao
	}

	// Check for average closing odds
	if !f.FieldIsBlank("AvgCH", row) {
		ho, _ := strconv.ParseFloat(row["AvgCH"], 64)
		do, _ := strconv.ParseFloat(row["AvgCD"], 64)
		ao, _ := strconv.ParseFloat(row["AvgCA"], 64)
		return ho, do, ao
	}

	// Check for average pre-match odds
	if !f.FieldIsBlank("AvgH", row) {
		ho, _ := strconv.ParseFloat(row["AvgH"], 64)
		do, _ := strconv.ParseFloat(row["AvgD"], 64)
		ao, _ := strconv.ParseFloat(row["AvgA"], 64)
		return ho, do, ao
	}

	// Calculate our own averages from individual bookmaker odds
	bookies := []string{"B365", "BF", "BS", "BW", "GB", "IW", "LB", "PS", "SO", "SB", "SJ", "SY", "VC", "WH"}

	var homeTotal, drawTotal, awayTotal float64
	var count int

	// Try both closing odds (with "C" suffix) and regular odds (no suffix)
	for _, suffix := range []string{"C", ""} {
		for _, bookie := range bookies {
			homeKey := bookie + suffix + "H"
			drawKey := bookie + suffix + "D"
			awayKey := bookie + suffix + "A"

			if !f.FieldIsBlank(homeKey, row) {
				if homeOdds, err := strconv.ParseFloat(row[homeKey], 64); err == nil {
					if drawOdds, err := strconv.ParseFloat(row[drawKey], 64); err == nil {
						if awayOdds, err := strconv.ParseFloat(row[awayKey], 64); err == nil {
							homeTotal += homeOdds
							drawTotal += drawOdds
							awayTotal += awayOdds
							count++
						}
					}
				}
			}
		}

		// If we found odds with this suffix, calculate averages and return
		if count > 0 {
			avgHome := homeTotal / float64(count)
			avgDraw := drawTotal / float64(count)
			avgAway := awayTotal / float64(count)

			// Round to 2 decimal places
			avgHome = math.Round(avgHome*100) / 100
			avgDraw = math.Round(avgDraw*100) / 100
			avgAway = math.Round(avgAway*100) / 100

			return avgHome, avgDraw, avgAway
		}
	}

	// No odds found
	return -1.0, -1.0, -1.0
}

// FieldIsBlank checks if a field in the row is blank/empty/missing
func (f *Datasource) FieldIsBlank(field string, row map[string]string) bool {
	if field == "" {
		return true
	}

	value, exists := row[field]
	if !exists {
		return true
	}

	return f.valueIsBlank(value)
}

// valueIsBlank checks if a value is considered blank/empty
func (f *Datasource) valueIsBlank(value string) bool {
	if value == "" {
		return true
	}

	// Check if it's -1 (integer)
	if intVal, err := strconv.Atoi(value); err == nil && intVal == -1 {
		return true
	}

	// Check if it's -1.0 (float)
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil && floatVal == -1.0 {
		return true
	}

	return false
}

// generateMatchID generates a unique match ID from team IDs and date
func (f *Datasource) generateMatchID(homeTeamID, awayTeamID int, matchTime time.Time) string {
	// Generate ID similar to how the Python code does it
	dateStr := matchTime.Format("20060102")
	return fmt.Sprintf("%s_%d_%d", dateStr, homeTeamID, awayTeamID)
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
