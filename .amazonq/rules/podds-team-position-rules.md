---
triggers: ["calculate","positions", "final", "initial", "football", "podds", "league", "season"]
task_type: "data_calculation"
priority: 1
---

## Prompt for Calculating team positions

Task: Calculate the starting or final positions for [LEAGUE] season [YEAR/YEAR+1]

Required Information:
• Target league: [e.g., Premier League (47), Championship (48), League One (108), League Two (109)]
• Target season: [e.g., 2025/2026]

Instructions: Follow the rules below to calculate the positions.

## Rules for FINAL positions Calculations
Final positions are those places that teams FINISHED a historical season in
Use the various sources to find the league table for the given season and league as it stood
at the end of that season and convert it to JSON format
Here is an example  step by step process:

### Step 1: Retrieve Historical League Table Data
1. Source Wikipedia page: Used the mcp___mcp___html_2_markdown tool to convert the Wikipedia page for "2010–11 Football League One" to markdown
format
2. Extract final table: Located the final league table showing all teams with their final positions, points, and promotion/relegation status

### Step 2: Identify Team ID Sources
1. Check local data file: First searched /Users/richard/mcp/pkg/util/podds/data.go using bash commands with grep patterns like:
  bash
   grep -E '"[TeamName]": \{"fotmobId": [0-9]+' /Users/richard/mcp/pkg/util/podds/data.go
  You can obviously do this in reverse to find the name of a team by id

2. Use FotMob search for missing teams: For teams not found in the local data file, used Google search with the pattern:

   "[Team Name]" fotmob.com/teams

  Then extracted the numeric team ID from URLs like https://www.fotmob.com/teams/[ID]/overview/[team-slug]

### Step 3: Map Final Positions to Team IDs
From the Wikipedia table, I extracted the final positions and matched them with the corresponding FotMob team IDs:

| Position | Team | Team ID | Source |
|----------|------|---------|---------|
| 1 | Brighton & Hove Albion | 10204 | data.go |
| 2 | Southampton | 8466 | data.go |
| 3 | Huddersfield Town | 9796 | data.go |
| 4 | Peterborough United | 8677 | data.go |
| 5 | Milton Keynes Dons | 8645 | data.go |
| 6 | Bournemouth | 8678 | data.go |
| 7 | Leyton Orient | 8351 | data.go |
| 8 | Exeter City | 9833 | data.go |
| 9 | Rochdale | 8493 | data.go |
| 10 | Colchester United | 8416 | data.go |
| 11 | Brentford | 9937 | data.go |
| 12 | Carlisle United | 10196 | data.go |
| 13 | Charlton Athletic | 8451 | data.go |
| 14 | Yeovil Town | 10198 | data.go |
| 15 | Sheffield Wednesday | 10163 | FotMob search |
| 16 | Hartlepool United | 8488 | FotMob search |
| 17 | Oldham Athletic | 9785 | FotMob search |
| 18 | Tranmere Rovers | 8313 | FotMob search |
| 19 | Notts County | 9819 | FotMob search |
| 20 | Walsall | 10006 | data.go |
| 21 | Dagenham & Redbridge | 8009 | FotMob search |
| 22 | Bristol Rovers | 10104 | data.go |
| 23 | Plymouth Argyle | 8401 | data.go |
| 24 | Swindon Town | 9795 | data.go |

### Step 4: Format Output
Created the final JSON structure in the requested compact format:
• leagueId: 108 (League One identifier)
• season: "2010/2011"
• positions: Object mapping position numbers (as strings) to team IDs (as integers)

### Step 5: Validation

#### 1. Historical Accuracy Verification
• **Source authentic league tables**: Use reliable sources (Wikipedia, official league sites, Football Wiki) to obtain the actual final table for the specified season
• **Verify season details**: Confirm champions, relegated teams, and notable season events match historical records
• **Cross-reference multiple sources**: When possible, validate information across different reliable sources

#### 2. Team Order Validation
• **Position-by-position check**: Verify each position (1-20 etc.) matches the actual final league standings
• **Points verification**: Where available, confirm the points totals align with the historical final table
• **Relegation/promotion confirmation**: Ensure relegated teams are in the bottom 3 positions and promoted teams from previous season are included

#### 3. FotMob Team ID Verification
• **Primary source check**: Search the local data.go file first for team ID mappings using grep patterns like:
 bash
  grep -E '"[TeamName]": \{"fotmobId": [0-9]+' /path/to/data.go

• **Alternative name searches**: Check for common abbreviations or alternative team names (e.g., "Man City", "QPR", "Wolves")
• **FotMob direct verification**: When IDs are missing from data.go, search FotMob directly using:

  "[Team Name]" site:fotmob.com/teams

• **Extract ID from URL**: Parse the team ID from FotMob URLs in format: fotmob.com/teams/[ID]/overview/[team-slug]

#### 4. Data Structure Validation
• **JSON format compliance**: Ensure the entry follows the correct structure:

json
  {
    "leagueId": 47,
    "season": "YYYY/YYYY+1",
    "positions": {
      "1": teamId, "2": teamId, ..., "20": teamId
    }
  }


• **League ID verification**: Confirm leagueId 47 corresponds to Premier League
• **Season format check**: Verify season follows "YYYY/YYYY+1" format (e.g., "2011/2012")
• **Complete position mapping**: Ensure all positions 1-20 are present with valid team IDs

