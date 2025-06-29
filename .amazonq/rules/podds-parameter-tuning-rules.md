---
triggers: ["tune","parameter", "predictions", "test", "football", "podds", "league", "season"]
task_type: "podds_parameter_tuning"
priority: 1
---

## Prompt for tuning individual parameters in the podds prediction system
The util/podds.config.go file contains many parameters that could have a bearing on the outcome of podds
predictions. Each of these parameters must be tuned in order to arrive at optimal values.
To do this we must create a unit test as detailed below.
We must create a shell script named 'test.sh' in the root of the project directory (alongside build.sh) and
invoke ONLY our new unit test within that script.
We must then invoke that script

# Creating relevant unit tests
- choose a parameter in ./util/podds/config.go to tune such as FormWeight etc.
- Create a unit test file in ./test
- Create the main test function which does the following:
  - Get a sample set of Matches from the database by invoking persistable.go#LoadExistingMatches(leagueId,season)
    Use leagueId 47 (premier league) and season 2024/2025
  - Get processed TeamStats for those matches by calling ProcessTeamStats(matches []*Match, leagueID int, season string) in teamStats.go
  - create an array of reasonable values for that chosen tuning parameter
  - iterate the array and do the following for each entry:
    - Alter the relevant value of the Config struct
    - Iterate the sample matches and call PredictMatch(match *Match, teamStats []*TeamStats) in poisson.go or
      doPredictMatch. This will add or alter some prediction data in teh passed *Match object.
    - Gather statistics on how accurate the prediction has been for that sample of matches and output it to console
