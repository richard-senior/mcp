# Podds Prediction System - Tunable Parameters

This document lists all parameters in the Podds prediction system that can affect prediction outcomes, organized by category.

## 📊 Parameter Categories

### 🎯 **CORE PREDICTION PARAMETERS** (High Impact)

| Parameter | Config Path | Default | Skip | Description |
|-----------|-------------|---------|------|-------------|
| `formWeight` | Function: `SetFormWeight()` | 0.1 | ❌ | Weight given to recent form vs overall statistics |
| `poissonSimulations` | `Config.PoissonSimulations` | 100000 | ✅ | Number of Monte Carlo simulations (computationally expensive) |
| `poissonRange` | `Config.PoissonRange` | 9 | ✅ | Maximum goals to consider (0-N, affects matrix size) |
| `maxGoalsCap` | `Config.MaxGoalsCap` | 10.0 | ✅ | Maximum expected goals cap (rarely reached) |
| `minGoalsFloor` | `Config.MinGoalsFloor` | 0.0 | ✅ | Minimum expected goals floor (rarely used) |
| `makeSensibleDefault` | `Config.MakeSensibleDefault` | 1.0 | ✅ | Division by zero protection value |

### 🔧 **DIXON-COLES CORRECTION** (Medium Impact)

| Parameter | Config Path | Default | Skip | Description |
|-----------|-------------|---------|------|-------------|
| `dixonColesRho` | `Config.DixonColesRho` | -0.01 | ❌ | Correlation parameter for low-scoring games |

### 🚗 **TRAVEL DISTANCE (POKE) ADJUSTMENTS** (Variable Impact)

| Parameter | Config Path | Default | Skip | Description |
|-----------|-------------|---------|------|-------------|
| `derbyDistanceThreshold` | `Config.DerbyDistanceThreshold` | 10 | ✅ | Distance threshold for derby matches (miles) |
| `derbyBoostMultiplier` | `Config.DerbyBoostMultiplier` | 1.06 | ❌ | Boost multiplier for derby matches |
| `shortTravelThreshold` | `Config.ShortTravelThreshold` | 50 | ✅ | No penalty below this distance |
| `mediumTravelThreshold` | `Config.MediumTravelThreshold` | 100 | ✅ | Medium penalty threshold |
| `longTravelThreshold` | `Config.LongTravelThreshold` | 200 | ✅ | Long penalty threshold |
| `veryLongTravelThreshold` | `Config.VeryLongTravelThreshold` | 300 | ✅ | Very long penalty threshold |
| `shortTravelPenalty` | `Config.ShortTravelPenalty` | 0.98 | ✅ | 50-99 miles penalty (minor effect) |
| `mediumTravelPenalty` | `Config.MediumTravelPenalty` | 0.96 | ✅ | 100-199 miles penalty (moderate effect) |
| `longTravelPenalty` | `Config.LongTravelPenalty` | 0.92 | ❌ | 200-299 miles penalty (significant effect) |
| `veryLongTravelPenalty` | `Config.VeryLongTravelPenalty` | 0.88 | ✅ | 300+ miles penalty (affects few matches) |

### 📈 **OVER/UNDER GOALS THRESHOLDS** (No Win/Draw/Loss Impact)

| Parameter | Config Path | Default | Skip | Description |
|-----------|-------------|---------|------|-------------|
| `over1p5GoalsThreshold` | `Config.Over1p5GoalsThreshold` | 1.5 | ✅ | Threshold for over 1.5 goals betting |
| `over2p5GoalsThreshold` | `Config.Over2p5GoalsThreshold` | 2.5 | ✅ | Threshold for over 2.5 goals betting |

### 📊 **FORM CALCULATION PARAMETERS** (Affects Team Stats)

| Parameter | Config Path | Default | Skip | Description |
|-----------|-------------|---------|------|-------------|
| `formLossValue` | `Config.FormLossValue` | 1 | ✅ | Value for losses in form calculation |
| `formDrawValue` | `Config.FormDrawValue` | 2 | ✅ | Value for draws in form calculation |
| `formWinValue` | `Config.FormWinValue` | 3 | ✅ | Value for wins in form calculation |

## 🎛️ **Usage Instructions**

### Running Parameter Tuning

```bash
# Run the tuning system
./tune.sh
```

### Enabling/Disabling Parameters

To enable a parameter for tuning, set `Skip: false` in the `params` array:

```go
{
    Name:       "poissonSimulations",
    ConfigPath: "Config.PoissonSimulations",
    Values:     []any{50000, 75000, 100000, 125000, 150000},
    Skip:       false, // Enable this parameter
}
```

### Adding New Parameters

1. **Identify the parameter** in `config.go`
2. **Add to params array** in `tuning_test.go`:

```go
{
    Name:       "newParameter",
    ConfigPath: "Config.NewParameter", // or FunctionCall: "SetNewParameter"
    Values:     []any{value1, value2, value3},
    Skip:       false,
}
```

## 📋 **Parameter Priority Recommendations**

### 🔥 **High Priority** (Enable by default)
- `formWeight` - Major impact on predictions
- `dixonColesRho` - Affects low-scoring game predictions
- `derbyBoostMultiplier` - Significant for local rivalries
- `longTravelPenalty` - Affects many away team predictions

### 🔶 **Medium Priority** (Enable for detailed tuning)
- `poissonRange` - Affects prediction matrix size
- `derbyDistanceThreshold` - Defines what constitutes a derby
- Travel penalty thresholds - Fine-tune distance categories

### 🔸 **Low Priority** (Enable for edge case optimization)
- `poissonSimulations` - Computationally expensive, diminishing returns
- `maxGoalsCap` / `minGoalsFloor` - Rarely reached values
- Form calculation values - Affects underlying team stats

## 🎯 **Current Test Results**

Based on Premier League 2024/2025 season (380 matches):

- **Best formWeight**: 0.2 (56.84% accuracy)
- **Best dixonColesRho**: -0.02 (testing needed)
- **Best derbyBoostMultiplier**: 1.06 (testing needed)

## 🔧 **Technical Notes**

- **Reflection-based setters**: Automatically handle type conversion
- **Skip functionality**: Prevents testing computationally expensive parameters
- **Comprehensive coverage**: All prediction-affecting parameters included
- **Easy extension**: Add new parameters with minimal code changes

## 📊 **Performance Considerations**

- `poissonSimulations`: Higher values = better accuracy but slower execution
- `poissonRange`: Higher values = larger matrices but more precision
- Travel parameters: Complex interactions, test systematically
- Form parameters: Affect underlying team statistics calculation
