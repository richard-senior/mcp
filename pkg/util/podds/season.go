package podds

import (
	"fmt"

	"github.com/richard-senior/mcp/pkg/util"
)

// Compile-time check to ensure Match implements Persistable interface
var _ Persistable = (*Match)(nil)

// Match represents a football match with database persistence and JSON processing annotations
type Season struct {
	Year   string `json:"year,omitempty" column:"year" dbtype:"TEXT" primary:"true" index:"true"`
	League int    `json:"league,omitempty" column:"league" dbtype:"INTEGER" primary:"true" index:"true"`
	TeamId int    `json:"teamid,omitempty" column:"teamid" dbtype:"INTEGER" primary:"true" index:"true"`
}

func ParseSeason(season any) (string, error) {
	if season == nil {
		return "", fmt.Errorf("must pass a season")
	}
	ss, err := util.GetAsString(season)
	if err != nil {
		return "", err
	}
	// determine the format of this season
	// the format we want to return is YYYY/YYYY This may already be the format
	// if it is, then we can just return it. It's also possible that the delimiter is a hyphen (-)
	// in which case we need to convert it to a slash (/)
	if len(ss) == 9 && ss[4] == '-' {
		return fmt.Sprintf("%s/%s", ss[:4], ss[5:]), nil
	} else if len(ss) == 9 && ss[4] == '/' {
		return ss, nil
	}
	// this could be a short form season of the type YYYY/YY as in 2023/24 (again delimiter may be hyphen)
	// we should return it by determining the missing prefix in the abbreviated year and adding it in
	if len(ss) == 7 && ss[4] == '-' {
		return fmt.Sprintf("20%s/%s", ss[:2], ss[3:]), nil
	} else if len(ss) == 7 && ss[4] == '/' {
		return fmt.Sprintf("20%s/%s", ss[:2], ss[3:]), nil
	}
	// this could be an encoded league/season format of the form:
	// 472324 as in leagueId=47 season=2023/2024 we should unencode it and return the season data only
	// bear in mind that the leagueID is not a fixed length (may be 47, may be 108 orn any other number etc.
	// however the season data will always be 4 digits representing two consecutive years in in the 21st century (2324-2023/2024, 2223 - 2022/2023) etc.
	// so we can just take the last 4 digits and use them as the season data
	if len(ss) > 7 {
		return fmt.Sprintf("%s/%s", ss[len(ss)-7:len(ss)-3], ss[len(ss)-3:]), nil
	}
	return "", fmt.Errorf("invalid season format: %s", ss)
}

// Given a season of the form yyyy/yyyy+1 return the first year
func GetFirstYear(season any) (int, error) {
	s, err := ParseSeason(season)
	if err != nil {
		return 0, err
	}
	// split on "/" and return the first token
	return util.GetAsInteger(s[:4])
}

// Given a season of the form yyyy/yyyy+1 return the second year
func GetSecondYear(season any) (int, error) {
	s, err := ParseSeason(season)
	if err != nil {
		return 0, err
	}
	// split on "/" and return the second token
	return util.GetAsInteger(s[5:])
}

/**
* Returns true if the given two parameters represent the same season (year/year+1)
 */
func IsSameSeason(s1 any, s2 any) (bool, error) {
	season1, err := ParseSeason(s1)
	if err != nil {
		return false, err
	}
	season2, err := ParseSeason(s2)
	if err != nil {
		return false, err
	}
	return season1 == season2, nil
}

// encodes the league/season combination
// ie premier league (47) and season 2023/2024 becomes 472324
// encoded seasons can be useful for passing information between functions etc.
func EncodeLeagueSeason(league any, season any) (string, error) {
	leagueId, err := util.GetAsInteger(league)
	if err != nil {
		return "", err
	}
	season, err = ParseSeason(season)
	if err != nil {
		return "", err
	}
	seasonYear, err := util.GetAsString(season)
	if err != nil {
		return "", err
	}
	seasonYear = seasonYear[:4] + seasonYear[5:]
	return fmt.Sprintf("%d%s", leagueId, seasonYear), nil
}
