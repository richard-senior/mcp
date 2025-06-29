package podds

import "fmt"

// PoddsConfig contains all configurable parameters that influence prediction outcomes
// This centralizes all magic numbers and constants for easy adjustment
type PoddsConfig struct {
	// Database and cache parameters
	PoddsAssetsPath string // The base directory of assets relating to podds
	PoddsCachePath  string // The location in which podds cached downloaded data is stored
	PoddsDbPath     string // The location of the podds sqlite database

	// === General Default vars ===
	Leagues                 []int    // the list of leagues in which we're interested (fotmob id's)
	Seasons                 []string // the list of seasons we're interested in
	CurrentSeasonFirstYear  int      // the first year of the current season
	CurrentSeasonSecondYear int      // the second year of the current season

	// === CORE PREDICTION PARAMETERS ===

	// Monte Carlo Simulation Settings
	PoissonSimulations int     // Number of Monte Carlo simulations (default: 100000)
	PoissonRange       int     // Maximum goals to consider 0-N (default: 9, so 0-8 goals)
	MaxGoalsCap        float64 // Maximum expected goals cap (default: 10.0)
	MinGoalsFloor      float64 // Minimum expected goals floor (default: 0.0)

	// === TEAM STATISTICS CALCULATION ===

	// Form vs Statistics Weighting
	FormWeight  float64 // Weight given to form in calculations (default: 0.3)
	StatsWeight float64 // Weight given to statistics (calculated as 1.0 - FormWeight)

	// Division by Zero Protection
	MakeSensibleDefault float64 // Default value when division by zero occurs (default: 1.0)

	// === DIXON-COLES CORRECTION ===

	// Dixon-Coles correlation parameter for low-scoring games
	DixonColesRho float64 // Correlation parameter (default: -0.03, range: -0.03 to -0.05)

	// === TRAVEL DISTANCE (POKE) ADJUSTMENTS ===

	// Derby Match Settings
	DerbyDistanceThreshold int     // Distance threshold for derby matches in miles (default: 10)
	DerbyBoostMultiplier   float64 // Boost multiplier for derby matches (default: 1.08 = 8% increase)

	// Travel Penalty Thresholds (miles)
	ShortTravelThreshold    int // No penalty below this distance (default: 50)
	MediumTravelThreshold   int // Medium penalty threshold (default: 100)
	LongTravelThreshold     int // Long penalty threshold (default: 200)
	VeryLongTravelThreshold int // Very long penalty threshold (default: 300)

	// Travel Penalty Multipliers (applied to away teams only)
	ShortTravelPenalty    float64 // 50-99 miles penalty (default: 0.98 = 2% reduction)
	MediumTravelPenalty   float64 // 100-199 miles penalty (default: 0.96 = 4% reduction)
	LongTravelPenalty     float64 // 200-299 miles penalty (default: 0.92 = 8% reduction)
	VeryLongTravelPenalty float64 // 300+ miles penalty (default: 0.88 = 12% reduction)

	// === OVER/UNDER GOALS THRESHOLDS ===

	Over1p5GoalsThreshold float64 // Threshold for over 1.5 goals (default: 1.5)
	Over2p5GoalsThreshold float64 // Threshold for over 2.5 goals (default: 2.5)

	// === DEFAULT LEAGUE AVERAGES ===

	// Used when no historical data is available
	DefaultHomeGoalsPerGame float64 // Default home team goals per game (default: 1.5)
	DefaultAwayGoalsPerGame float64 // Default away team goals per game (default: 1.1)

	// League Team Counts
	PremierLeagueTeams int // Premier League team count (default: 20)
	ChampionshipTeams  int // Championship team count (default: 24)
	LeagueOneTeams     int // League One team count (default: 24)
	LeagueTwoTeams     int // League Two team count (default: 24)
	DefaultLeagueTeams int // Default assumption for unknown leagues (default: 20)

	// === TIME-BASED RESTRICTIONS ===

	// Current Season Settings
	CurrentSeason        string // Current season for predictions (default: "2025/2026")
	PredictionTimeBuffer int    // Minutes before match to stop predictions (default: 15)

	// === FORM CALCULATION PARAMETERS ===

	// Form calculation uses quaternary system (0=loss, 1=loss, 2=draw, 3=win)
	// These could be made configurable if needed
	FormLossValue int // Value for losses in form calculation (default: 1)
	FormDrawValue int // Value for draws in form calculation (default: 2)
	FormWinValue  int // Value for wins in form calculation (default: 3)
}

