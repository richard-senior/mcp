package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/richard-senior/mcp/internal/logger"
)

/**
* Returns true if the two terms are a fuzzy match
* In this case, if the 'Levenshtein distance' is <= than 2
 */
func IsFuzzyMatch(str1, str2 string) bool {
	ld := FuzzyMatch(str1, str2)
	logger.Info("Levenshtein distance for " + str1 + " and " + str2 + " is " + string(ld))
	threshold := 2
	return ld <= threshold
}

// FuzzyMatch performs fuzzy string matching using Levenshtein distance
// Returns the minimum edit distance between str1 and the best matching substring of str2
func FuzzyMatch(str1, str2 string) int {
	// Normalize strings: lowercase and remove extra spaces
	str1 = strings.ToLower(strings.TrimSpace(str1))
	str2 = strings.ToLower(strings.TrimSpace(str2))

	// Find the shortest and longest strings
	var shorter, longer string
	if len(str1) <= len(str2) {
		shorter = str1
		longer = str2
	} else {
		shorter = str2
		longer = str1
	}

	// Try to find the best partial match by sliding the shorter string
	// across the longer string
	minDistance := math.MaxInt32

	for i := 0; i <= len(longer)-len(shorter); i++ {
		substring := longer[i : i+len(shorter)]
		distance := LevenshteinDistance(shorter, substring)
		if distance < minDistance {
			minDistance = distance
		}

		// Early exit if we find a perfect match
		if minDistance == 0 {
			break
		}
	}

	return minDistance
}

// LevenshteinDistance calculates the Levenshtein distance between two strings
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// FuzzyMatchScore returns a similarity score between 0.0 and 1.0
// where 1.0 is a perfect match and 0.0 is completely different
func FuzzyMatchScore(str1, str2 string) float64 {
	distance := FuzzyMatch(str1, str2)
	maxLen := len(str1)
	if len(str2) > maxLen {
		maxLen = len(str2)
	}

	if maxLen == 0 {
		return 1.0 // Both strings are empty
	}

	return 1.0 - (float64(distance) / float64(maxLen))
}

// GetAsString converts various types to string
// If s is a string, return it
// If s is any form of number, parse it into a string and return it
// If s is any other type, convert it to string representation
func GetAsString(s any) (string, error) {
	if s == nil {
		return "", fmt.Errorf("cannot convert nil to string")
	}

	switch v := s.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case fmt.Stringer:
		// Handle types that implement String() method
		return v.String(), nil
	default:
		// Fallback to fmt.Sprintf for other types
		return fmt.Sprintf("%v", v), nil
	}
}

// GetAsInteger converts various types to integer
// If s is an integer, return it
// If s is a string that represents an integer, convert it to an integer and return it
// If s is any other type, return an error
func GetAsInteger(s any) (int, error) {
	if s == nil {
		return 0, fmt.Errorf("cannot convert nil to integer")
	}

	switch v := s.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		// Check if it fits in int range using safe conversion
		if v > 2147483647 || v < -2147483648 {
			return 0, fmt.Errorf("int64 value %d is out of int range", v)
		}
		return int(v), nil
	case uint:
		// Check if it fits in int range
		if v > 2147483647 {
			return 0, fmt.Errorf("uint value %d is out of int range", v)
		}
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		if v > 2147483647 {
			return 0, fmt.Errorf("uint32 value %d is out of int range", v)
		}
		return int(v), nil
	case uint64:
		if v > 2147483647 {
			return 0, fmt.Errorf("uint64 value %d is out of int range", v)
		}
		return int(v), nil
	case float32:
		// Check if it's a whole number
		if v != float32(int(v)) {
			return 0, fmt.Errorf("float32 value %f is not a whole number", v)
		}
		return int(v), nil
	case float64:
		// Check if it's a whole number
		if v != float64(int(v)) {
			return 0, fmt.Errorf("float64 value %f is not a whole number", v)
		}
		return int(v), nil
	case string:
		// Try to parse the string as an integer
		result, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to integer: %w", v, err)
		}
		return result, nil
	default:
		return 0, fmt.Errorf("cannot convert type %T to integer", s)
	}
}
