# Football-Data.co.uk Historical Data Fetching

This document describes the new `FetchFootballDataHistorical` method added to the datasource package.

## Overview

The `FetchFootballDataHistorical` method fetches historical match data from https://www.football-data.co.uk/ and caches it locally. This provides an alternative data source for historical matches that have already been played.

## Method Signature

```go
func (f *Datasource) FetchFootballDataHistorical(leagueID int, season string) ([]*Match, error)
```

## Parameters

- `leagueID`: The FotMob league ID (47=Premier League, 48=Championship, 108=League One, 109=League Two)
- `season`: Season in format "YYYY/YYYY" (e.g., "2023/2024")

## Returns

- `[]*Match`: Array of Match structs containing historical match data
- `error`: Any error that occurred during fetching or parsing

## Features

### Caching
- Data is cached locally to avoid repeated HTTP requests
- Cache files are stored in the configured cache directory
- For current season data, cache is automatically refreshed to get latest results
- Cache filename format: `raw-league-csv-{season}-{leagueID}.csv`

### League Support
The method supports the following English football leagues:
- Premier League (ID: 47, Code: E0)
- Championship (ID: 48, Code: E1)  
- League One (ID: 108, Code: E2)
- League Two (ID: 109, Code: E3)

### Data Processing
- Parses CSV data from football-data.co.uk
- Maps team names to internal team IDs using existing team data
- Handles various date formats commonly used by the source
- Creates Match structs compatible with the existing system
- Sets appropriate default values for prediction fields
- **Calculates average betting odds** from multiple bookmakers automatically

## Betting Odds Integration

The method automatically calculates average betting odds from the football-data.co.uk CSV data using the same algorithm as the original Python implementation.

### Odds Calculation Priority
1. **Pre-calculated odds** - Uses existing `aho`, `ado`, `aao` fields if available
2. **Average closing odds** - Uses `AvgCH`, `AvgCD`, `AvgCA` fields if available  
3. **Average pre-match odds** - Uses `AvgH`, `AvgD`, `AvgA` fields if available
4. **Individual bookmaker odds** - Calculates average from individual bookmaker odds

### Supported Bookmakers
The system averages odds from these bookmakers when available:
- **B365** - Bet365
- **BF** - Betfair
- **BS** - Bet&Win
- **BW** - Betway
- **GB** - Gamebookers
- **IW** - Interwetten
- **LB** - Ladbrokes
- **PS** - Pinnacle Sports
- **SO** - Sporting Odds
- **SB** - Sportingbet
- **SJ** - Stan James
- **SY** - Stanleybet
- **VC** - VC Bet
- **WH** - William Hill

### Match Struct Fields
The calculated odds are stored in these new Match struct fields:
- `AverageHomeOdds` - Average odds for home team win
- `AverageDrawOdds` - Average odds for draw
- `AverageAwayOdds` - Average odds for away team win

Values of -1.0 indicate no odds were available for that match.

## Usage Example

```go
// Get datasource instance
ds := podds.GetDatasourceInstance()

// Fetch Premier League 2023/2024 historical data
matches, err := ds.FetchFootballDataHistorical(47, "2023/2024")
if err != nil {
    log.Fatal("Failed to fetch data:", err)
}

fmt.Printf("Fetched %d matches\n", len(matches))

// Process matches as needed
for _, match := range matches {
    fmt.Printf("%s vs %s: %d-%d", 
        match.HomeTeamName, match.AwayTeamName,
        match.ActualHomeGoals, match.ActualAwayGoals)
    
    // Display betting odds if available
    if match.AverageHomeOdds > 0 {
        fmt.Printf(" (Odds: %.2f/%.2f/%.2f)", 
            match.AverageHomeOdds, match.AverageDrawOdds, match.AverageAwayOdds)
    }
    fmt.Println()
}

// Example: Direct odds calculation from CSV row
csvRow := map[string]string{
    "B365H": "2.40", "B365D": "3.10", "B365A": "2.90",
    "WHH":   "2.60", "WHD":   "3.30", "WHA":   "2.70",
}

homeOdds, drawOdds, awayOdds := ds.AverageOdds(csvRow)
fmt.Printf("Average odds: %.2f/%.2f/%.2f\n", homeOdds, drawOdds, awayOdds)
```

## Error Handling

The method handles several types of errors:
- Invalid league ID (unsupported leagues)
- Invalid season format
- HTTP request failures (403, 404, network issues)
- CSV parsing errors
- Team lookup failures

## Integration with Existing System

The returned Match structs are fully compatible with the existing podds system:
- Can be saved to database using existing `SaveMatches()` function
- Can be processed for team statistics using `ProcessAndSaveTeamStats()`
- Can be used for predictions using `PredictMatch()`

## Limitations

- Only supports English football leagues available on football-data.co.uk
- Requires team names to match existing team data for ID lookup
- Subject to rate limiting or access restrictions from the source website
- Historical data availability depends on what's published by football-data.co.uk

## Cache Management

Cache files are automatically managed:
- Created on first fetch for each league/season combination
- Reused on subsequent calls for the same league/season
- Automatically refreshed for current season data
- Stored in the configured `PoddsCachePath` directory

To manually clear cache, delete the relevant CSV files from the cache directory.

## Testing

Unit tests are provided in `/test/football_data_test.go`:
- `TestFetchFootballDataHistorical`: Basic functionality test
- `TestFetchFootballDataHistoricalCaching`: Cache behavior test  
- `TestFotmobLeagueIDToNative`: League ID mapping test
- `TestFotmobSeasonToNative`: Season format conversion test

Run tests with:
```bash
./test.sh
```

Note: Tests may fail if football-data.co.uk blocks automated requests (403 errors).