// DefaultPoddsConfig returns the default configuration with all standard values
func DefaultPoddsConfig() *PoddsConfig {
	poddsAssetsPath := "/Users/richard/mcp/.podds/"
	config := &PoddsConfig{

		PoddsAssetsPath: poddsAssetsPath,
		PoddsCachePath:  poddsAssetsPath + "cache/",
		PoddsDbPath:     poddsAssetsPath + "podds.db",

		Leagues:                 []int{47, 48, 108, 109},
		Seasons:                 []string{"2010/2011", "2011/2012", "2012/2013", "2013/2014", "2014/2015", "2015/2016", "2016/2017", "2017/2018", "2018/2019", "2019/2020", "2020/2021", "2021/2022", "2022/2023", "2023/2024", "2024/2025", "2025/2026"},
		CurrentSeasonFirstYear:  2025,
		CurrentSeasonSecondYear: 2026,

		// === CORE PREDICTION PARAMETERS ===
		PoissonSimulations: 100000,
		PoissonRange:       9,
		MaxGoalsCap:        10.0,
		MinGoalsFloor:      0.0,

		// === TEAM STATISTICS CALCULATION ===
		FormWeight:          0.3,
		StatsWeight:         0.7, // Will be recalculated as 1.0 - FormWeight
		MakeSensibleDefault: 1.0,

		// === DIXON-COLES CORRECTION ===
		DixonColesRho: -0.03,

		// === TRAVEL DISTANCE (POKE) ADJUSTMENTS ===
		DerbyDistanceThreshold: 10,
		DerbyBoostMultiplier:   1.08,

		ShortTravelThreshold:    50,
		MediumTravelThreshold:   100,
		LongTravelThreshold:     200,
		VeryLongTravelThreshold: 300,

		ShortTravelPenalty:    0.98,
		MediumTravelPenalty:   0.96,
		LongTravelPenalty:     0.92,
		VeryLongTravelPenalty: 0.88,

		// === OVER/UNDER GOALS THRESHOLDS ===
		Over1p5GoalsThreshold: 1.5,
		Over2p5GoalsThreshold: 2.5,

		// === DEFAULT LEAGUE AVERAGES ===
		DefaultHomeGoalsPerGame: 1.5,
		DefaultAwayGoalsPerGame: 1.1,

		PremierLeagueTeams: 20,
		ChampionshipTeams:  24,
		LeagueOneTeams:     24,
		LeagueTwoTeams:     24,
		DefaultLeagueTeams: 20,

		// === TIME-BASED RESTRICTIONS ===
		CurrentSeason:        "2025/2026",
		PredictionTimeBuffer: 15,

		// === FORM CALCULATION PARAMETERS ===
		FormLossValue: 1,
		FormDrawValue: 2,
		FormWinValue:  3,
	}

	// Ensure StatsWeight is always calculated correctly
	config.StatsWeight = 1.0 - config.FormWeight

	return config
}

// Global configuration instance
var Config *PoddsConfig

// init initializes the global configuration with default values
func init() {
	Config = DefaultPoddsConfig()
}

// UpdateConfig allows updating the global configuration
func UpdateConfig(newConfig *PoddsConfig) {
	// Ensure StatsWeight is recalculated when FormWeight changes
	newConfig.StatsWeight = 1.0 - newConfig.FormWeight
	Config = newConfig
}

// GetFormWeight returns the current form weight
func GetFormWeight() float64 {
	return Config.FormWeight
}

// GetStatsWeight returns the current stats weight
func GetStatsWeight() float64 {
	return Config.StatsWeight
}

// SetFormWeight updates the form weight and recalculates stats weight
func SetFormWeight(weight float64) {
	Config.FormWeight = weight
	Config.StatsWeight = 1.0 - weight
}

// === CONFIGURATION VALIDATION ===

// ValidateConfig ensures all configuration values are within reasonable ranges
func ValidateConfig(config *PoddsConfig) error {
	if config.FormWeight < 0.0 || config.FormWeight > 1.0 {
		return fmt.Errorf("FormWeight must be between 0.0 and 1.0, got: %f", config.FormWeight)
	}

	if config.PoissonSimulations < 1000 {
		return fmt.Errorf("PoissonSimulations should be at least 1000 for accuracy, got: %d", config.PoissonSimulations)
	}

	if config.PoissonRange < 3 {
		return fmt.Errorf("PoissonRange should be at least 3 to capture realistic scores, got: %d", config.PoissonRange)
	}

	if config.DixonColesRho > 0 || config.DixonColesRho < -0.1 {
		return fmt.Errorf("DixonColesRho should be between -0.1 and 0, got: %f", config.DixonColesRho)
	}

	if config.DerbyBoostMultiplier < 1.0 || config.DerbyBoostMultiplier > 1.5 {
		return fmt.Errorf("DerbyBoostMultiplier should be between 1.0 and 1.5, got: %f", config.DerbyBoostMultiplier)
	}

	// Validate travel penalties are reductions (< 1.0)
	penalties := []float64{
		config.ShortTravelPenalty,
		config.MediumTravelPenalty,
		config.LongTravelPenalty,
		config.VeryLongTravelPenalty,
	}

	for i, penalty := range penalties {
		if penalty < 0.5 || penalty > 1.0 {
			return fmt.Errorf("Travel penalty %d should be between 0.5 and 1.0, got: %f", i, penalty)
		}
	}

	return nil
}

// === HELPER FUNCTIONS FOR EASY ACCESS ===

// GetCurrentSeason returns the current season for predictions
func GetCurrentSeason() string {
	return Config.CurrentSeason
}

// SetCurrentSeason updates the current season
func SetCurrentSeason(season string) {
	Config.CurrentSeason = season
}

// GetPredictionTimeBuffer returns the time buffer in minutes
func GetPredictionTimeBuffer() int {
	return Config.PredictionTimeBuffer
}

// GetDixonColesRho returns the Dixon-Coles correlation parameter
func GetDixonColesRho() float64 {
	return Config.DixonColesRho
}

// GetMakeSensibleDefault returns the default value for division by zero protection
func GetMakeSensibleDefault() float64 {
	return Config.MakeSensibleDefault
}
