package scorespoke

import (
	"os"
	"strconv"
)

const bitSizeFloat = 64

// getEnvAsMap returns a map of key values where the keys are passed as a parameter
// and the values are from environment variables.
func getEnvAsMap(keys []string) map[string]string {
	envVars := make(map[string]string)

	for _, key := range keys {
		envVars[key] = os.Getenv(key)
	}

	return envVars
}

// mergeMaps merges two maps, preferring the value from map2 if the value in map1 is an empty slice.
// It returns a new map containing the merged values. Both map1 and map2 should have the same length.
func mergeMaps(map1, map2 map[string]string) map[string]string {
	merged := make(map[string]string)

	for key, value := range map1 {
		if len(value) == 0 {
			merged[key] = map2[key]
		} else {
			merged[key] = value
		}
	}

	return merged
}

// convertMapStringToFloat64 converts a map of string keys and string values to
// a map of string keys and float64 values.
func convertMapStringToFloat64(inputMap map[string]string) (map[string]float64, error) {
	result := make(map[string]float64)

	for key, valueStr := range inputMap {
		valueFloat, err := strconv.ParseFloat(valueStr, bitSizeFloat)
		if err != nil {
			return nil, err
		}
		result[key] = valueFloat
	}

	return result, nil
}
