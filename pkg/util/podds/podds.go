package podds

import "fmt"

/**
* Podds is a golang library for estimating the results of EFL football matches
 */
const (
	poddsAssetsPath = "/Users/richard/mcp/.podds/"
	poddsCachePath  = poddsAssetsPath + "cache/"
	poddsDbPath     = poddsAssetsPath + "podds.db"
)

var (
	// Define leagues and seasons to load
	Leagues = []int{47, 48, 108, 109} // Premier League, Championship, League One, League Two
	// manually populate the list in case we need to weed out individual seasons
	Seasons                 = []string{"2010/2011", "2011/2012", "2012/2013", "2013/2014", "2014/2015", "2015/2016", "2016/2017", "2017/2018", "2018/2019", "2019/2020", "2020/2021", "2021/2022", "2022/2023", "2023/2024", "2024/2025", "2025/2026"}
	CurrentSeasonFirstYear  = 2025
	CurrentSeasonSecondYear = 2026
	TData                   = GetDataInstance()
)

type Podds struct {
}

func NewPodds() *Podds {
	return &Podds{}
}

// a method which causes the re-parsing of any league data which doesn't
// already exist in cache/db, and also fetches any remaining fixtures for this season (2025/2026)
// those fixtures are then re-calculated and re-predicted IF they haven't already been played or are
// and are more than one hour away from being played.

func (p *Podds) Update() error {
	ds := GetDatasourceInstance()
	if ds == nil {
		return fmt.Errorf("failed to load or init the datasource")
	}
	return nil
}
