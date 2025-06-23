package podds

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/richard-senior/mcp/internal/logger"
	"github.com/richard-senior/mcp/pkg/transport"
)

// FotmobDatasource provides methods to fetch data from Fotmob
type FotmobDatasource struct {
	BaseURL      string
	MatchesURL   string
	LeaguesURL   string
	TeamsURL     string
	PlayerURL    string
	MatchDetails string
	SearchURL    string
}

// NewFotmobDatasource creates a new instance of FotmobDatasource
func NewFotmobDatasource() *FotmobDatasource {
	baseURL := "https://www.fotmob.com/api"
	return &FotmobDatasource{
		BaseURL:      baseURL,
		MatchesURL:   fmt.Sprintf("%s/matches?", baseURL),
		LeaguesURL:   fmt.Sprintf("%s/leagues?", baseURL),
		TeamsURL:     fmt.Sprintf("%s/teams?", baseURL),
		PlayerURL:    fmt.Sprintf("%s/playerData?", baseURL),
		MatchDetails: fmt.Sprintf("%s/matchDetails?", baseURL),
		SearchURL:    fmt.Sprintf("%s/searchData?", baseURL),
	}
}

/////////////////////////////////////////////////////////////////////////
////// Persistance
/////////////////////////////////////////////////////////////////////////

// BulkLoadData loads match data for specified leagues and seasons
func BulkLoadData() error {
	// Initialize database
	if err := InitDatabase(poddsDbPath); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer CloseDatabase()

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Define leagues and seasons to load
	leagues := []int{47, 48, 108} // Premier League, Championship, La Liga
	seasons := generateSeasons(2015, 2024)

	// Initialize data source
	datasource := NewFotmobDatasource()

	logger.Info("Starting bulk data load for leagues", leagues, "seasons", seasons)

	// Load data for each league/season combination
	for _, leagueID := range leagues {
		for _, season := range seasons {
			logger.Info("Loading data for league", leagueID, "season", season)

			if err := loadLeagueSeasonData(datasource, leagueID, season); err != nil {
				logger.Error("Failed to load data for league", leagueID, "season", season, "error", err)
				continue // Continue with next league/season
			}

			logger.Info("Successfully loaded data for league", leagueID, "season", season)
		}
	}

	logger.Info("Bulk data load completed")
	return nil
}

// loadLeagueSeasonData loads and processes data for a specific league/season
func loadLeagueSeasonData(datasource *FotmobDatasource, leagueID int, season string) error {
	// Get matches from Fotmob
	matches, err := datasource.GetMatches(leagueID, season)
	if err != nil {
		return fmt.Errorf("failed to get matches: %w", err)
	}

	logger.Info("Retrieved matches", len(matches), "for league", leagueID, "season", season)

	// Set league ID and season for all matches
	for _, match := range matches {
		match.LeagueID = leagueID
		match.Season = season
	}

	// Extract and save teams
	teams := ExtractTeamsFromMatches(matches)
	if err := SaveTeams(teams); err != nil {
		return fmt.Errorf("failed to save teams: %w", err)
	}

	// Save matches
	if err := SaveMatches(matches); err != nil {
		return fmt.Errorf("failed to save matches: %w", err)
	}

	// Process and save team statistics
	if err := ProcessAndSaveTeamStats(matches, leagueID, season); err != nil {
		return fmt.Errorf("failed to process team stats: %w", err)
	}

	return nil
}

// generateSeasons creates season strings from start year to end year
func generateSeasons(startYear, endYear int) []string {
	var seasons []string
	for year := startYear; year <= endYear; year++ {
		season := fmt.Sprintf("%d/%d", year, year+1)
		seasons = append(seasons, season)
	}
	return seasons
}

/////////////////////////////////////////////////////////////////////////
////// Transport and Parsing
/////////////////////////////////////////////////////////////////////////

// get performs an HTTP GET request to the specified URL
func (f *FotmobDatasource) get(url string) ([]byte, error) {
	ret, err := transport.GetHtml(url)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// GetLeagueFromScreenScrape fetches match data for any given season by screen scraping the Fotmob website
func (f *FotmobDatasource) GetLeagueData(leagueID int, season string) (map[string]any, error) {

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
		return nil, fmt.Errorf("failed to fetch data from Fotmob: %w", err)
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

// GetLeagueFromScreenScrape fetches match data for any given season by screen scraping the Fotmob website
func (f *FotmobDatasource) GetMatches(leagueID int, season string) ([]*Match, error) {

	// first see if this exists in cache
	safeSeason := strings.ReplaceAll(season, "/", "-")
	cacheFilename := fmt.Sprintf(poddsCachePath+"fotmob-%d-%s-league.json", leagueID, safeSeason)
	// create a variable to hold the season data
	var pageProps map[string]any
	// does cache file exist?
	_, err := os.Stat(cacheFilename)
	if err != nil {
		d, err := f.GetLeagueData(leagueID, season)
		if err != nil {
			return nil, fmt.Errorf("error fetching league data: %w", err)
		}

		// Extract the league data from the props.pageProps path
		props, ok := d["props"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("could not find 'props' in data")
		}

		pageProps, ok := props["pageProps"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("could not find 'pageProps' in props")
		}
		// write to cache
		cacheData, err := json.MarshalIndent(pageProps, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling pageProps to JSON: %w", err)
		}
		if err := os.WriteFile(cacheFilename, cacheData, 0644); err != nil {
			return nil, fmt.Errorf("error writing cache file %s: %w", cacheFilename, err)
		}
	} else {
		// read from cache
		cacheData, err := os.ReadFile(cacheFilename)
		if err != nil {
			return nil, fmt.Errorf("error reading cache file %s: %w", cacheFilename, err)
		}
		if err := json.Unmarshal(cacheData, &pageProps); err != nil {
			return nil, fmt.Errorf("error unmarshaling cache file %s: %w", cacheFilename, err)
		}
	}
	// get matches
	matches, err := f.extractMatches(pageProps)
	if err != nil {
		return nil, fmt.Errorf("error extracting matches: %w", err)
	}
	return matches, nil
}

// extractMatches extracts and parses matches from pageProps data
func (f *FotmobDatasource) extractMatches(pageProps map[string]any) ([]*Match, error) {
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
