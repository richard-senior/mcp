# Poisson Prediction Debugging Session Summary
**Date**: 2025-06-27  
**Issue**: All Poisson predictions showing 0-0 with 0% probabilities

## 🎯 ROOT CAUSE IDENTIFIED
**Team statistics exist but have zero attack/defense values**

### Key Evidence
```
[DEBUG] Found team stats for 8586 round 38 attack: 0.00 / 0.00 defense: 0.00 / 0.00
[DEBUG] Home team stats: Tottenham HomeAttack: 0.00 AwayAttack: 0.00 HomeDefense: 0.00 AwayDefense: 0.00
```

## 🔧 What We Fixed
1. **SQL Column Names**: `teamId` → `team_id`, `leagueId` → `league_id`
2. **Team Stats Retrieval**: Now gets most recent round instead of hardcoded Round 1
3. **Bulk Caching**: Uses persistable `FindWhere()` instead of direct SQL
4. **Enhanced Logging**: Added comprehensive debug output

## ✅ What We Proved Works
1. **Prediction Algorithm**: Unit test `TestAPredictionPipeline` passes with artificial data
2. **Database Queries**: Team statistics are found and retrieved correctly
3. **Data Flow**: Bulk processing and caching working properly

## ❌ What's Still Broken
**Team Statistics Calculation Logic** in `teamStats.go:calculateTeamStatsForRound()`
- Attack/defense strengths are calculated as 0.00 instead of realistic values (1.2, 0.8, etc.)
- This is the mathematical calculation problem, not data retrieval

## 🔍 Debug Commands Used
```bash
# Enable debug logging
# Change internal/logger/logger.go: NewLogger(DEBUG)

# Run filtered debug output
cd /Users/richard/mcp && timeout 60s ./test.sh 2>&1 | grep -E "(team stats|attack: 0\.00)" | head -20

# Run unit test
go test ./test -v -run TestAPredictionPipeline
```

## 📊 Next Investigation
**Focus on**: `pkg/util/podds/teamStats.go` lines 198-213
**Look for**: 
- Division by zero in attack/defense calculations
- Round average calculations returning zero
- `makeSensible()` function issues
- Default value assignments

## 🎯 Expected Fix
Team statistics should show values like:
```
[DEBUG] Found team stats for 8586 round 38 attack: 1.20 / 0.90 defense: 0.80 / 1.10
```
Instead of all zeros.

## 📁 Files Modified
- `/Users/richard/mcp/pkg/util/podds/poisson.go` - Enhanced debugging
- `/Users/richard/mcp/pkg/util/podds/datasource.go` - Bulk caching with persistable
- `/Users/richard/mcp/pkg/util/podds/match.go` - SaveMatches handles updates
- `/Users/richard/mcp/test/prediction_debug_test.go` - Unit test proving algorithm works
- `/Users/richard/mcp/.amazonq/rules/poisson-prediction-debugging-rules.md` - Session rules
