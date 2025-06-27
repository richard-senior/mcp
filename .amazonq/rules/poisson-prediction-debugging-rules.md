---
triggers: ["prediction", "poisson", "zero", "team stats", "debugging"]
task_type: "debugging_analysis"
priority: 1
---

# Poisson Prediction Debugging Rules & Session Summary

## Critical Issue Identified: Zero Team Statistics Values

### Problem Summary
- **Symptom**: All Poisson predictions showing 0-0 scores with 0% win probabilities
- **Root Cause**: Team statistics exist in database but have zero attack/defense values
- **Impact**: Zero team stats → Zero expected goals → Zero predictions

### Debugging Session Findings

#### ✅ What Works
1. **Prediction Algorithm**: Confirmed working via unit test with artificial data
2. **Database Queries**: Team statistics are being found and retrieved correctly
3. **SQL Column Names**: Fixed teamId → team_id issue in queries
4. **Data Flow**: Bulk caching system working properly

#### ❌ Root Cause Discovered
**Team Statistics Calculation Problem:**
```
[DEBUG] Found team stats for 8586 round 38 attack: 0.00 / 0.00 defense: 0.00 / 0.00
[DEBUG] Home team stats: Tottenham HomeAttack: 0.00 AwayAttack: 0.00 HomeDefense: 0.00 AwayDefense: 0.00
```

**Evidence:**
- Team statistics exist (Round 38 = plenty of historical data)
- Team statistics are retrieved successfully
- BUT all attack/defense strength values are 0.00
- This causes: 0.00 × 0.00 × 1.5 = 0.00 expected goals

### Technical Investigation Path

#### 1. Initial Hypothesis (INCORRECT)
- Missing team statistics in database
- SQL column name mismatches (teamId vs team_id)
- Season format problems (2024/2025 vs 2024-2025)

#### 2. Debugging Tools Created
- **Unit Test**: `TestAPredictionPipeline` - Proved algorithm works with proper data
- **Bulk Cache System**: Pre-loads existing matches to avoid 30,000 individual queries
- **Enhanced Logging**: Added debug output for team stats retrieval

#### 3. Key Code Locations
- **Team Stats Calculation**: `pkg/util/podds/teamStats.go:calculateTeamStatsForRound()`
- **Attack/Defense Logic**: Lines 198-213 in teamStats.go
- **Prediction Entry Point**: `pkg/util/podds/poisson.go:PredictMatch()`
- **Team Stats Retrieval**: `pkg/util/podds/poisson.go:getTeamStats()`

### Critical Code Flow Analysis

#### Data Processing Sequence (datasource.go:Update())
1. **Line 183**: `ProcessAndSaveTeamStats(matches, leagueID, season)` - Creates team stats
2. **Lines 198-206**: `PredictMatch(match)` - Uses team stats for predictions
3. **Line 209**: `SaveMatches(matches)` - Saves predictions to database

#### Team Stats Processing Issue
**File**: `teamStats.go:calculateTeamStatsForRound()`
**Problem**: Attack/defense strength calculations producing 0.00 values
**Key Logic**: Lines around 198-213 where strengths are calculated

### Debugging Commands Used

#### Enable Debug Logging
```go
// In internal/logger/logger.go
defaultLogger = NewLogger(DEBUG)  // Change from INFO to DEBUG
```

#### Run Filtered Debug Output
```bash
cd /Users/richard/mcp && timeout 60s ./test.sh 2>&1 | grep -E "(team stats|TeamID|attack: 0\.00)" | head -20
```

#### Unit Test for Verification
```bash
go test ./test -v -run TestAPredictionPipeline
```

### Next Investigation Steps

#### 1. Examine Team Stats Calculation Logic
**Focus Areas:**
- `calculateTeamStatsForRound()` function
- Attack/defense strength formulas
- Division by zero issues
- Default value assignments

#### 2. Check Round Average Calculations
**Potential Issues:**
- League averages might be zero
- Division by makeSensible() function
- Round average data missing

#### 3. Verify Match Data Quality
**Requirements:**
- Matches must have `HasBeenPlayed() == true`
- Actual goals must be >= 0
- Match results must be realistic

### Code Fixes Applied

#### 1. SQL Column Name Fix
```go
// OLD (WRONG)
whereClause := "teamId = ? AND leagueId = ? AND season = ? ORDER BY round DESC LIMIT 1"

// NEW (CORRECT)  
whereClause := "team_id = ? AND league_id = ? AND season = ? ORDER BY round DESC LIMIT 1"
```

#### 2. Bulk Cache Implementation
```go
// Pre-load existing matches to avoid individual queries
existingMatches, err := loadExistingMatches(leagueID, season)
matches, err := datasource.extractMatchesWithCache(pageProps, existingMatches)
```

#### 3. Enhanced Debug Logging
```go
logger.Debug("Found team stats for", teamID, "round", teamStats.Round, 
    "attack:", teamStats.HomeAttackStrength, "/", teamStats.AwayAttackStrength,
    "defense:", teamStats.HomeDefenseStrength, "/", teamStats.AwayDefenseStrength)
```

### Test Cases Created

#### Unit Test: TestAPredictionPipeline
**Purpose**: Verify prediction algorithm with controlled data
**Result**: ✅ PASSED - Algorithm works with proper team statistics
**Location**: `/Users/richard/mcp/test/prediction_debug_test.go`

**Key Findings:**
- Prediction algorithm is correct
- Issue is in data quality, not calculation logic
- Test creates realistic team stats (1.2, 0.8, etc.) and gets proper predictions

### Resolution Strategy

#### Immediate Fix Required
**Target**: Team statistics calculation in `teamStats.go`
**Symptom**: All attack/defense values = 0.00
**Expected**: Values like HomeAttackStrength: 1.2, HomeDefenseStrength: 0.8

#### Investigation Priority
1. **HIGH**: Check `calculateTeamStatsForRound()` math
2. **HIGH**: Verify round average calculations  
3. **MEDIUM**: Check match data quality (HasBeenPlayed, actual goals)
4. **LOW**: Verify league average defaults

### Session Context
- **Date**: 2025-06-27
- **Primary Issue**: Zero Poisson predictions
- **Status**: Root cause identified - zero team statistics values
- **Next Step**: Fix team statistics calculation logic

### Key Learnings
1. **Unit tests are invaluable** for isolating algorithm vs data issues
2. **Debug logging at multiple levels** reveals data flow problems
3. **Bulk operations** significantly improve performance over individual queries
4. **Systematic debugging** from symptoms to root cause is essential
