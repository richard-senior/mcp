package podds

import "fmt"

var (
	TData = GetDataInstance() // our data.go Data instance containing precalculated data
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