#### 5. Consistency Checks
• **No duplicate team IDs**: Verify each team ID appears only once in the positions object
• **No missing positions**: Confirm all positions from 1-20 are included
• **Team count verification**: Ensure exactly 20 teams are present (Premier League standard)

#### 6. Historical Context Validation
• **Promoted teams inclusion**: Verify teams promoted from Championship previous season are present
• **Relegated teams confirmation**: Ensure teams relegated to Championship are in positions 18-20
• **Season narrative alignment**: Check that major season events (title races, relegation battles) align with the final positions

#### 7. Error Reporting Standards
• **Specific mismatch identification**: Report exact positions where team IDs don't match
• **Provide correct alternatives**: When errors found, supply the accurate team IDs and positions
• **Source attribution**: Reference where correct information was obtained (data.go, FotMob, Wikipedia)

#### 8. Success Confirmation
• **Explicit validation statement**: Clearly state when an entry is completely correct
• **Position-by-position confirmation**: List each position with team name and ID to show verification
• **Historical significance note**: Include relevant context about the season (champions, notable events)

These rules ensure comprehensive validation of FINAL_POSITIONS entries by combining historical accuracy, technical data
verification, and proper formatting standards.

## Rules for INITIAL_POSITIONS Calculations
The initial positions are the position in which the team logically started any given season before any matches were played.
Each year some teams (the number differs by league) are relegated from the league above and promoted from the
league below.
Generally:
- for any given league the initial positions are the 'final' positions from the previous season for that league.
  That is : if we want the initial positions of the premier league season 2014/2015 we start with the final positions of the
  premier league season 2013/2014
- Then we must replace those teams that were promoted out of the target league/season with those teams relegated from the league above
  in the previous season. No teams are relegated INTO the premier league as it is the highest league.
  So if we want the initial positions of the championship 2014/2015 we must know those teams relegated from the premier league at the end of
  2013/2014 etc.
- And we must also replace those teams that were relagated from the target season the year before,
  with those teams promoted from the league below the year before
  so for example if we wanted the initial positions of the premier league 2014/2015 we would need to know those teams promoted
  from the championship in 2013/2014
- Sometimes teams can be promoted by 'playoff'. Those teams are always considered less-worthy than those that were automatically promoted.
  You must determine how many teams are promoted or relegated from each league, below is some guidance.
For information about what the final positions were in any given league or season you can see /Users/richard/mcp/pkg/util/podds/fp.py
Or use wikipedia or google to find out if the information is not in that file.

### League-Specific Rules

#### Premier League (leagueId: 47)
• **Base**: Use positions 1-17 from previous season's FINAL table
• **Replacements**:
  • Position 18: Championship winner
  • Position 19: Championship runner-up
  • Position 20: Championship playoff winner
• **Note**: No relegation TO a higher league (Premier League is top tier)

#### Championship (leagueId: 48)
• **Base**: Use previous season's FINAL table, remove relegated teams
• **From above (Premier League)**: Add 3 relegated teams to top positions
• **From below (League One)**: Add promoted teams to bottom positions
  • Automatic promotion winners go above playoff winner

#### League One (leagueId: 108)
• **Base**: Use previous season's FINAL table, remove relegated teams
• **From above (Championship)**: Add 3 relegated teams to top positions
• **From below (League Two)**: Add promoted teams to bottom positions

#### League Two (leagueId: 109)
• **Base**: Use previous season's FINAL table, remove relegated teams
• **From above (Championship)**: Add 2 relegated teams to top positions
• **From below (League Two)**: Add promoted 2 teams (From the 'National League') to bottom positions

### Verification Steps
1. The teams that start a particular league/season must necesarilly match those that finish that league/seasons.
   So if available, get the table for the FINAL_POSITIONS for that league and season and ensure that
   all teams listed in your 'Initial Positions' are present in the final positions of the same season, and that no teams
   which are NOT present in the final positions are present in the initial positions.
   Final positions can may (or may not) be found in data.go near the initial positions.
2. Verify promotion/relegation: Cross-reference multiple sources for promoted/relegated teams
3. Check team IDs: Search data.go file to confirm correct team ID mappings. If you can't find an ID then you should ask.
4. Validate structure: Ensure JSON format matches existing entries

You may run test.sh which will invoke podds_data_test.go which will attempt to validate that the INITIAL_POSITIONS array
is correct. If you run test.sh and find errors then you should attempt to find out why by looking for correct data online

### Output Format
json
{
  "leagueId": [LEAGUE_ID],
  "season": "[YEAR/YEAR+1]",
  "positions": {
    "1": [TEAM_ID], "2": [TEAM_ID], "3": [TEAM_ID], "4": [TEAM_ID], "5": [TEAM_ID],
    "6": [TEAM_ID], "7": [TEAM_ID], "8": [TEAM_ID], "9": [TEAM_ID], "10": [TEAM_ID],
    "11": [TEAM_ID], "12": [TEAM_ID], "13": [TEAM_ID], "14": [TEAM_ID], "15": [TEAM_ID],
    "16": [TEAM_ID], "17": [TEAM_ID], "18": [TEAM_ID], "19": [TEAM_ID], "20": [TEAM_ID]
  }
}

### Common Mistakes to Avoid
1. Using starting positions instead of final positions - Always use how the previous season ENDED when calculating starting positions
2. Incorrect promotion order - Championship winner goes to position 18, runner-up to 19, playoff winner to 20 etc.
3. Wrong team IDs - Always verify team IDs by searching the existing ip.go file
4. Missing relegation/promotion - Ensure all relegated teams are removed and all promoted teams are added
